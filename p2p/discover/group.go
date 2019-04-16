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
	XvcDelimiter   = "xvcmsg"
	Dccp_groupList *group
	Xp_groupList   *group
	tmpdccpmsg     = &getdccpmessage{Number: [3]byte{0, 0, 0}, Msg: ""}
	setlocaliptrue = false
	localIP        = "0.0.0.0"
	changed        = 0
	Xp_changed     = 0
)

const (
	Dccp_groupMemNum = 3
	Xp_grouMemNum = 3
)

const (
	Dccprotocol_type = iota + 1
	Xprotocol_type

	Dccp_findGroupPacket = iota + 10 + neighborsPacket //14
	Xp_findGroupPacket
	Dccp_groupPacket
	Xp_groupPacket
	Dccp_groupInfoPacket
	PeerMsgPacket
	getDccpPacket
	Xp_getXvcPacket
	getXpPacket
	gotDccpPacket
	gotXpPacket
)

type (
	findgroup struct {
		P2pType    byte
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
		P2pType    byte
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
		P2pType    byte
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
		P2pType    byte
		Target     NodeID // doesn't need to be an actual public key
		Msg        string
		Expiration uint64
	}

	dccpmessage struct {
		//sync.Mutex
		Target     NodeID // doesn't need to be an actual public key
		P2pType    byte
		Msg        string
		Expiration uint64
	}
)

func (req *findgroup) name() string { return "FINDGROUP/v4" }
func (req *group) name() string     { return "GROUP/v4" }

func getGroupList(p2pType int) *group {
	switch (p2pType) {
	case Dccprotocol_type:
		return Dccp_groupList
	case Xprotocol_type:
		return Xp_groupList
	}
	return nil
}

func getGroupChange(p2pType int) *int {
	switch (p2pType) {
	case Dccprotocol_type:
		return &changed
	case Xprotocol_type:
		return &Xp_changed
	}
	return nil
}

func getXvcPacket(p2pType int) int {
	switch (p2pType) {
	case Dccprotocol_type:
		return getDccpPacket
	case Xprotocol_type:
		return Xp_getXvcPacket
	}
	return 0
}
func getGroupPacket(p2pType int) int {
	switch (p2pType) {
	case Dccprotocol_type:
		return Dccp_groupPacket
	case Xprotocol_type:
		return Xp_groupPacket
	}
	return 0
}

func getFindGroupPacket(p2pType int) int {
	switch (p2pType) {
	case Dccprotocol_type:
		return Dccp_findGroupPacket
	case Xprotocol_type:
		return Xp_findGroupPacket
	}
	return 0
}

func getGroupMemNum(p2pType int) int {
	switch (p2pType) {
	case Dccprotocol_type:
		return Dccp_groupMemNum
	case Xprotocol_type:
		return Xp_grouMemNum
	}
	return 0
}

func getGotPacket(p2pType int) int {
	switch (p2pType) {
	case Dccprotocol_type:
		return gotDccpPacket
	case Xprotocol_type:
		return gotXpPacket
	}
	return 0
}

