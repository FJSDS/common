package tcp

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"reflect"
	"sync"
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

type SessionInterface interface {
	GetSession() *Session
}

type CloseError struct {
	Err error
}

func (this_ *CloseError) Error() string {
	return this_.Err.Error()
}

type Session struct {
	rpcIndex uint32
	closed   uint32
	CodeC
	sync.Once
	conn        net.Conn
	remoteAddr  string
	localAddr   string
	remoteIP    string
	sendBuf     [][]byte
	sendCond    *sync.Cond
	queue       *eventloop.EventLoop
	onClose     func(error)
	SessionName string
	SessionID   string
	Dispatch    DispatchInterface
	rpcTimeout  time.Duration
	rpcFunc     *sync.Map

	logger *logger.Logger
	// 采用哪种发送模式，0 默认模式，一个消息一个消息的发送出去，适用于客户端和服务器的连接
	// 1 批量发送出去，但是每次最多发送65535个字节，适用于服务器和服务器之间的连接
	sendFlag     int
	lastPongTime int64
	isDebugLog   bool
}

func NewSession(conn net.Conn, queue *eventloop.EventLoop, batchSend bool, codec CodeC, logger *logger.Logger, rpcTimeout time.Duration, isDebugLog bool) *Session {
	if rpcTimeout < time.Millisecond*500 {
		rpcTimeout = time.Millisecond * 500
	}

	s := &Session{
		conn:         conn,
		sendCond:     sync.NewCond(new(sync.Mutex)),
		queue:        queue,
		CodeC:        codec,
		rpcTimeout:   rpcTimeout,
		rpcFunc:      &sync.Map{},
		logger:       logger,
		lastPongTime: timer.NowUnixSecond(),
		isDebugLog:   isDebugLog,
	}

	if batchSend {
		s.sendFlag = 1
	}

	s.remoteAddr = conn.RemoteAddr().String()
	s.localAddr = conn.LocalAddr().String()
	s.remoteIP = conn.RemoteAddr().(*net.TCPAddr).IP.String()
	s.Dispatch = &DefaultLogDispatch{logger}

	return s
}

func (this_ *Session) getRPCRequestIndex() uint32 {
	return atomic.AddUint32(&this_.rpcIndex, 1)
}

func (this_ *Session) GetLogger() *logger.Logger {
	return this_.logger
}

func (this_ *Session) RemoteTcp4IP() string {
	h, _, err := net.SplitHostPort(this_.remoteAddr)
	if err != nil {
		return ""
	}
	return h
}

func (this_ *Session) Next(prevTime time.Time) time.Time {
	if this_.isClose() {
		return time.Time{}
	}
	return prevTime.Add(time.Second * 2)
}

func (this_ *Session) RemoteAddr() string {
	return this_.remoteAddr
}

func (this_ *Session) LocalAddr() string {
	return this_.localAddr
}

func (this_ *Session) Start() {
	this_.recvLoop()

	if this_.sendFlag == 0 {
		this_.sendLoop()
	} else {
		this_.sendLoopBatch()
	}
}

func (this_ *Session) response(rpcIndex uint32, msg proto.Message) {

	v, ok := this_.rpcFunc.Load(rpcIndex)
	if ok {
		this_.rpcFunc.Delete(rpcIndex)

		if f, ok := v.(RPCResponse); ok {
			if this_.isDebugLog {
				this_.logger.Debug("session recv response", zap.String("session name", this_.SessionName), zap.String("session id", this_.SessionID),
					zap.Uint32("rpcIndex", rpcIndex), zap.String("msg name", proto.MessageName(msg)), zap.Any("msg", msg))
			}
			f(msg)
		} else {
			if this_.isDebugLog {
				this_.logger.Debug("session recv response,but callback type is not RPCResponse", zap.String("session name", this_.SessionName), zap.String("session id", this_.SessionID),
					zap.Uint32("rpcIndex", rpcIndex), zap.String("msg name", proto.MessageName(msg)), zap.Any("msg", msg),
					zap.String("callback type", reflect.TypeOf(f).Name()))
			}
		}
	} else {
		if this_.isDebugLog {
			this_.logger.Debug("session recv response,but not found callback", zap.String("session name", this_.SessionName), zap.String("session id", this_.SessionID),
				zap.Uint32("rpcIndex", rpcIndex), zap.String("msg name", proto.MessageName(msg)), zap.Any("msg", msg))
		}
	}
}

