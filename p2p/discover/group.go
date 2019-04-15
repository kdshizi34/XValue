// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

// Package discover implements the Node Discovery Protocol.
//
// The Node Discovery protocol provides a way to find RLPx nodes that
// can be connected to. It uses a Kademlia-like protocol to maintain a
// distributed database of the IDs and endpoints of all listening
// nodes.
package discover

import (
	"bytes"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"sync"
	"time"

	"github.com/xvalue/go-xvalue/log"
	"github.com/xvalue/go-xvalue/rlp"
)

var (
	setgroup       = 0
	Dccpdelimiter  = "dccpmsg"
	grouplist      *group
	tmpdccpmsg     = &getdccpmessage{Number: [3]byte{0, 0, 0}, Msg: ""}
	setlocaliptrue = false
	localIP        = "0.0.0.0"
	changed        = 0
)

const (
	groupnum = 3

	findgroupPacket = iota + 10 + neighborsPacket //14
	groupPacket
	DccpGroupPacket
	PeerMsgPacket
	getDccpPacket
	gotDccpPacket
)

type (
	findgroup struct {
		Target     NodeID // doesn't need to be an actual public key
		Expiration uint64
		// Ignore additional fields (for forward compatibility).
		Rest []rlp.RawValue `rlp:"tail"`
	}

	group struct {
		sync.Mutex
		gname      []string
		msg        string
		count      int
		Nodes      []rpcNode
		Expiration uint64
		// Ignore additional fields (for forward compatibility).
		Rest []rlp.RawValue `rlp:"tail"`
	}

	groupmessage struct {
		sync.Mutex
		gname      []string
		msg        string
		count      int
		Nodes      []rpcNode
		Expiration uint64
		// Ignore additional fields (for forward compatibility).
		Rest []rlp.RawValue `rlp:"tail"`
	}

	message struct {
		//sync.Mutex
		Msg        string
		Expiration uint64
	}

	getdccpmessage struct {
		//sync.Mutex
		Number     [3]byte
		Target     NodeID // doesn't need to be an actual public key
		Msg        string
		Expiration uint64
	}

	dccpmessage struct {
		//sync.Mutex
		Target     NodeID // doesn't need to be an actual public key
		Msg        string
		Expiration uint64
	}
)

func (req *findgroup) name() string { return "FINDGROUP/v4" }
func (req *group) name() string     { return "GROUP/v4" }

// findgroup sends a findgroup request to the bootnode and waits until
// the node has sent up to a group.
func (t *udp) findgroup(toid NodeID, toaddr *net.UDPAddr, target NodeID) ([]*Node, error) {
	log.Debug("====  (t *udp) findgroup()  ====")
	nodes := make([]*Node, 0, bucketSize)
	nreceived := 0
	errc := t.pending(toid, groupPacket, func(r interface{}) bool {
		reply := r.(*group)
		for _, rn := range reply.Nodes {
			nreceived++
			n, err := t.nodeFromRPC(toaddr, rn)
			if err != nil {
				log.Trace("Invalid neighbor node received", "ip", rn.IP, "addr", toaddr, "err", err)
				continue
			}
			nodes = append(nodes, n)
		}
		log.Debug("findgroup", "return nodes", nodes)
		return nreceived >= groupnum
	})
	log.Debug("\nfindgroup, t.send\n")
	t.send(toaddr, findgroupPacket, &findgroup{
		Target:     target,
		Expiration: uint64(time.Now().Add(expiration).Unix()),
	})
	err := <-errc
	return nodes, err
}

func (req *findgroup) handle(t *udp, from *net.UDPAddr, fromID NodeID, mac []byte) error {
	log.Debug("====  (req *findgroup) handle()  ====")
	if expired(req.Expiration) {
		return errExpired
	}
	if !t.db.hasBond(fromID) {
		// No bond exists, we don't process the packet. This prevents
		// an attack vector where the discovery protocol could be used
		// to amplify traffic in a DDOS attack. A malicious actor
		// would send a findnode request with the IP address and UDP
		// port of the target as the source address. The recipient of
		// the findnode packet would then send a neighbors packet
		// (which is a much bigger packet than findnode) to the victim.
		return errUnknownNode
	}
	if p := getGroupInfo(); p != nil {
		t.send(from, groupPacket, p)
	}
	return nil
}

