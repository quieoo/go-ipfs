// Code generated by protoc-gen-go.
// source: message.proto
// DO NOT EDIT!

/*
Package bitswap is a generated protocol buffer package.

It is generated from these files:
	message.proto

It has these top-level messages:
	PBMessage
*/
package bitswap

import proto "code.google.com/p/goprotobuf/proto"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = math.Inf

type PBMessage_MessageType int32

const (
	PBMessage_GET_BLOCK  PBMessage_MessageType = 0
	PBMessage_WANT_BLOCK PBMessage_MessageType = 1
)

var PBMessage_MessageType_name = map[int32]string{
	0: "GET_BLOCK",
	1: "WANT_BLOCK",
}
var PBMessage_MessageType_value = map[string]int32{
	"GET_BLOCK":  0,
	"WANT_BLOCK": 1,
}

func (x PBMessage_MessageType) Enum() *PBMessage_MessageType {
	p := new(PBMessage_MessageType)
	*p = x
	return p
}
func (x PBMessage_MessageType) String() string {
	return proto.EnumName(PBMessage_MessageType_name, int32(x))
}
func (x *PBMessage_MessageType) UnmarshalJSON(data []byte) error {
	value, err := proto.UnmarshalJSONEnum(PBMessage_MessageType_value, data, "PBMessage_MessageType")
	if err != nil {
		return err
	}
	*x = PBMessage_MessageType(value)
	return nil
}

type PBMessage struct {
	Type             *PBMessage_MessageType `protobuf:"varint,1,req,enum=bitswap.PBMessage_MessageType" json:"Type,omitempty"`
	Id               *uint64                `protobuf:"varint,2,req,name=id" json:"id,omitempty"`
	Key              *string                `protobuf:"bytes,3,req,name=key" json:"key,omitempty"`
	Value            []byte                 `protobuf:"bytes,4,opt,name=value" json:"value,omitempty"`
	Response         *bool                  `protobuf:"varint,5,opt,name=response" json:"response,omitempty"`
	Success          *bool                  `protobuf:"varint,6,opt,name=success" json:"success,omitempty"`
	Wantlist         []string               `protobuf:"bytes,7,rep,name=wantlist" json:"wantlist,omitempty"`
	XXX_unrecognized []byte                 `json:"-"`
}

func (m *PBMessage) Reset()         { *m = PBMessage{} }
func (m *PBMessage) String() string { return proto.CompactTextString(m) }
func (*PBMessage) ProtoMessage()    {}

func (m *PBMessage) GetType() PBMessage_MessageType {
	if m != nil && m.Type != nil {
		return *m.Type
	}
	return PBMessage_GET_BLOCK
}

func (m *PBMessage) GetId() uint64 {
	if m != nil && m.Id != nil {
		return *m.Id
	}
	return 0
}

func (m *PBMessage) GetKey() string {
	if m != nil && m.Key != nil {
		return *m.Key
	}
	return ""
}

func (m *PBMessage) GetValue() []byte {
	if m != nil {
		return m.Value
	}
	return nil
}

func (m *PBMessage) GetResponse() bool {
	if m != nil && m.Response != nil {
		return *m.Response
	}
	return false
}

func (m *PBMessage) GetSuccess() bool {
	if m != nil && m.Success != nil {
		return *m.Success
	}
	return false
}

func (m *PBMessage) GetWantlist() []string {
	if m != nil {
		return m.Wantlist
	}
	return nil
}

func init() {
	proto.RegisterEnum("bitswap.PBMessage_MessageType", PBMessage_MessageType_name, PBMessage_MessageType_value)
}