func (this_ *Session) SendNoError(msg proto.Message) {
	err := this_.Send(msg)
	if err != nil {
		this_.logger.Error("send msg failed", zap.Error(err), zap.String("SessionName", this_.SessionName), zap.String("SessionID", this_.SessionID), zap.Any("msg", msg))
	}
}

func (this_ *Session) SendBytes(msgID string, msg []byte) error {
	return this_.sendRawByte(packetProtocolNormal, 0, msgID, msg)
}

func (this_ *Session) SendBytesNoError(msgID string, msg []byte) {
	err := this_.SendBytes(msgID, msg)
	if err != nil {
		this_.logger.Error("SendBytesNoError failed", zap.Error(err), zap.String("SessionName", this_.SessionName), zap.String("msgID", msgID), zap.ByteString("msg", msg))
	}
}

func (this_ *Session) Send(msg proto.Message) error {
	if this_.isClose() {
		return errors.New("session closed")
	}
	err := this_.sendRaw(packetProtocolNormal, 0, msg)
	if err == nil && this_.isDebugLog {
		this_.logger.Debug("send normal msg success", zap.String("remote", this_.remoteAddr), zap.String("session name", this_.SessionName), zap.String("session id", this_.SessionID),
			zap.String("msg name", proto.MessageName(msg)), zap.Any("msg", msg))
	}
	return err
}

func (this_ *Session) SendPing() error {
	if this_.isClose() {
		return errors.New("session closed")
	}
	err := this_.sendRawByte(packetProtocolPing, 0, "ping", nil)
	if err == nil && this_.isDebugLog {
		this_.logger.Debug("send ping msg success", zap.String("remote", this_.remoteAddr), zap.String("session name", this_.SessionName), zap.String("session id", this_.SessionID))
	}
	return err
}

func (this_ *Session) SendPong() error {
	if this_.isClose() {
		return errors.New("session closed")
	}
	err := this_.sendRawByte(packetProtocolPong, 0, "pong", nil)
	if err == nil && this_.isDebugLog {
		this_.logger.Debug("send pong msg success", zap.String("remote", this_.remoteAddr), zap.String("session name", this_.SessionName), zap.String("session id", this_.SessionID))
	}
	return err
}

type RPCResponse func(m proto.Message)

func (this_ *Session) Request(msg proto.Message, resp RPCResponse) error {
	index := this_.getRPCRequestIndex()
	err := this_.sendRaw(packetProtocolRPCRequest, index, msg)
	if err != nil {
		return err
	}
	if this_.isDebugLog {
		this_.logger.Debug("send request msg success", zap.String("session name", this_.SessionName), zap.String("session id", this_.SessionID),
			zap.String("msg name", proto.MessageName(msg)), zap.Any("msg", msg), zap.Uint32("index", index))
	}
	this_.rpcFunc.Store(index, resp)
	this_.queue.AfterFuncQueue(this_.rpcTimeout, func() {
		value, ok := this_.rpcFunc.Load(index)
		if ok {
			this_.rpcFunc.Delete(index)
			switch f := value.(type) {
			case RPCResponse:
				this_.logger.Error("recv response msg timeout", zap.String("session name", this_.SessionName), zap.String("session id", this_.SessionID),
					zap.String("msg name", proto.MessageName(msg)), zap.Any("msg", msg), zap.Uint32("index", index))
				f(&basepb.Base_Error{
					ErrorCode:    1,
					ErrorMessage: "RPCResponse Timeout",
				})
			default:
				this_.logger.Warn("unknown type", zap.Any("response timeout", value))
			}
		}
	})

	return nil
}