func (req *group) handle(t *udp, from *net.UDPAddr, fromID NodeID, mac []byte) error {
	log.Debug("====  (req *group) handle()  ====")
	log.Debug("group handle", "group handle: ", req)
	if expired(req.Expiration) {
		return errExpired
	}
	if !t.handleReply(fromID, groupPacket, req) {
		return errUnsolicitedReply
	}
	return nil
}

func (req *getdccpmessage) name() string { return "GETDCCPMSG/v4" }
func (req *dccpmessage) name() string    { return "DCCPMSG/v4" }

var number [3]byte

// sendgroup sends to group dccp and waits until
// the node has reply.
func (t *udp) sendToGroupDCCP(toid NodeID, toaddr *net.UDPAddr, msg string) (string, error) {
	log.Debug("====  (t *udp) sendToGroupDCCP()  ====\n")
	err := errors.New("")
	retmsg := ""
	number[0]++
	log.Debug("sendToGroupDCCP", "send toaddr: ", toaddr)
	if len(msg) <= 800 {
		number[1] = 1
		number[2] = 1
		_, err = t.send(toaddr, getDccpPacket, &getdccpmessage{
			Number:     number,
			Msg:        msg,
			Expiration: uint64(time.Now().Add(expiration).Unix()),
		})
		log.Debug("dccp", "number = ", number, "msg(<800) = ", msg)
	} else if len(msg) > 800 && len(msg) < 1600 {
		number[1] = 1
		number[2] = 2
		t.send(toaddr, getDccpPacket, &getdccpmessage{
			Number:     number,
			Msg:        msg[0:800],
			Expiration: uint64(time.Now().Add(expiration).Unix()),
		})
		log.Debug("send", "msg(> 800):", msg)
		number[1] = 2
		number[2] = 2
		_, err = t.send(toaddr, getDccpPacket, &getdccpmessage{
			Number:     number,
			Msg:        msg[800:],
			Expiration: uint64(time.Now().Add(expiration).Unix()),
		})
	} else {
		log.Error("send, msg size > 1600, sent failed.\n")
		return "", nil
	}
	//errc := t.pending(toid, gotDccpPacket, func(r interface{}) bool {
	//	fmt.Printf("dccp, gotDccpPacket: %+v\n", r)
	//	retmsg = r.(*dccpmessage).Msg
	//	return true
	//})
	//err := <-errc
	//fmt.Printf("dccp, retmsg: %+v\n", retmsg)
	return retmsg, err
}

