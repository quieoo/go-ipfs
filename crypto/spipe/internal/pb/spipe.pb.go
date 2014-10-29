// Code generated by protoc-gen-gogo.
// source: spipe.proto
// DO NOT EDIT!

/*
Package spipe_pb is a generated protocol buffer package.

It is generated from these files:
	spipe.proto

It has these top-level messages:
	Propose
	Exchange
	DataSig
*/
package spipe_pb

import proto "code.google.com/p/gogoprotobuf/proto"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = math.Inf

type Propose struct {
	Rand             []byte  `protobuf:"bytes,1,opt,name=rand" json:"rand,omitempty"`
	Pubkey           []byte  `protobuf:"bytes,2,opt,name=pubkey" json:"pubkey,omitempty"`
	Exchanges        *string `protobuf:"bytes,3,opt,name=exchanges" json:"exchanges,omitempty"`
	Ciphers          *string `protobuf:"bytes,4,opt,name=ciphers" json:"ciphers,omitempty"`
	Hashes           *string `protobuf:"bytes,5,opt,name=hashes" json:"hashes,omitempty"`
	XXX_unrecognized []byte  `json:"-"`
}

func (m *Propose) Reset()         { *m = Propose{} }
func (m *Propose) String() string { return proto.CompactTextString(m) }
func (*Propose) ProtoMessage()    {}

func (m *Propose) GetRand() []byte {
	if m != nil {
		return m.Rand
	}
	return nil
}

func (m *Propose) GetPubkey() []byte {
	if m != nil {
		return m.Pubkey
	}
	return nil
}

func (m *Propose) GetExchanges() string {
	if m != nil && m.Exchanges != nil {
		return *m.Exchanges
	}
	return ""
}

func (m *Propose) GetCiphers() string {
	if m != nil && m.Ciphers != nil {
		return *m.Ciphers
	}
	return ""
}

func (m *Propose) GetHashes() string {
	if m != nil && m.Hashes != nil {
		return *m.Hashes
	}
	return ""
}

type Exchange struct {
	Epubkey          []byte `protobuf:"bytes,1,opt,name=epubkey" json:"epubkey,omitempty"`
	Signature        []byte `protobuf:"bytes,2,opt,name=signature" json:"signature,omitempty"`
	XXX_unrecognized []byte `json:"-"`
}

func (m *Exchange) Reset()         { *m = Exchange{} }
func (m *Exchange) String() string { return proto.CompactTextString(m) }
func (*Exchange) ProtoMessage()    {}

func (m *Exchange) GetEpubkey() []byte {
	if m != nil {
		return m.Epubkey
	}
	return nil
}

func (m *Exchange) GetSignature() []byte {
	if m != nil {
		return m.Signature
	}
	return nil
}

type DataSig struct {
	Data             []byte  `protobuf:"bytes,1,opt,name=data" json:"data,omitempty"`
	Sig              []byte  `protobuf:"bytes,2,opt,name=sig" json:"sig,omitempty"`
	Id               *uint64 `protobuf:"varint,3,opt,name=id" json:"id,omitempty"`
	XXX_unrecognized []byte  `json:"-"`
}

func (m *DataSig) Reset()         { *m = DataSig{} }
func (m *DataSig) String() string { return proto.CompactTextString(m) }
func (*DataSig) ProtoMessage()    {}

func (m *DataSig) GetData() []byte {
	if m != nil {
		return m.Data
	}
	return nil
}

func (m *DataSig) GetSig() []byte {
	if m != nil {
		return m.Sig
	}
	return nil
}

func (m *DataSig) GetId() uint64 {
	if m != nil && m.Id != nil {
		return *m.Id
	}
	return 0
}

func init() {
}
