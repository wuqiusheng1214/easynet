/*
@Time       : 2022/1/3
@Author     : wuqiusheng
@File       : msgque_msg.go
@Description: 消息头，消息
*/
package easynet

import (
	"easyutil"
	"unsafe"
)

const (
	MsgHeadSize = 9
)

const (
	FlagEncrypt  = 1 << 0 //数据是经过加密的
	FlagCompress = 1 << 1 //数据是经过压缩的
	//FlagContinue = 1 << 2 //消息还有后续
	//FlagNeedAck  = 1 << 3 //消息需要确认
	//FlagAck      = 1 << 4 //确认消息
	//FlagReSend   = 1 << 5 //重发消息
	//FlagClient   = 1 << 6 //消息来自客服端，用于判断index来之服务器还是其他玩家
)

var MaxMsgDataSize uint32 = 1024 * 1024

type MessageHead struct {
	Len   uint32 //数据长度
	Id    uint16 //消息ID
	Index uint16 //序号
	Flags uint8  //标记

	data []byte
}

func (r *MessageHead) Bytes() []byte {
	r.data = make([]byte, MsgHeadSize)
	phead := (*MessageHead)(unsafe.Pointer(&r.data[0]))
	phead.Len = r.Len
	phead.Id = r.Id
	phead.Index = r.Index
	phead.Flags = r.Flags
	return r.data
}

func (r *MessageHead) FastBytes(data []byte) []byte {
	phead := (*MessageHead)(unsafe.Pointer(&data[0]))
	phead.Len = r.Len
	phead.Id = r.Id
	phead.Index = r.Index
	phead.Flags = r.Flags
	return data
}

func (r *MessageHead) BytesWithData(wdata []byte) []byte {
	r.Len = uint32(len(wdata))
	r.data = make([]byte, MsgHeadSize+r.Len)
	phead := (*MessageHead)(unsafe.Pointer(&r.data[0]))
	phead.Len = r.Len
	phead.Id = r.Id
	phead.Index = r.Index
	phead.Flags = r.Flags
	if wdata != nil {
		copy(r.data[MsgHeadSize:], wdata)
	}
	return r.data
}

func (r *MessageHead) FromBytes(data []byte) error {
	if len(data) < MsgHeadSize {
		return ErrMsgLenTooShort
	}
	phead := (*MessageHead)(unsafe.Pointer(&data[0]))
	r.Len = phead.Len
	r.Id = phead.Id
	r.Index = phead.Index
	r.Flags = phead.Flags
	if r.Len > MaxMsgDataSize {
		return ErrMsgLenTooLong
	}
	return nil
}

func (r *MessageHead) Tag() uint32 {
	return Tag(r.Id, r.Index)
}

func (r *MessageHead) String() string {
	return easyutil.Sprintf("Len:%v Id:%v Index:%v Flags:%v",
		r.Len, r.Id, r.Index, r.Flags)
}

func NewMessageHead(data []byte) *MessageHead {
	head := &MessageHead{}
	if err := head.FromBytes(data); err != nil {
		return nil
	}
	return head
}

func MessageHeadFromByte(data []byte) *MessageHead {
	if len(data) < MsgHeadSize {
		return nil
	}
	phead := new(*MessageHead)
	*phead = (*MessageHead)(unsafe.Pointer(&data[0]))
	if (*phead).Len > MaxMsgDataSize {
		return nil
	}
	return *phead
}

type Message struct {
	Head       *MessageHead //消息头，可能为nil
	Data       []byte       //消息数据
	IMsgParser              //消息解析器
	User       interface{}  //用户自定义数据
}

func (r *Message) Len() uint32 {
	if r.Head != nil {
		return r.Head.Len
	}
	return 0
}

func (r *Message) Id() uint16 {
	if r.Head != nil {
		return r.Head.Id
	}
	return 0
}

func (r *Message) Index() uint16 {
	if r.Head != nil {
		return r.Head.Index
	}
	return 0
}

func (r *Message) Flags() uint8 {
	if r.Head != nil {
		return r.Head.Flags
	}
	return 0
}

func (r *Message) Tag() uint32 {
	if r.Head != nil {
		return Tag(r.Head.Id, r.Head.Index)
	}
	return 0
}

func (r *Message) Bytes() []byte {
	if r.Head != nil {
		if r.Data != nil {
			return r.Head.BytesWithData(r.Data)
		}
		return r.Head.Bytes()
	}
	return r.Data
}

func (r *Message) CopyTag(old *Message) *Message {
	if r.Head != nil && old.Head != nil {
		r.Head.Id = old.Head.Id
		r.Head.Index = old.Head.Index
	}
	return r
}

func NewStrMsg(str string) *Message {
	return &Message{
		Data: []byte(str),
	}
}

func NewDataMsg(data []byte) *Message {
	return &Message{
		Head: &MessageHead{
			Len: uint32(len(data)),
		},
		Data: data,
	}
}

func NewMsg(id, index uint16, data []byte) *Message {
	return &Message{
		Head: &MessageHead{
			Len:   uint32(len(data)),
			Id:    id,
			Index: index,
		},
		Data: data,
	}
}

func NewTagMsg(id uint16, index uint16) *Message {
	return &Message{
		Head: &MessageHead{
			Id:    id,
			Index: index,
		},
	}
}

func Tag(id uint16, index uint16) uint32 {
	return uint32(id)<<16 + uint32(index)
}
