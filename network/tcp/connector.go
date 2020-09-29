package tcp

import (
	"errors"
	"fmt"
	"net"
	"sync/atomic"
	"time"

	"github.com/golang/protobuf/proto"
	"go.uber.org/zap"

	"github.com/FJSDS/common/eventloop"
	"github.com/FJSDS/common/logger"
	"github.com/FJSDS/common/network/basepb"
	"github.com/FJSDS/common/timer"
	"github.com/FJSDS/common/utils"
)

const (
	ConnectStateInvalid = iota
	ConnectStateConnecting
	ConnectStateConnected
	ConnectStateDisconnected
)

type ConnectError struct {
	Err error
}

func (this_ *ConnectError) Error() string {
	return this_.Err.Error()
}

type Connector struct {
	isReconnect    bool
	isStartHeart   bool
	ci             DispatchInterface
	dc             DoConcurrentInterface
	cf             func(connector *Connector, err error)
	connectState   int64
	addr           string
	ConnectTimeout time.Duration
	queue          *eventloop.EventLoop
	sess           *Session
	codeC          CodeC
	logger         *logger.Logger
	ConnectorName  string
	isDebugLog     bool
}

type Option func(c *Connector)

func WithOpenDebugLog() Option {
	return func(c *Connector) {
		c.isDebugLog = true
	}
}

func WithReconnect() Option {
	return func(c *Connector) {
		c.isReconnect = true
	}
}

func WithJsonCodeC() Option {
	return func(c *Connector) {
		c.codeC = &JSONCodeC{}
	}
}

func WithConnectTimeout(timeout time.Duration) Option {
	return func(c *Connector) {
		c.ConnectTimeout = timeout
	}
}

func (this_ *Connector) GetAddr() string {
	return this_.addr
}

func (this_ *Connector) OnSessionConnected(s *Session) {
	if this_.isDebugLog {
		this_.logger.Debug("connect success", zap.String("addr", this_.addr))
	}
	if this_.ci != nil {
		this_.ci.OnSessionConnected(s)
	}
}

func (this_ *Connector) OnSessionDisConnected(s *Session, err error) {
	if this_.isDebugLog {
		this_.logger.Debug("connect disconnected", zap.Error(err), zap.String("addr", this_.addr))
	}

	if s == nil && this_.cf != nil {
		this_.cf(this_, err)
	} else {
		if this_.isReconnect {
			this_.Connect()
		}
	}
	if this_.ci != nil {
		this_.ci.OnSessionDisConnected(s, err)
	}
}

func (this_ *Connector) OnRPCRequest(s *Session, msg proto.Message) proto.Message {
	if this_.isDebugLog {
		this_.logger.Debug("OnRPCRequest", zap.String("addr", this_.addr), zap.Any("msg name", proto.MessageName(msg)), zap.Any("msg", msg))
	}
	if this_.ci != nil {
		return this_.ci.OnRPCRequest(s, msg)
	}
	return &basepb.Base_Success{}
}

func (this_ *Connector) OnNormalMsg(s *Session, msg proto.Message) {
	if this_.isDebugLog {
		this_.logger.Debug("OnNormalMsg", zap.String("addr", this_.addr), zap.Any("msg name", proto.MessageName(msg)), zap.Any("msg", msg))
	}
	if this_.ci != nil {
		this_.ci.OnNormalMsg(s, msg)
	}
}

func (this_ *Connector) Request(msg proto.Message, resp RPCResponse) (err error) {
	if this_.sess != nil {
		err = this_.sess.Request(msg, resp)
		if err != nil {
			this_.logger.Error("request msg error", zap.Error(err), zap.String("session name", this_.sess.SessionName),
				zap.Any("msgID", proto.MessageName(msg)), zap.Any("msg", msg))
			this_.sess.Close(err)
		}
	} else {
		if resp != nil {
			resp(&basepb.Base_Error{
				ErrorCode:    1,
				ErrorMessage: "connector not connected",
			})
		}
		err = errors.New("invalid session")
	}

	return
}

func (this_ *Connector) RequestNoError(msg proto.Message, resp RPCResponse) {
	if this_.sess != nil {
		this_.sess.RequestNoError(msg, resp)
	} else {
		if resp != nil {
			resp(&basepb.Base_Error{
				ErrorCode:    1,
				ErrorMessage: "connector not connected",
			})
		}
	}
}

