// Copyright 2014 The go-ethereum Authors
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

package layer2

import (
	"context"
	"fmt"
	"net"
	"os"
	"time"
	"errors"

	mapset "github.com/deckarep/golang-set"
	"github.com/xvalue/go-xvalue/common"
	"github.com/xvalue/go-xvalue/crypto/sha3"
	"github.com/xvalue/go-xvalue/log"
	"github.com/xvalue/go-xvalue/p2p"
	"github.com/xvalue/go-xvalue/p2p/discover"
	"github.com/xvalue/go-xvalue/rlp"
	"github.com/xvalue/go-xvalue/rpc"
)

// txs start
func SendToDccpGroup(msg string) string {
	return discover.SendToDccpGroup(msg)
}

// broadcast
// to group's nodes
func BroadcastToGroup(msg string) {
	emitter.Lock()
	defer emitter.Unlock()

	log.Debug("==== BroadcastToGroup() ====\n")
	if msg == "" || emitter == nil {
		return
	}
	log.Debug("BroadcastToGroup", "sendMsg", msg)
	dccpgroupfail := NewDccpGroup()
	broatcast := func(dccpGroup *Group, fail bool) int {
		if dccpGroup == nil {
			return 0
		}
		var ret int = 0
		log.Debug("emitter", "peer: ", emitter)
		for _, g := range dccpGroup.group {
			log.Debug("group", "g: ", g)
			//if selfid == g.id {
			//	continue
			//}
			if fail == false {
				if dccpgroupfail.group[g.id.String()] == nil {
					continue
				}
			}
			p := emitter.peers[g.id]
			if p == nil {
				log.Debug("BroadastToGroup", "NodeID: ", g.id, "not in peers\n")
				continue
			}
			if err := p2p.Send(p.ws, dccpMsgCode, msg); err != nil {
				log.Debug("send to node(group) failed", "g = ", g, "p.peer = ", p.peer)
				log.Error("BroadcastToGroup", "p2p.Send err", err, "peer id", p.peer.ID())
				ret += 1
				if fail == true {
					dccpgroupfail.group[g.id.String()] = &group{id: g.id, ip: g.ip, port: g.port, enode: g.enode}
				}
			} else {
				tx := Transaction{Payload: []byte(msg)}
				p.knownTxs.Add(tx.hash())
				log.Debug("send to node(group) success", "g = ", g, "p.peer = ", p.peer)
				if fail == false {
					dccpgroupfail.group[g.id.String()] = nil
				}
			}
		}
		return ret
	}
	log.Debug("BroadcastToGroup", "group: ", dccpgroup)
	failret := broatcast(dccpgroup, true)
	if failret != 0 {
		go func() {
			if broatcastFailTimes == 0 {
				return
			}
			log.Debug("BroatcastToGroupFail", "group: ", dccpgroupfail)
			var i int
			for i = 0; i < broatcastFailTimes; i++ {
				log.Debug("BroatcastToGroupFail", "group times", i+1)
				if broatcast(dccpgroupfail, false) == 0 {
					break
				}
				time.Sleep(time.Duration(broatcastFailOnce) * time.Second)
			}
			if i > broatcastFailTimes {
				log.Debug("BroatcastToGroupFail", "group: ", "failed")
				log.Debug("BroatcastToGroupFail", "group: ", dccpgroupfail)
			} else {
				log.Debug("BroatcastToGroupFail", "group: ", "success")
			}
		}()
	}
}
// to all peers
func Broadcast(msg string) {
	log.Debug("==== Broadcast() ====\n")
	if msg == "" || emitter == nil {
		return
	}
	log.Debug("Broadcast", "sendMsg", msg)
	emitter.Lock()
	defer emitter.Unlock()
	func() {
		log.Debug("peer", "emitter", emitter)
		for _, p := range emitter.peers {
			log.Debug("Broadcast", "to , p", p, "msg", p, msg)
			log.Debug("Broadcast", "p.ws", p.ws)
			if err := p2p.Send(p.ws, dccpMsgCode, msg); err != nil {
				log.Error("Broadcast", "p2p.Send err", err, "peer id", p.peer.ID())
				continue
			}
		}
	}()
}

// unicast
// to anyone
func SendMsgToNode(toid discover.NodeID, toaddr *net.UDPAddr, msg string) error {
	log.Debug("==== SendMsgToNode() ====\n")
	return discover.SendMsgToNode(toid, toaddr, msg)
}
// to peers
func SendMsgToPeer(enode string, msg string) error {
	log.Debug("==== SendMsgToPeer() ====\n")
	node, _ := discover.ParseNode(enode)
	p := emitter.peers[node.ID]
	if p == nil {
		log.Debug("Failed: SendToPeer peers mismatch peerID", "peerID", node.ID)
		return errors.New("peerID mismatch!")
	}
	if err := p2p.Send(p.ws, dccpMsgCode, msg); err == nil {
		log.Debug("Failed: SendToPeer", "peerID", node.ID, "msg", msg)
		return err
	}
	log.Debug("Success: SendToPeer", "peerID", node.ID, "msg", msg)
	return nil
}