func (req *getdccpmessage) handle(t *udp, from *net.UDPAddr, fromID NodeID, mac []byte) error {
	log.Debug("====  (req *getdccpmessage) handle()  ====")
	if expired(req.Expiration) {
		return errExpired
	}
	if !t.db.hasBond(fromID) {
		// No bond exists, we don't process the packet. This prevents
		// an attack vector where the discovery protocol could be used
		// to amplify traffic in a DDOS attack. A malicious actor
		// would send a findnode request with the IP address and UDP
		// port of the target as the source address. The recipient of
		// the findnode packet would then send a neighbors packet
		// (which is a much bigger packet than findnode) to the victim.
		return errUnknownNode
	}
	msgp := req.Msg
	num := req.Number
	log.Debug("dccp handle", "req.Number", num)
	if num[2] > 1 {
		if tmpdccpmsg.Number[0] == 0 || num[0] != tmpdccpmsg.Number[0] {
			tmpdccpmsg = &(*req)
			log.Debug("dccp handle", "tmpdccpmsg = ", tmpdccpmsg)
			return nil
		}
		log.Debug("dccp handle", "tmpdccpmsg.Number = ", tmpdccpmsg.Number)
		if tmpdccpmsg.Number[1] == num[1] {
			return nil
		}
		var buffer bytes.Buffer
		if tmpdccpmsg.Number[1] < num[1] {
			buffer.WriteString(tmpdccpmsg.Msg)
			buffer.WriteString(req.Msg)
		} else {
			buffer.WriteString(req.Msg)
			buffer.WriteString(tmpdccpmsg.Msg)
		}
		msgp = buffer.String()
	}

	go func() {
		log.Debug("getdccpmessage", "calldccpEvent msg: ", msgp)
		msgc := calldccpEvent(msgp)
		log.Debug("getdccpmessage", "calldccpEvent retmsg: ", msgc)
		msg := <-msgc
		//tmpdccpmsg.Number = [3]byte{}
		//t.send(from, gotDccpPacket, &getdccpmessage{
		log.Debug("getdccpmessage", "send(from: ", from, "msg = ", msg)
		t.send(from, gotDccpPacket, &dccpmessage{
			Target:     fromID,
			Msg:        msg,
			Expiration: uint64(time.Now().Add(expiration).Unix()),
		})
		log.Debug("dccp handle", "send to from: ", from, ", message: ", msg)
	}()
	return nil
}

func (req *dccpmessage) handle(t *udp, from *net.UDPAddr, fromID NodeID, mac []byte) error {
	log.Debug("====  (req *dccpmessage) handle()  ====\n")
	log.Debug("dccpmessage", "handle, req: ", req)
	//if expired(req.Expiration) {
	//        return errExpired
	//}
	//if !t.handleReply(fromID, gotDccpPacket, req) {
	//	return errUnsolicitedReply
	//}
	log.Debug("dccpmessage", "handle, calldccpReturn req.Msg", req.Msg)
	go calldccpReturn(req.Msg)
	return nil
}

func getGroupInfo() *group {
	if setgroup == 1 && grouplist.count == groupnum {
		grouplist.Lock()
		defer grouplist.Unlock()
		p := grouplist
		p.Expiration = uint64(time.Now().Add(expiration).Unix())
		return p
	}
	return nil
}

func InitGroup() error {
	log.Debug("==== InitGroup() ====")
	setgroup = 1
	grouplist = &group{msg: "xvc", count: 0, Expiration: ^uint64(0)}
	return nil
}

func SendToDccpGroup(msg string) string {
	log.Debug("==== SendToGroup() ====")
	bn := Table4group.nursery[0]
	if bn == nil {
		log.Debug("SendToGroup(), bootnode is nil\n")
		return ""
	}
	ipa := &net.UDPAddr{IP: bn.IP, Port: int(bn.UDP)}
	g := GetGroup(bn.ID, ipa, bn.ID)
	if g == nil || len(g) != groupnum {
		log.Debug("SendToGroup(), group is nil\n")
		return ""
	}
	var sent [groupnum + 1]int
	ret := ""
	for i := 1; i <= groupnum; {
		r := rand.Intn(groupnum)
		j := 1
		for ; j < i; j++ {
			if r+1 == sent[j] {
				break
			}
		}
		if j < i {
			continue
		}
		sent[i] = r + 1
		i += 1
		log.Debug("sendToDccpGroup", "group[", r, "]", g[r])
		n := g[r]
		ipa = &net.UDPAddr{IP: n.IP, Port: int(n.UDP)}
		err := Table4group.net.ping(n.ID, ipa)
		if err != nil {
			log.Debug("sendToDccpGroup, err", "group[", r, "]", g[r])
			continue
		}
		ret, err = Table4group.net.sendToGroupDCCP(n.ID, ipa, msg)
		break
	}
	return ret
}

func GetGroup(id NodeID, addr *net.UDPAddr, target NodeID) []*Node {
	log.Debug("==== GetGroup() ====")
	g, _ := Table4group.net.findgroup(id, addr, target)
	log.Debug("tab.net.findgroup: %+v", g)
	return g
}