func (this_ *Session) RequestNoError(msg proto.Message, resp RPCResponse) {
	err := this_.Request(msg, resp)
	if err != nil {
		resp(&basepb.Base_Error{
			ErrorCode:    1,
			ErrorMessage: "RPCRequest send failed",
		})
		this_.logger.Error("request msg error", zap.Error(err), zap.String("session name", this_.SessionName),
			zap.String("session id", this_.SessionID),
			zap.Any("msg name", proto.MessageName(msg)), zap.Any("msg", msg))
		this_.Close(err)
	}
}

func (this_ *Session) sendResponse(rpcIndex uint32, msg proto.Message) error {
	err := this_.sendRaw(packetProtocolRPCResponse, rpcIndex, msg)
	if err != nil {
		this_.logger.Error("send response msg failed", zap.Error(err), zap.String("session name", this_.SessionName), zap.String("session id", this_.SessionID),
			zap.String("msg name", proto.MessageName(msg)), zap.Any("msg", msg), zap.Uint32("rpcIndex", rpcIndex))
	} else {
		if this_.isDebugLog {
			this_.logger.Debug("send response msg success", zap.String("session name", this_.SessionName), zap.String("session id", this_.SessionID),
				zap.String("msg name", proto.MessageName(msg)), zap.Any("msg", msg), zap.Uint32("rpcIndex", rpcIndex))
		}
	}
	return err
}

func (this_ *Session) isClose() bool {
	return atomic.LoadUint32(&this_.closed) == 1
}

func (this_ *Session) sendRaw(pp packetProtocol, rpcIndex uint32, msg proto.Message) error {
	if this_.isClose() {
		return errors.New("session closed")
	}

	msgID := proto.MessageName(msg)
	if msgID == "" {
		return errors.New("unknown msg,because MessageName is '' ")
	}

	bodyData, err := this_.CodeC.Encode(msg)
	if err != nil {
		return err
	}

	return this_.sendRawByte(pp, rpcIndex, msgID, bodyData)
}

func (this_ *Session) sendRawByte(pp packetProtocol, rpcIndex uint32, msgID string, bodyData []byte) error {
	if this_.isClose() {
		return errors.New("session closed")
	}

	var data []byte

	msgIDLen := len(msgID)
	bodyLen := len(bodyData)

	totalLen := bodyLen + headerLen + msgIDLen
	if totalLen >= maxPacketLen {
		return errors.New("package too large")
	}

	data = make([]byte, totalLen)
	binary.LittleEndian.PutUint32(data[:totalLenLen], uint32(totalLen))
	data[totalLenLen] = uint8(msgIDLen)
	data[totalLenLen+msgIDLenLen] = uint8(pp)
	binary.LittleEndian.PutUint32(data[totalLenLen+msgIDLenLen+packetProtocolLen:], rpcIndex)
	copy(data[headerLen:headerLen+msgIDLen], msgID)
	copy(data[headerLen+msgIDLen:], bodyData)

	this_.sendCond.L.Lock()
	this_.sendBuf = append(this_.sendBuf, data)
	this_.sendCond.L.Unlock()
	this_.sendCond.Signal()

	return nil
}

func (this_ *Session) writeTimeout(duration time.Duration, data []byte) (int, error) {
	_ = this_.conn.SetWriteDeadline(timer.Now().Add(duration))
	n, err := this_.conn.Write(data)
	_ = this_.conn.SetWriteDeadline(time.Time{})

	return n, err
}