// callback
// receive private key
func RegisterRecvCallback(recvPrivkeyFunc func(interface{})) {
	discover.RegistermsgCallback(recvPrivkeyFunc)
}
// receive message form peers
func RegisterCallback(recvDccpFunc func(interface{})) {
	callback = recvDccpFunc
}
func callEvent(msg string) {
	callback(msg)
}
// receive message from dccp
func RegisterDccpCallback(dccpcallback func(interface{}) <-chan string) {
	discover.RegisterDccpCallback(dccpcallback)
}
// receive message from dccp result
func RegisterDccpRetCallback(dccpcallback func(interface{})) {
	discover.RegisterDccpRetCallback(dccpcallback)
}

// get info
func (dccp *DccpAPI) Version(ctx context.Context) (v string) {
	return ProtocolVersionStr
}
func (dccp *DccpAPI) Peers(ctx context.Context) []*p2p.PeerInfo {
	var ps []*p2p.PeerInfo
	for _, p := range dccp.dccp.peers {
		ps = append(ps, p.peer.Info())
	}

	return ps
}
// Protocols returns the whisper sub-protocols ran by this particular client.
func (dccp *Dccp) Protocols() []p2p.Protocol {
	return []p2p.Protocol{dccp.protocol}
}
func GetGroup() (int, string) {
	log.Debug("==== GetGroup() ====\n")
	if dccpgroup == nil {
		return 0, ""
	}
	enode := ""
	count := 0
	for i, e := range dccpgroup.group {
		log.Debug("GetGroup", "i", i, "e", e)
		if enode != "" {
			enode += discover.Dccpdelimiter
		}
		enode += e.enode
		count++
	}
	log.Debug("group", "count = ", count, "enode = ", enode)
	//TODO
	return count, enode
}
func GetSelfID() discover.NodeID {
	return discover.GetLocalID()
}

// p2p layer 2
// New creates a Whisper client ready to communicate through the Ethereum P2P network.
func New(cfg *Config) *Dccp {
	log.Debug("====  dccp New  ====\n")
	dccp := &Dccp{
		peers: make(map[discover.NodeID]*peer),
		quit:  make(chan struct{}),
		cfg:   cfg,
	}

	// p2p dccp sub protocol handler
	dccp.protocol = p2p.Protocol{
		Name:    ProtocolName,
		Version: ProtocolVersion,
		Length:  NumberOfMessageCodes,
		Run:     HandlePeer,
		NodeInfo: func() interface{} {
			return map[string]interface{}{
				"version": ProtocolVersionStr,
			}
		},
		PeerInfo: func(id discover.NodeID) interface{} {
			if p := emitter.peers[id]; p != nil {
				return p.peerInfo
			}
			return nil
		},
	}

	return dccp
}
func init() {
	glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(false)))
	//glogger.Verbosity(log.Lvl(*verbosity))
	log.Root().SetHandler(glogger)

	emitter = NewEmitter()
	discover.RegisterGroupCallback(recvGroupInfo)
	//TODO callback
	//RegisterRecvCallback(recvPrivkeyInfo)
}
func NewEmitter() *Emitter {
	//fmt.Println("========  NewEmitter()  ========")
	return &Emitter{peers: make(map[discover.NodeID]*peer)}
}
func NewDccpGroup() *Group {
	return &Group{group: make(map[string]*group)}
}

// update p2p
func (e *Emitter) addPeer(p *p2p.Peer, ws p2p.MsgReadWriter) {
	fmt.Println("==== addPeer() ====")
	fmt.Printf("id: %+v ...\n", p.ID().String()[:8])
	log.Debug("addPeer", "p: ", p, "ws: ", ws)
	e.Lock()
	defer e.Unlock()
	e.peers[p.ID()] = &peer{ws: ws, peer: p, peerInfo: &peerInfo{int(ProtocolVersion)}, knownTxs: mapset.NewSet()}
}
func HandlePeer(peer *p2p.Peer, rw p2p.MsgReadWriter) error {
	log.Debug("==== HandlePeer() ====\n")
	emitter.addPeer(peer, rw)
	id := peer.ID()
	log.Debug("emitter", "emitter.peers: ", emitter.peers)
	for {
		msg, err := rw.ReadMsg()
		log.Debug("HandlePeer", "ReadMsg", msg)
		if err != nil {
			return err
		}
		log.Debug("HandlePeer", "receive Msgs msg.Payload", msg.Payload)
		switch msg.Code {
		case dccpMsgCode:
			log.Debug("HandlePeer", "receive Msgs from peer", peer)
			log.Debug("emitter", "emitter.peers[id]: ", emitter.peers[id])
			var recv []byte

			err := rlp.Decode(msg.Payload, &recv)
			log.Debug("Decode", "rlp.Decode", recv)
			if err != nil {
				fmt.Printf("Err: decode msg err %+v\n", err)
			} else {
				log.Debug("HandlePeer", "callback(msg): ", recv)
				//tx := Transaction{Payload: []byte(recv)}
				//emitter.broadcastInGroup(tx)
				go callEvent(string(recv))
			}
		default:
			fmt.Println("unkown msg code")
		}
	}
	return nil
}
func recvGroupInfo(req interface{}) {
	log.Debug("==== recvGroupInfo() ====\n")
	selfid = discover.GetLocalID()
	log.Debug("recvGroupInfo", "local ID: ", selfid)
	log.Debug("recvGroupInfo", "req = ", req)
	dccpgroup = NewDccpGroup()
	for i, enode := range req.([]*discover.Node) {
		log.Debug("recvGroupInfo", "i: ", i, "e: ", enode)
		node, _ := discover.ParseNode(enode.String())
		dccpgroup.group[node.ID.String()] = &group{id: node.ID, ip: node.IP, port: node.UDP, enode: enode.String()}
		log.Debug("recvGroupInfo", "dccpgroup.group = ", dccpgroup.group[node.ID.String()])
	}
	log.Debug("recvGroupInfo", "dccpgroup = ", dccpgroup)
}
//TODO callback
func recvPrivkeyInfo(msg interface{}) {
	log.Debug("==== recvPrivkeyInfo() ====\n")
	log.Debug("recvprikey", "msg = ", msg)
	//TODO
	//store privatekey slice
	time.Sleep(time.Duration(10) * time.Second)
	BroadcastToGroup("aaaa")
}