func setGroup(n *Node, replace string) {
	//log.Debug("====================== SetGroup() ====================")
	if setgroup == 0 || changed == 2 {
		return
	}
	grouplist.Lock()
	defer grouplist.Unlock()
	//fmt.Printf("node: %+v, tabal.self: %+v\n", n, Table4group.self)
	//if n.ID == Table4group.self.ID {
	//	return
	//}
	if replace == "add" {
		log.Debug("group add")
		if grouplist.count >= groupnum {
			grouplist.count = groupnum
			return
		}
		log.Debug("connect", "NodeID", n.ID.String())
		//if n.ID.String() == "ead5708649f3fb10343a61249ea8509b3d700f1f51270f13ecf889cdf8dafce5e7eb649df3ee872fb027b5a136e17de73965ec34c46ea8a5553b3e3150a0bf8d" ||
		//	n.ID.String() == "bd6e097bb40944bce309f6348fe4d56ee46edbdf128cc75517df3cc586755737733c722d3279a3f37d000e26b5348c9ec9af7f5b83122d4cfd8c9ad836a0e1ee" ||
		//	n.ID.String() == "1520992e0053bbb92179e7683b3637ea0d43bb2cd3694a94a1e90e909108421c2ce22e0abdb0a335efdd8e6391eb08ba967f641b42e4ebde39997c8ad000e8c8" {
		//grouplist.gname = append(grouplist.gname, "dddddddddd")
		grouplist.Nodes = append(grouplist.Nodes, nodeToRPC(n))
		grouplist.count++
		if changed == 0 {
			changed = 1
		}
		log.Debug("group(add)", "node", n)
		log.Debug("group", "grouplist", grouplist)
		//}
	} else if replace == "remove" {
		log.Debug("group remove")
		if grouplist.count <= 0 {
			grouplist.count = 0
			return
		}
		log.Debug("connect", "NodeID", n.ID.String())
		for i := 0; i < grouplist.count; i++ {
			if grouplist.Nodes[i].ID == n.ID {
				grouplist.Nodes = append(grouplist.Nodes[:i], grouplist.Nodes[i+1:]...)
				grouplist.count--
				if changed == 0 {
					changed = 1
				}
				log.Debug("group(remove)", "node", n)
				log.Debug("group", "grouplist", grouplist)
				break
			}
		}
	}
	if grouplist.count == groupnum && changed == 1 {
		count := 0
		enode := ""
		for i := 0; i < grouplist.count; i++ {
			count++
			node := grouplist.Nodes[i]
			if enode != "" {
				enode += Dccpdelimiter
			}
			e := fmt.Sprintf("enode://%v@%v:%v", node.ID, node.IP, node.UDP)
			enode += e
			ipa := &net.UDPAddr{IP: node.IP, Port: int(node.UDP)}
			go SendToPeer(node.ID, ipa, "")
			//TODO get and send privatekey slice
			//go SendMsgToNode(node.ID, ipa, "0xff00ff")
		}
		enodes := fmt.Sprintf("%v,%v", count, enode)
		log.Debug("send group to nodes", "group: ", enodes)
		go callPrivKeyEvent(enodes)
		changed = 2
	}
	return
}

//send group info
func SendMsgToNode(toid NodeID, toaddr *net.UDPAddr, msg string) error {
	log.Debug("==== discover.SendMsgToNode() ====\n")
	log.Debug("toid: %#v, toaddr: %#v, msg: %#v\n", toid, toaddr, msg)
	if msg == "" {
		return nil
	}
	return Table4group.net.sendMsgToPeer(toid, toaddr, msg)
}

