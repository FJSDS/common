package tcp

import (
	"reflect"

	"github.com/golang/protobuf/proto"
	"go.uber.org/zap"

	"github.com/FJSDS/common/logger"
	"github.com/FJSDS/common/network/basepb"
)

type DispatchInterface interface {
	OnSessionConnected(*Session)
	OnSessionDisConnected(*Session, error)
	OnRPCRequest(*Session, proto.Message) proto.Message
	OnNormalMsg(*Session, proto.Message)
}

type ConnectFailedInterface interface {
	OnConnectFailed(connector *Connector, err error)
}

type HookDispatchInterface interface {
	OnSessionConnected(*Session) bool
	OnSessionDisConnected(*Session, error) bool
	OnRPCRequest(*Session, proto.Message) (proto.Message, bool)
	OnNormalMsg(*Session, proto.Message) bool
}

type DoConcurrentInterface interface {
	DoConcurrent(msgID string) bool
}

type DefaultDoConcurrent struct {
	m     map[string]struct{}
	isAll bool
}

func (this_ *DefaultDoConcurrent) DoConcurrent(msgID string) bool {
	if this_.m == nil {
		return false
	}
	_, ok := this_.m[msgID]
	return ok
}

// 如果注册了 对应的并发消息。不管是 rpc 还是 普通消息 ，那么收到此消息将会在协程池中调用相关函数，
func (this_ *DefaultDoConcurrent) RegisterConcurrentMsg(message proto.Message) {
	if this_.m == nil {
		this_.m = map[string]struct{}{}
	}
	msgName := proto.MessageName(message)
	if msgName != "" {
		this_.m[msgName] = struct{}{}
	}
}

type AllDoConcurrent struct {
	notConcurrent map[string]struct{}
}

func (this_ *AllDoConcurrent) DoConcurrent(msgID string) bool {
	if this_.notConcurrent == nil {
		return true
	}
	_, ok := this_.notConcurrent[msgID]
	return !ok
}

// 如果注册了 对应的并发消息。不管是 rpc 还是 普通消息 ，那么收到此消息将会在协程池中调用相关函数，
func (this_ *AllDoConcurrent) RegisterNotConcurrentMsg(message proto.Message) {
	if this_.notConcurrent == nil {
		this_.notConcurrent = map[string]struct{}{}
	}
	msgName := proto.MessageName(message)
	if msgName != "" {
		this_.notConcurrent[msgName] = struct{}{}
	}
}

type DefaultLogDispatch struct {
	log *logger.Logger
}

func (this_ *DefaultLogDispatch) OnSessionConnected(s *Session) {
	this_.log.Debug("session connected", zap.String("addr", s.RemoteAddr()))
}

func (this_ *DefaultLogDispatch) OnSessionDisConnected(s *Session, err error) {
	if s != nil {
		this_.log.Debug("session disconnected", zap.Error(err), zap.String("addr", s.RemoteAddr()))
	} else {
		this_.log.Debug("session disconnected", zap.Error(err))
	}

}

func (this_ *DefaultLogDispatch) OnRPCRequest(s *Session, msg proto.Message) proto.Message {
	this_.log.Debug("session recv request msg", zap.String("server name", s.SessionName), zap.String("server id", s.SessionID),
		zap.String("msg name", reflect.TypeOf(msg).Name()), zap.Any("msg", msg))
	return &basepb.Base_Success{}
}

func (this_ *DefaultLogDispatch) OnNormalMsg(s *Session, msg proto.Message) {
	this_.log.Debug("session recv normal msg", zap.String("server name", s.SessionName), zap.String("server id", s.SessionID),
		zap.String("msg name", reflect.TypeOf(msg).Name()), zap.Any("msg", msg))
}

func DispatchMsg(f func(event interface{})) func(event interface{}) {
	if f == nil {
		panic("f==nil")
	}
	return func(event interface{}) {
		switch e := event.(type) {
		case *ConnectorInfo:
			session := e.GetSession()
			if session == nil {
				if e.Error != nil {
					e.Connector.OnSessionDisConnected(nil, e.Error)
				} else {
					e.Connector.OnSessionConnected(e.GetSession())
				}
			} else {
				if session.Dispatch != nil {
					if e.Error != nil {
						session.Dispatch.OnSessionDisConnected(e.GetSession(), e.Error)
					} else {
						session.Dispatch.OnSessionConnected(e.GetSession())
					}
				} else {
					f(event)
				}
			}

		case *AcceptSession:
			if e.Dispatch != nil {
				e.Dispatch.OnSessionConnected(e.Session)
			} else {
				f(event)
			}
		case *SessionClosed:
			if e.GetSession().Dispatch != nil {
				e.GetSession().Dispatch.OnSessionDisConnected(e.GetSession(), e.Err)
			} else {
				f(event)
			}
		case *Packet:
			switch e.Protocol {
			case packetProtocolRPCRequest:
				sess := e.GetSession()
				if sess != nil && sess.Dispatch != nil {
					resp := sess.Dispatch.OnRPCRequest(sess, e.Msg)
					err := sess.sendResponse(e.RPCIndex, resp)
					if err != nil {
						sess.Close(err)
					}
				} else {
					f(event)
				}
			case packetProtocolRPCResponse:
				if e.Sess != nil {
					e.Sess.response(e.RPCIndex, e.Msg)
				} else {
					f(event)
				}

			case packetProtocolNormal:
				sess := e.GetSession()
				if sess != nil && sess.Dispatch != nil {
					if sess.isDebugLog {
						sess.logger.Debug("session recv msg", zap.String("sessionName", sess.SessionName), zap.String("sessionID", sess.SessionID), zap.String("msgID", e.MsgID), zap.Any("msg", e.Msg))
					}
					sess.Dispatch.OnNormalMsg(sess, e.Msg)
				} else {
					f(event)
				}
			}
		default:
			f(event)
		}
	}
}
