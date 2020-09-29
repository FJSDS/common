package natsclient

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/nats-io/nats.go"
	uuid "github.com/satori/go.uuid"

	"github.com/FJSDS/common/logger"
)

type Request struct {
	Guid     int64           `json:"guid"`
	Path     string          `json:"path"`
	Body     json.RawMessage `json:"post_body,omitempty"`
	IP       string          `json:"ip"`
	UniqueID string          `json:"unique_id"`
	Timeout  time.Duration   `json:"-"`
}

type Response struct {
	Code    int64       `json:"code"`
	Message string      `json:"message"`
	Display string      `json:"display"`
	Fields  []string    `json:"fields"`
	Data    interface{} `json:"data"`
}

func (this_ *Response) Error() string {
	return fmt.Sprintf("code:%d,message:%s,display:%s,fields:%v", this_.Code, this_.Message, this_.Display, this_.Fields)
}

type NATSClient struct {
	*nats.Conn
	log *logger.Logger
}

func NewNATSClient(log *logger.Logger, urls []string) (*NATSClient, error) {
	c, err := nats.Connect(strings.Join(urls, ", "), nats.MaxReconnects(math.MaxInt64), nats.ReconnectWait(time.Millisecond))
	if err != nil {
		return nil, err
	}
	return &NATSClient{Conn: c, log: log}, err
}

type MessageOpt func(r *Request)

func WithGuidOpt(guid int64) func(r *Request) {
	return func(r *Request) {
		r.Guid = guid
	}
}

func WithIPOpt(ip string) func(r *Request) {
	return func(r *Request) {
		r.IP = ip
	}
}

func WithTimeout(d time.Duration) func(r *Request) {
	return func(r *Request) {
		r.Timeout = d
	}
}

func (this_ *NATSClient) RequestProto(req proto.Message, out proto.Message, opts ...MessageOpt) error {
	name := proto.MessageName(req)
	ns := strings.Split(name, ".")
	body, _ := json.Marshal(req)
	uid, err := uuid.NewV4()
	if err != nil {
		return err
	}
	r := &Request{
		Path:     name,
		Body:     body,
		IP:       "",
		UniqueID: uid.String(),
		Guid:     0,
	}
	for _, v := range opts {
		v(r)
	}
	if r.Timeout == 0 {
		r.Timeout = time.Second * 5
	}
	data, _ := json.Marshal(r)
	msg, err := this_.Conn.Request(strings.Join(ns[:2], "."), data, time.Second*5)
	if err != nil {
		return fmt.Errorf("request:%s,error:%s", name, err.Error())
	}
	resp := &Response{
		Data: out,
	}
	err = json.Unmarshal(msg.Data, resp)
	if err != nil {
		return err
	}
	if resp.Code != 0 {
		return resp
	}
	return nil
}

func (this_ *NATSClient) NotifyProto(reqs ...proto.Message) {
	if len(reqs) == 0 {
		return
	}
	for _, req := range reqs {
		name := proto.MessageName(req)
		ns := strings.Split(name, ".")
		if len(ns) != 3 {
			continue
		}
		body, _ := json.Marshal(req)
		uid, err := uuid.NewV4()
		if err != nil {
			continue
		}
		r := &Request{
			Path:     name,
			Body:     body,
			IP:       "",
			UniqueID: uid.String(),
			Guid:     0,
		}
		data, _ := json.Marshal(r)
		_ = this_.Conn.Publish(strings.Join(ns[:2], "."), data)
	}
}
