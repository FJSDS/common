package tcp

import (
	"encoding/json"

	"github.com/golang/protobuf/proto"
)

type CodeC interface {
	Encode(v proto.Message) ([]byte, error)
	Decode(data []byte, v proto.Message) error
	String() string
}

type JSONCodeC struct{}

func (this_ *JSONCodeC) String() string {
	return "json"
}

func (this_ *JSONCodeC) Encode(v proto.Message) ([]byte, error) {
	return json.Marshal(v)
}

func (this_ *JSONCodeC) Decode(data []byte, v proto.Message) error {
	return json.Unmarshal(data, v)
}

type ProtoCodeC struct{}

func (this_ *ProtoCodeC) String() string {
	return "proto"
}

func (this_ *ProtoCodeC) Encode(v proto.Message) ([]byte, error) {
	return proto.Marshal(v)
}

func (this_ *ProtoCodeC) Decode(data []byte, v proto.Message) error {
	return proto.Unmarshal(data, v)
}