func (this_ *Session) recvLoop() {
	utils.SafeGO(func(e interface{}) {
		this_.Close(fmt.Errorf("%+v", e))
	}, func() {
		if this_.isDebugLog {
			defer this_.logger.Debug("recvLoop closed", zap.String("RemoteAddr", this_.RemoteAddr()), zap.String("LocalAddr", this_.LocalAddr()))
		}
		readData := make([]byte, maxPacketLen)
		readStart := 0
		for !this_.isClose() {
			n, err := this_.conn.Read(readData[readStart:])
			if err != nil {
				this_.Close(err)
				return
			}
			readStart += n
			gotData := readData[:readStart]
			pkgCount := 0
			for {
				p, l, err := this_.recvOne(gotData)
				if err != nil {
					this_.Close(err)
					return
				}
				if p != nil {
					pkgCount++
					switch p.Protocol {
					case packetProtocolPing:
						err = this_.SendPong()
						if err != nil {
							this_.Close(err)
							return
						}
						atomic.StoreInt64(&this_.lastPongTime, timer.Now().Unix())
					case packetProtocolPong:
						atomic.StoreInt64(&this_.lastPongTime, timer.Now().Unix())
					default:
						this_.queue.PostEventQueue(p)
					}
					gotData = gotData[l:]
				} else {
					break
				}
			}
			if pkgCount > 0 {
				copy(readData, gotData)
			}
			readStart = len(gotData)
		}
	})
}

func (this_ *Session) recvOne(gotData []byte) (*Packet, int, error) {
	if len(gotData) < headerLen {
		return nil, 0, nil
	}

	msgLen := int(binary.LittleEndian.Uint32(gotData))
	if msgLen > maxPacketLen {
		return nil, 0, fmt.Errorf("msgLen too big %d", msgLen)
	}

	if msgLen < headerLen {
		return nil, 0, fmt.Errorf("msgLen too small %d", msgLen)
	}

	if msgLen > len(gotData) {
		return nil, 0, nil
	}

	msgIDLen := int(gotData[totalLenLen])
	pp := packetProtocol(gotData[totalLenLen+msgIDLenLen])
	rpcIndex := binary.LittleEndian.Uint32(gotData[totalLenLen+msgIDLenLen+packetProtocolLen:])
	bodyLen := msgLen - headerLen

	if bodyLen < msgIDLen {
		return nil, 0, fmt.Errorf("bodyLen too small %d", msgLen)
	}

	msgID := string(gotData[headerLen : headerLen+msgIDLen])
	if msgID == "" {
		this_.Close(fmt.Errorf("invalid msgID %s", msgID))
		return nil, 0, fmt.Errorf("invalid msgID %s", msgID)
	}
	if pp == packetProtocolPing || pp == packetProtocolPong {
		return &Packet{MsgID: msgID, Sess: this_, Msg: nil, Protocol: pp, RPCIndex: rpcIndex}, msgLen, nil
	}
	t := proto.MessageType(msgID)
	if t == nil {
		return nil, 0, fmt.Errorf("invalid msgID %s,can`t found type", msgID)
	}

	msg := reflect.New(t.Elem()).Interface().(proto.Message)

	if bodyLen > msgIDLen {
		body := gotData[msgIDLen+headerLen : msgLen]
		err := this_.CodeC.Decode(body, msg)
		if err != nil {
			return nil, 0, err
		}
	}
	return &Packet{MsgID: msgID, Sess: this_, Msg: msg, Protocol: pp, RPCIndex: rpcIndex}, msgLen, nil
}