// findgroup sends a findgroup request to the bootnode and waits until
// the node has sent up to a group.
func (t *udp) findgroup(toid NodeID, toaddr *net.UDPAddr, target NodeID, p2pType int) ([]*Node, error) {
	log.Debug("====  (t *udp) findgroup()  ====")
	nodes := make([]*Node, 0, bucketSize)
	nreceived := 0
	groupPacket := getGroupPacket(p2pType)
	findgroupPacket := getFindGroupPacket(p2pType)
	groupMemNum := getGroupMemNum(p2pType)

	errc := t.pending(toid, byte(groupPacket), func(r interface{}) bool {
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
		return nreceived >= groupMemNum
	})
	log.Debug("\nfindgroup, t.send\n")
	t.send(toaddr, byte(findgroupPacket), &findgroup{
		P2pType:    byte(p2pType),
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
	groupPacket := getGroupPacket(int(req.P2pType))
	if p := getGroupInfo(int(req.P2pType)); p != nil {
		t.send(from, byte(groupPacket), p)
	}
	return nil
}

func (req *group) handle(t *udp, from *net.UDPAddr, fromID NodeID, mac []byte) error {
	log.Debug("====  (req *group) handle()  ====")
	log.Debug("group handle", "group handle: ", req)
	if expired(req.Expiration) {
		return errExpired
	}
	groupPacket := getGroupPacket(int(req.P2pType))
	if !t.handleReply(fromID, byte(groupPacket), req) {
		return errUnsolicitedReply
	}
	return nil
}

func (req *getdccpmessage) name() string { return "GETDCCPMSG/v4" }
func (req *dccpmessage) name() string    { return "DCCPMSG/v4" }

var number [3]byte

// sendgroup sends to group dccp and waits til
// the node has reply.
func (t *udp) sendToGroupXvc(toid NodeID, toaddr *net.UDPAddr, msg string, p2pType int) (string, error) {
	log.Debug("====  (t *udp) sendToGroupXvc()  ====\n")
	err := errors.New("")
	retmsg := ""
	getxvcPacket := getXvcPacket(p2pType)
	number[0]++
	log.Debug("sendToGroupXvc", "send toaddr: ", toaddr)
	if len(msg) <= 800 {
		number[1] = 1
		number[2] = 1
		_, err = t.send(toaddr, byte(getxvcPacket), &getdccpmessage{
			Number:     number,
			P2pType:    byte(p2pType),
			Msg:        msg,
			Expiration: uint64(time.Now().Add(expiration).Unix()),
		})
		log.Debug("dccp", "number = ", number, "msg(<800) = ", msg)
	} else if len(msg) > 800 && len(msg) < 1600 {
		number[1] = 1
		number[2] = 2
		t.send(toaddr, byte(getxvcPacket), &getdccpmessage{
			Number:     number,
			P2pType:    byte(p2pType),
			Msg:        msg[0:800],
			Expiration: uint64(time.Now().Add(expiration).Unix()),
		})
		log.Debug("send", "msg(> 800):", msg)
		number[1] = 2
		number[2] = 2
		_, err = t.send(toaddr, byte(getxvcPacket), &getdccpmessage{
			Number:     number,
			P2pType:    byte(p2pType),
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
		log.Debug("getmessage", "callEvent msg: ", msgp)
		msgc := callMsgEvent(msgp, int(req.P2pType))
		log.Debug("getmessage", "callEvent retmsg: ", msgc)
		msg := <-msgc
		log.Debug("getmessage", "send(from: ", from, "msg = ", msg)
		gotpacket := getGotPacket(int(req.P2pType))
		t.send(from, byte(gotpacket), &dccpmessage{
			Target:     fromID,
			P2pType:    req.P2pType,
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
	log.Debug("dccpmessage", "handle, calldReturn req.Msg", req.Msg)
	go callXvcReturn(req.Msg, int(req.P2pType))
	return nil
}

func getGroupInfo(p2pType int) *group {
	log.Debug("getGroupInfo", "p2pType", p2pType)
	groupMemNum := getGroupMemNum(p2pType)
	groupList := getGroupList(p2pType)
	if setgroup == 1 && groupList.count == groupMemNum {
		groupList.Lock()
		defer groupList.Unlock()
		p := groupList
		p.P2pType = byte(p2pType)
		p.Expiration = uint64(time.Now().Add(expiration).Unix())
		return p
	}
	log.Warn("getGroupInfo nil")
	return nil
}

func InitGroup() error {
	log.Debug("==== InitGroup() ====")
	setgroup = 1
	Dccp_groupList = &group{msg: "xvc", count: 0, Expiration: ^uint64(0)}
	Xp_groupList = &group{msg: "xvc", count: 0, Expiration: ^uint64(0)}
	return nil
}

func SendToGroup(msg string, p2pType int) string {
	log.Debug("==== SendToGroup() ====")
	bn := Table4group.nursery[0]
	if bn == nil {
		log.Warn("SendToGroup(), bootnode is nil\n")
		return ""
	}
	ipa := &net.UDPAddr{IP: bn.IP, Port: int(bn.UDP)}
	g := GetGroup(bn.ID, ipa, bn.ID, p2pType)
	groupMemNum := getGroupMemNum(p2pType)
	if g == nil || len(g) != groupMemNum {
		log.Warn("SendToGroup(), group is nil\n")
		return ""
	}
	sent := make ([]int, groupMemNum + 1)
	ret := ""
	for i := 1; i <= groupMemNum; {
		r := rand.Intn(groupMemNum)
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
		log.Debug("sendToXvcGroup", "group[", r, "]", g[r])
		n := g[r]
		ipa = &net.UDPAddr{IP: n.IP, Port: int(n.UDP)}
		err := Table4group.net.ping(n.ID, ipa)
		if err != nil {
			log.Debug("sendToDccpGroup, err", "group[", r, "]", g[r])
			continue
		}
		ret, err = Table4group.net.sendToGroupXvc(n.ID, ipa, msg, p2pType)
		break
	}
	return ret
}

func GetGroup(id NodeID, addr *net.UDPAddr, target NodeID, p2pType int) []*Node {
	log.Debug("==== GetGroup() ====")
	g, _ := Table4group.net.findgroup(id, addr, target, p2pType)
	log.Debug("tab.net.findgroup: %+v", g)
	return g
}

func setGroup(n *Node, replace string) {
	setGroupXvc(n, replace, Dccprotocol_type)
	setGroupXvc(n, replace, Xprotocol_type)
}

func setGroupXvc(n *Node, replace string, p2pType int) {
	groupList := getGroupList(p2pType)
	groupChanged := getGroupChange(p2pType)
	groupMemNum := getGroupMemNum(p2pType)
	if setgroup == 0 || *groupChanged == 2 {
		return
	}
	groupList.Lock()
	defer groupList.Unlock()
	//fmt.Printf("node: %+v, tabal.self: %+v\n", n, Table4group.self)
	//if n.ID == Table4group.self.ID {
	//	return
	//}
	if replace == "add" {
		log.Debug("group add")
		if groupList.count >= groupMemNum {
			groupList.count = groupMemNum
			return
		}
		log.Debug("connect", "NodeID", n.ID.String())
		//if n.ID.String() == "ead5708649f3fb10343a61249ea8509b3d700f1f51270f13ecf889cdf8dafce5e7eb649df3ee872fb027b5a136e17de73965ec34c46ea8a5553b3e3150a0bf8d" ||
		//	n.ID.String() == "bd6e097bb40944bce309f6348fe4d56ee46edbdf128cc75517df3cc586755737733c722d3279a3f37d000e26b5348c9ec9af7f5b83122d4cfd8c9ad836a0e1ee" ||
		//	n.ID.String() == "1520992e0053bbb92179e7683b3637ea0d43bb2cd3694a94a1e90e909108421c2ce22e0abdb0a335efdd8e6391eb08ba967f641b42e4ebde39997c8ad000e8c8" {
		//groupList.gname = append(groupList.gname, "dddddddddd")
		groupList.Nodes = append(groupList.Nodes, nodeToRPC(n))
		groupList.count++
		if *groupChanged == 0 {
			*groupChanged = 1
		}
		log.Debug("group(add)", "node", n)
		log.Debug("group", "groupList", groupList)
		//}
	} else if replace == "remove" {
		log.Debug("group remove")
		if groupList.count <= 0 {
			groupList.count = 0
			return
		}
		log.Debug("connect", "NodeID", n.ID.String())
		for i := 0; i < groupList.count; i++ {
			if groupList.Nodes[i].ID == n.ID {
				groupList.Nodes = append(groupList.Nodes[:i], groupList.Nodes[i+1:]...)
				groupList.count--
				if *groupChanged == 0 {
					*groupChanged = 1
				}
				log.Debug("group(remove)", "node", n)
				log.Debug("group", "groupList", groupList)
				break
			}
		}
	}
	if groupList.count == groupMemNum && *groupChanged == 1 {
		count := 0
		enode := ""
		for i := 0; i < groupList.count; i++ {
			count++
			node := groupList.Nodes[i]
			if enode != "" {
				enode += XvcDelimiter
			}
			e := fmt.Sprintf("enode://%v@%v:%v", node.ID, node.IP, node.UDP)
			enode += e
			ipa := &net.UDPAddr{IP: node.IP, Port: int(node.UDP)}
			go SendToPeer(node.ID, ipa, "", p2pType)
			//TODO get and send privatekey slice
			//go SendMsgToNode(node.ID, ipa, "0xff00ff")
		}
		enodes := fmt.Sprintf("%v,%v", count, enode)
		log.Info("send group to nodes", "group", p2pType, "enodes", enodes)
		//go callPrivKeyEvent(enodes) //TODO
		*groupChanged = 2
	}
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

func SendToPeer(toid NodeID, toaddr *net.UDPAddr, msg string, p2pType int) error {
	log.Debug("==== SendToPeer() ====\n")
	log.Debug("msg: %v\n", msg)
	return Table4group.net.sendToPeer(toid, toaddr, msg, p2pType)
}
func (t *udp) sendToPeer(toid NodeID, toaddr *net.UDPAddr, msg string, p2pType int) error {
	log.Debug("====  (t *udp) sendToPeer()  ====")
	req := getGroupInfo(p2pType)
	if req == nil {
		return nil
	}
	errc := t.pending(toid, byte(Dccp_groupInfoPacket), func(r interface{}) bool {
		return true
	})
	t.send(toaddr, byte(Dccp_groupInfoPacket), req)
	err := <-errc
	return err
}
func (req *groupmessage) name() string { return "GROUPMSG/v4" }
func (req *groupmessage) handle(t *udp, from *net.UDPAddr, fromID NodeID, mac []byte) error {
	log.Debug("\n\n====  (req *groupmessage) handle()  ====")
	if expired(req.Expiration) {
		return errExpired
	}
	log.Debug("groupmessage", "req", req)
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
	go callGroupEvent(nodes, int(req.P2pType))
	return nil
}

//send msg
func (t *udp) sendMsgToPeer(toid NodeID, toaddr *net.UDPAddr, msg string) error {
	log.Debug("====  (t *udp) sendMsgToPeer()  ====")
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
	go callPriKeyEvent(req.Msg)
	return nil
}

var groupcallback func(interface{}, int)

func RegisterGroupCallback(callbackfunc func(interface{}, int)) {
	groupcallback = callbackfunc
}

func callGroupEvent(n []*Node, p2pType int) {
	groupcallback(n, p2pType)
}

var prikeycallback func(interface{})

func RegisterPriKeyCallback(callbackfunc func(interface{})) {
	prikeycallback = callbackfunc
}

func callPriKeyEvent(msg string) {
	prikeycallback(msg)
}

func callMsgEvent(e interface{}, p2pType int) <-chan string {
	switch (p2pType) {
	case Dccprotocol_type:
		return dccpcallback(e)
	case Xprotocol_type:
		return xpcallback(e)
	}
	ch := make(chan string)
	ch <- "p2pType invalid"
	return ch
}



//peer(of DCCP group) receive other peer msg to run dccp
var dccpcallback func(interface{}) <-chan string

func RegisterDccpMsgCallback(callbackfunc func(interface{}) <-chan string) {
	dccpcallback = callbackfunc
}
func calldccpEvent(e interface{}) <-chan string {
	return dccpcallback(e)
}

//return
var dccpretcallback func(interface{})

func RegisterDccpMsgRetCallback(callbackfunc func(interface{})) {
	dccpretcallback = callbackfunc
}
func calldccpReturn(e interface{}) {
	dccpretcallback(e)
}

//peer(of Xp group) receive other peer msg to run dccp
var xpcallback func(interface{}) <-chan string

func RegisterXpMsgCallback(callbackfunc func(interface{}) <-chan string) {
	xpcallback = callbackfunc
}
func callxpEvent(e interface{}) <-chan string {
	return xpcallback(e)
}

//return
var xpretcallback func(interface{})

func RegisterXpMsgRetCallback(callbackfunc func(interface{})) {
	xpretcallback = callbackfunc
}
func callxpReturn(e interface{}) {
	xpretcallback(e)
}

func callXvcReturn(e interface{}, p2pType int) {
	switch (p2pType) {
	case Dccprotocol_type:
		calldccpReturn(e)
	case Xprotocol_type:
		callxpReturn(e)
	}
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
			enode += XvcDelimiter
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