func NewConnector(addr string, queue *eventloop.EventLoop, logger *logger.Logger, connectorName string, options ...Option) *Connector {
	c := &Connector{
		addr:           addr,
		ConnectTimeout: time.Second,
		queue:          queue,
		codeC:          &ProtoCodeC{},
		logger:         logger,
		ConnectorName:  connectorName,
	}
	for _, v := range options {
		v(c)
	}
	c.ci = c
	return c
}

func (this_ *Connector) SetOnConnectFailed(f func(connector *Connector, err error)) {
	this_.cf = f
}

func (this_ *Connector) SetCallback(ci DispatchInterface) {
	this_.ci = ci
	if this_.sess != nil {
		this_.sess.Dispatch = ci
	}

}

func (this_ *Connector) SetDoConcurrent(dc DoConcurrentInterface) {
	this_.dc = dc
}

func (this_ *Connector) DoConcurrent(msgID string) bool {
	if this_.dc != nil {
		this_.dc.DoConcurrent(msgID)
	}
	return false
}

func (this_ *Connector) RemoteAddr() string {
	return this_.addr
}

func (this_ *Connector) GetSession() *Session {
	return this_.sess
}

func (this_ *Connector) Send(msg proto.Message) (err error) {
	if this_.sess != nil {
		err = this_.sess.Send(msg)
		if err != nil {
			this_.logger.Error("Send msg error", zap.Error(err), zap.String("msgID", proto.MessageName(msg)), zap.Any("data", msg))
			this_.sess.Close(err)
		}
	} else {
		this_.logger.Error("Send msg failed,session not connected", zap.String("msgID", proto.MessageName(msg)), zap.Any("data", msg))
		err = errors.New("invalid session")
	}
	return
}

func (this_ *Connector) SendNoError(msg proto.Message) {
	_ = this_.Send(msg)
}

//func (this_ *Connector) SendRaw(msg proto.Message) error {
//	err := this_.sess.Send(msg)
//	if err != nil {
//		this_.logger.Error("sendRaw msg error", zap.Error(err), zap.String("msgID", proto.MessageName(msg)), zap.Any("data", msg))
//	}
//
//	return err
//}

func (this_ *Connector) setConnectState(state int64) {
	atomic.StoreInt64(&this_.connectState, state)
}

func (this_ *Connector) getConnectState() int64 {
	return atomic.LoadInt64(&this_.connectState)
}

func (this_ *Connector) Close(err error) {
	if this_.getConnectState() == ConnectStateConnected {
		if this_.sess != nil {
			this_.sess.Close(err)
		}

		this_.setConnectState(ConnectStateDisconnected)
	}
}

func (this_ *Connector) startSession(conn net.Conn) {
	this_.setConnectState(ConnectStateConnected)
	this_.sess = NewSession(conn, this_.queue, false, this_.codeC, this_.logger, time.Second*2, this_.isDebugLog)
	this_.sess.Dispatch = this_
	this_.sess.onClose = func(err error) {
		this_.setConnectState(ConnectStateDisconnected)
		this_.queue.PostEventQueue(&ConnectorInfo{
			Connector: this_,
			Error:     &CloseError{err},
		})
	}
	this_.queue.PostEventQueue(&ConnectorInfo{
		Connector: this_,
		Error:     nil,
	})
	this_.sess.Start()
	this_.startHeartbeat()
}

func (this_ *Connector) startHeartbeat() {
	if this_.isStartHeart {
		return
	}

	if this_.sess != nil {
		atomic.StoreInt64(&this_.sess.lastPongTime, timer.NowUnixSecond())
	}

	this_.isStartHeart = true

	this_.queue.TickPool(time.Second*10, func() bool {
		if timer.Now().Unix()-atomic.LoadInt64(&this_.sess.lastPongTime) > 30 {
			this_.sess.Close(errors.New("ping pong timeout"))
			return false
		}
		return true
	})
}

func (this_ *Connector) Connect() {
	connectState := this_.getConnectState()
	if !this_.queue.Stopped() && (connectState == ConnectStateInvalid || connectState == ConnectStateDisconnected) {
		this_.setConnectState(ConnectStateConnecting)
		utils.SafeGO(func(e interface{}) {
			this_.Close(fmt.Errorf("%+v", e))
			this_.setConnectState(ConnectStateInvalid)
		}, func() {
			conn, err := net.DialTimeout("tcp4", this_.addr, this_.ConnectTimeout)
			if err != nil {
				this_.setConnectState(ConnectStateInvalid)
				this_.queue.PostEventQueue(&ConnectorInfo{
					Connector: this_,
					Error:     &ConnectError{err},
				})
				return
			}
			this_.startSession(conn)
		})
	}
}