func (this_ *Session) sendLoopBatch() {
	utils.SafeGO(func(e interface{}) {
		this_.Close(fmt.Errorf("%+v", e))
	}, func() {
		sendCacheBuf := make([]byte, maxPacketLen)
		sendCacheBufLen := 0
		if this_.isDebugLog {
			defer this_.logger.Debug("sendLoop closed", zap.String("RemoteAddr", this_.RemoteAddr()), zap.String("LocalAddr", this_.LocalAddr()))
		}
		for !this_.isClose() {
			this_.sendCond.L.Lock()
			for len(this_.sendBuf) == 0 {
				this_.sendCond.Wait()
				if this_.isClose() {
					return
				}
			}

			writeBuf := this_.sendBuf
			this_.sendBuf = nil
			this_.sendCond.L.Unlock()

			lenWriteBuf := len(writeBuf)
			if lenWriteBuf == 1 {
				err := this_.sendOneBuf(writeBuf[0])
				if err != nil {
					return
				}
			} else {
				for i := 0; i < lenWriteBuf; i++ {
					if len(writeBuf[i])+sendCacheBufLen > maxPacketLen {
						err := this_.sendOneBuf(sendCacheBuf[:sendCacheBufLen])
						if err != nil {
							return
						}

						sendCacheBufLen = 0
						copy(sendCacheBuf[sendCacheBufLen:], writeBuf[i])
						sendCacheBufLen += len(writeBuf[i])
					} else {
						copy(sendCacheBuf[sendCacheBufLen:], writeBuf[i])
						sendCacheBufLen += len(writeBuf[i])
					}
				}

				if sendCacheBufLen > 0 {
					err := this_.sendOneBuf(sendCacheBuf[:sendCacheBufLen])
					if err != nil {
						return
					}
				}
				sendCacheBufLen = 0
			}
		}
	})
}

func (this_ *Session) sendOneBuf(b []byte) error {
	for len(b) > 0 {
		n, err := this_.writeTimeout(time.Second*2, b)
		if err != nil {
			this_.Close(err)
			return err
		}

		if n == len(b) {
			break
		}

		b = b[n:]
	}

	return nil
}

func (this_ *Session) sendLoop() {
	utils.SafeGO(func(e interface{}) {
		this_.Close(fmt.Errorf("%+v", e))
	}, func() {
		defer this_.logger.Debug("sendLoop closed", zap.String("RemoteAddr", this_.RemoteAddr()), zap.String("LocalAddr", this_.LocalAddr()))
		for !this_.isClose() {
			this_.sendCond.L.Lock()
			for len(this_.sendBuf) == 0 {
				this_.sendCond.Wait()
				if this_.isClose() {
					return
				}
			}

			writeBuf := this_.sendBuf
			this_.sendBuf = nil
			this_.sendCond.L.Unlock()
			lenWriteBuf := len(writeBuf)
			for i := 0; i < lenWriteBuf; i++ {
				item := writeBuf[i]
				for len(item) > 0 {
					n, err := this_.writeTimeout(time.Second*2, item)
					if err != nil {
						this_.Close(err)
						return
					}
					if n == len(item) {
						break
					}
					item = item[n:]
				}
			}
		}
	})
}

func (this_ *Session) Close(err error) {
	this_.Do(func() {
		atomic.StoreUint32(&this_.closed, 1)
		_ = this_.conn.Close()
		this_.sendCond.Signal()
		if this_.onClose != nil {
			this_.onClose(err)
			this_.onClose = nil
		}
	})
}

var syncWaitRPCResponsePool = sync.Pool{
	New: func() interface{} {
		return make(chan proto.Message, 1)
	},
}

var basepbErrorName = proto.MessageName(&basepb.Base_Error{})

// 用于异步RPC同步等待，别在queue或者逻辑线程里面用，只适合http逻辑里面调用，会阻塞
func SyncRequest(s *Session, msg proto.Message, out proto.Message) error {
	if s == nil {
		return errors.New("session is nil")
	}
	c := syncWaitRPCResponsePool.Get().(chan proto.Message)
	defer syncWaitRPCResponsePool.Put(c)
	err := s.Request(msg, func(m proto.Message) {
		c <- m
	})
	if err != nil {
		return err
	}
	resp := <-c
	outName := proto.MessageName(out)
	name := proto.MessageName(resp)
	switch name {
	case basepbErrorName:
		o := resp.(*basepb.Base_Error)
		return errors.New(o.ErrorMessage)
	case outName:
		out = resp
		return nil
	default:
		return fmt.Errorf("recv unknown message:%s,expected:%s", name, outName)
	}
}