// other
// Start implements node.Service, starting the background data propagation thread
// of the Whisper protocol.
func (dccp *Dccp) Start(server *p2p.Server) error {
	fmt.Println("==== func (dccp *Dccp) Start() ====")
	return nil
}

// Stop implements node.Service, stopping the background data propagation thread
// of the Whisper protocol.
func (dccp *Dccp) Stop() error {
	return nil
}
// APIs returns the RPC descriptors the Whisper implementation offers
func (dccp *Dccp) APIs() []rpc.API {
	return []rpc.API{
		{
			Namespace: ProtocolName,
			Version:   ProtocolVersionStr,
			Service:   &DccpAPI{dccp: dccp},
			Public:    true,
		},
	}
}

// broadcastInGroup will propagate a batch of message to all peers which are not known to
// already have the given message.
func (e *Emitter) broadcastInGroup(tx Transaction) {
	e.Lock()
	defer e.Unlock()

	var txset = make(map[*peer][]Transaction)

	// Broadcast message to a batch of peers not knowing about it
	peers := e.peersWithoutTx(tx.hash(), true)
	log.Debug("broadcastInGroup", "peers", peers)
	for _, peer := range peers {
		txset[peer] = append(txset[peer], tx)
	}
	log.Trace("Broadcast transaction", "hash", tx.hash(), "recipients", len(peers))

	for peer, txs := range txset {
		peer.sendTx(txs)
	}
}

// group: true, in group
//        false, peers
func (e *Emitter) peersWithoutTx(hash common.Hash, group bool) []*peer {
	list := make([]*peer, 0, len(e.peers))
	if group == true {
		if dccpgroup == nil || len(dccpgroup.group) == 0 {
			return list
		}
		for _, g := range dccpgroup.group {
			if g.id == selfid {
				continue
			}
			log.Debug("peersWithoutTx", "emitter", e)
			log.Debug("peersWithoutTx", "g.id", g.id)
			p := e.peers[g.id]
			if p != nil && !p.knownTxs.Contains(hash) {
				list = append(list, p)
			}
		}
	} else {
		for _, p := range e.peers {
			if !p.knownTxs.Contains(hash) {
				list = append(list, p)
			}
		}
	}
	return list
}

// SendTransactions sends transactions to the peer and includes the hashes
// in its transaction hash set for future reference.
func (p *peer) sendTx(txs []Transaction) {
	for _, tx := range txs {
		if err := p2p.Send(p.ws, dccpMsgCode, string(tx.Payload)); err == nil {
			if len(p.queuedTxs) >= maxKnownTxs {
				p.knownTxs.Pop()
			}
			p.knownTxs.Add(tx.hash())
		}
	}
}

// Hash hashes the RLP encoding of tx.
// It uniquely identifies the transaction.
func (tx *Transaction) hash() common.Hash {
	if hash := tx.Hash.Load(); hash != nil {
		return hash.(common.Hash)
	}
	v := rlpHash(tx.Payload)
	tx.Hash.Store(v)
	return v
}

func rlpHash(x interface{}) (h common.Hash) {
	hw := sha3.NewKeccak256()
	rlp.Encode(hw, x)
	hw.Sum(h[:0])
	return h
}

func SendMsg(msg string) {
	BroadcastToGroup(msg)
}

func GetEnodes() (int, string) {
	return GetGroup()
}

func SendToPeer(enode string, msg string) {
	log.Debug("==== DCCP SendToPeer ====\n")
	log.Debug("SendToPeer", "enode: ", enode, "msg: ", msg)
	node, _ := discover.ParseNode(enode)
	//log.Debug("node.id: %+v, node.IP: %+v, node.UDP: %+v\n", node.ID, node.IP, node.UDP)
	ipa := &net.UDPAddr{IP: node.IP, Port: int(node.UDP)}
	discover.SendMsgToNode(node.ID, ipa, msg)
}

