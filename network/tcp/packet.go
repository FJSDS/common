package tcp

import (
	"github.com/golang/protobuf/proto"
)

type Packet struct {
	Protocol packetProtocol
	RPCIndex uint32
	MsgID    string
	Msg      proto.Message
	Sess     *Session
}

const (
	maxPacketLen      = 1024 * 4
	totalLenLen       = 4
	msgIDLenLen       = 1
	packetProtocolLen = 1
	rpcIndexLen       = 4
	headerLen         = totalLenLen + msgIDLenLen + packetProtocolLen + rpcIndexLen
	//packet = totalLenLen +msgIDLenLen+ packetProtocolLen + rpcIndexLen + msgID + body
)

type MsgIDType string

const (
	RPCTimeout MsgIDType = "RPC_REQUEST_TIMEOUT"
	RPCError   MsgIDType = "RPC_REQUEST_ERROR"
)

type packetProtocol uint8

const (
	packetProtocolNormal      packetProtocol = 0
	packetProtocolRPCRequest  packetProtocol = 1
	packetProtocolRPCResponse packetProtocol = 2
	packetProtocolPing        packetProtocol = 3
	packetProtocolPong        packetProtocol = 4
)

func (this_ *Packet) GetSession() *Session {
	return this_.Sess
}

type ConnectorInfo struct {
	Connector *Connector
	Error     error
}

func (this_ *ConnectorInfo) GetSession() *Session {
	return this_.Connector.sess
}
