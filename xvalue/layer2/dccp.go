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
	"net"
	"time"

	"github.com/xvalue/go-xvalue/log"
	"github.com/xvalue/go-xvalue/p2p"
	"github.com/xvalue/go-xvalue/p2p/discover"
	"github.com/xvalue/go-xvalue/rpc"
)

// txs start
func Dccprotocol_sendToGroup(msg string) string {
	return discover.SendToGroup(msg, Dccprotocol_type)
}

// broadcast
// to group's nodes
func Dccprotocol_broadcastToGroup(msg string) {
	BroadcastToGroup(msg, Dccprotocol_type)
}

// unicast
// to anyone
func Dccprotocol_sendMsgToNode(toid discover.NodeID, toaddr *net.UDPAddr, msg string) error {
	log.Debug("==== SendMsgToNode() ====\n")
	return discover.SendMsgToNode(toid, toaddr, msg)
}

// to peers
func Dccprotocol_sendMsgToPeer(enode string, msg string) error {
	return sendMsgToPeer(enode, msg)
}

// callback
// receive private key
func Dccprotocol_registerPriKeyCallback(recvPrivkeyFunc func(interface{})) {
	discover.RegisterPriKeyCallback(recvPrivkeyFunc)
}

// receive message form peers
func Dccprotocol_registerCallback(recvDccpFunc func(interface{})) {
	Dccp_callback = recvDccpFunc
}
func Dccp_callEvent(msg string) {
	Dccp_callback(msg)
}

// receive message from dccp
func Dccprotocol_registerMsgCallback(dccpcallback func(interface{}) <-chan string) {
	discover.RegisterDccpMsgCallback(dccpcallback)
}

// receive message from dccp result
func Dccprotocol_registerMsgRetCallback(dccpcallback func(interface{})) {
	discover.RegisterDccpMsgRetCallback(dccpcallback)
}

// get info
func Dccprotocol_getGroup() (int, string) {
	return getGroup(Dccprotocol_type)
}

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

// p2p layer 2
// New creates a Whisper client ready to communicate through the Ethereum P2P network.
func DccpNew(cfg *Config) *Dccp {
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

//TODO callback
func recvPrivkeyInfo(msg interface{}) {
	log.Debug("==== recvPrivkeyInfo() ====\n")
	log.Debug("recvprikey", "msg = ", msg)
	//TODO
	//store privatekey slice
	time.Sleep(time.Duration(10) * time.Second)
	Dccprotocol_broadcastToGroup("aaaa")
}

// other
// Start implements node.Service, starting the background data propagation thread
// of the Whisper protocol.
func (dccp *Dccp) Start(server *p2p.Server) error {
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

func SendMsg(msg string) {
	Dccprotocol_broadcastToGroup(msg)
}

func Dccprotocol_getEnodes() (int, string) {
	return getGroup(Dccprotocol_type)
}