func SendToPeer(toid NodeID, toaddr *net.UDPAddr, msg string) error {
	log.Debug("==== SendToPeer() ====\n")
	log.Debug("msg: %v\n", msg)
	return Table4group.net.sendToPeer(toid, toaddr, msg)
}
func (t *udp) sendToPeer(toid NodeID, toaddr *net.UDPAddr, msg string) error {
	log.Debug("====  (t *udp) sendToPeer()  ====")
	req := getGroupInfo()
	if req == nil {
		return nil
	}
	errc := t.pending(toid, DccpGroupPacket, func(r interface{}) bool {
		return true
	})
	t.send(toaddr, DccpGroupPacket, req)
	err := <-errc
	return err
}
func (req *groupmessage) name() string { return "GROUPMSG/v4" }
func (req *groupmessage) handle(t *udp, from *net.UDPAddr, fromID NodeID, mac []byte) error {
	log.Debug("\n\n====  (req *groupmessage) handle()  ====")
	if expired(req.Expiration) {
		return errExpired
	}
	log.Debug("req: %+v\n", req)
	nodes := make([]*Node, 0, bucketSize)
	for _, rn := range req.Nodes {
		n, err := t.nodeFromRPC(from, rn)
		if err != nil {
			log.Trace("Invalid neighbor node received", "ip", rn.IP, "addr", from, "err", err)
			continue
		}
		nodes = append(nodes, n)
	}

	log.Debug("group msg handle", "req.Nodes: ", nodes)
	go callGroupEvent(nodes)
	return nil
}

//send msg
func (t *udp) sendMsgToPeer(toid NodeID, toaddr *net.UDPAddr, msg string) error {
	log.Debug("====  (t *udp) sendMsgToPeer()  ====")
	//TODO
	errc := t.pending(toid, PeerMsgPacket, func(r interface{}) bool {
		return true
	})
	t.send(toaddr, PeerMsgPacket, &message{
		Msg:        msg,
		Expiration: uint64(time.Now().Add(expiration).Unix()),
	})
	err := <-errc
	return err
}
func (req *message) name() string { return "MESSAGE/v4" }

func (req *message) handle(t *udp, from *net.UDPAddr, fromID NodeID, mac []byte) error {
	log.Debug("\n\n====  (req *message) handle()  ====")
	log.Debug("req: %#v\n", req)
	if expired(req.Expiration) {
		return errExpired
	}
	go callMsgEvent(req.Msg)
	return nil
}

var groupcallback func(interface{})

func RegisterGroupCallback(callbackfunc func(interface{})) {
	groupcallback = callbackfunc
}

func callGroupEvent(n []*Node) {
	groupcallback(n)
}

var msgcallback func(interface{})

func RegistermsgCallback(callbackfunc func(interface{})) {
	msgcallback = callbackfunc
}

func callMsgEvent(msg string) {
	msgcallback(msg)
}

//peer(of DCCP group) receive other peer msg to run dccp
var dccpcallback func(interface{}) <-chan string

func RegisterDccpCallback(callbackfunc func(interface{}) <-chan string) {
	dccpcallback = callbackfunc
}
func calldccpEvent(e interface{}) <-chan string {
	return dccpcallback(e)
}

//return
var dccpretcallback func(interface{})

func RegisterDccpRetCallback(callbackfunc func(interface{})) {
	dccpretcallback = callbackfunc
}
func calldccpReturn(e interface{}) {
	log.Debug("calldccpReturn", "args", e)
	dccpretcallback(e)
}

//get private Key
var privatecallback func(interface{})

func RegisterSendCallback(callbackfunc func(interface{})) {
	privatecallback = callbackfunc
}

func callPrivKeyEvent(e string) {
	privatecallback(e)
}

func ParseNodes(n []*Node) (int, string) {
	i := 0
	enode := ""
	for _, e := range n {
		if enode != "" {
			enode += Dccpdelimiter
		}
		i++
		enode += e.String()
	}
	return i, enode
}

func setLocalIP(data interface{}) {
	if setlocaliptrue == true {
		return
	}
	localIP = data.(*pong).To.IP.String()
	setlocaliptrue = true
}

func GetLocalIP() string {
	return localIP
}

func GetLocalID() NodeID {
	return Table4group.self.ID
}
