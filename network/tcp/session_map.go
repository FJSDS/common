package tcp

import (
	"reflect"

	"github.com/golang/protobuf/proto"
	"go.uber.org/zap"

	"github.com/FJSDS/common/logger"
)

type SessionMap struct {
	m   map[string]*Session
	log *logger.Logger
}

func NewSessionMap(log *logger.Logger) *SessionMap {
	return &SessionMap{
		m:   map[string]*Session{},
		log: log,
	}
}

func (this_ *SessionMap) GetSession(sessionID string) (*Session, bool) {
	v, ok := this_.m[sessionID]
	return v, ok
}

func (this_ *SessionMap) SendBySessionID(sessionID string, msg proto.Message) {
	if v, ok := this_.m[sessionID]; ok {
		v.SendNoError(msg)
	} else {
		this_.log.Error("no session found", zap.String("sessionID", sessionID), zap.String("msg name", proto.MessageName(msg)),
			zap.Any("msg", msg))
	}
}

func (this_ *SessionMap) AddSession(s *Session) {
	this_.m[s.SessionID] = s
}

func (this_ *SessionMap) DelSession(s *Session) {
	if v, ok := this_.m[s.SessionID]; ok {
		if v.RemoteAddr() == s.RemoteAddr() {
			delete(this_.m, s.SessionID)
		} else {
			this_.log.Error("DelSession failed :RemoteAddr not equal", zap.String("v", v.RemoteAddr()), zap.String("del", s.RemoteAddr()))
		}
	}
}

func (this_ *SessionMap) SendToAll(msg proto.Message) {
	msgID := proto.MessageName(msg)
	if msgID == "" {
		this_.log.Error("SessionMap SendToAll unknown msg,because MessageName is '' ", zap.String("msg type", reflect.TypeOf(msg).String()))
		return
	}
	d := map[string][]byte{}
	for _, v := range this_.m {
		bodyData, ok := d[v.CodeC.String()]
		if !ok {
			var err error
			bodyData, err = v.CodeC.Encode(msg)
			if err != nil {
				this_.log.Error("SessionMap SendToAll error msg,because CodeC.Encode(msg) failed", zap.String("msg type", reflect.TypeOf(msg).String()))
				return
			}
		}
		err := v.SendBytes(msgID, bodyData)
		if err != nil {
			v.Close(err)
			delete(this_.m, v.SessionID)
			this_.log.Error("SendBytesNoError failed", zap.Error(err), zap.String("SessionName", v.SessionName), zap.String("msgID", msgID), zap.Any("msg", msg))
		}
	}
}
