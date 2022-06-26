/*
@Time       : 2022/1/3
@Author     : wuqiusheng
@File       : msgque_parse_msgpack.go
@Description: msgpack解析器
*/
package easynet

import (
	"easyutil"
)

type MsgpackParser struct {
	*Parser
}

func (r *MsgpackParser) ParseC2S(msg *Message) (IMsgParser, error) {
	if msg == nil {
		return nil, ErrMsgPackUnPack
	}

	if msg.Head == nil {
		if len(msg.Data) == 0 {
			return nil, ErrMsgPackUnPack
		}
		for _, p := range r.typMap {
			if p.C2S() != nil {
				err := easyutil.MsgPackUnPack(msg.Data, p.C2S())
				if err != nil {
					continue
				}
				p.parser = r
				return &p, nil
			}
		}
	} else if p, ok := r.msgMap[msg.Head.Id]; ok {
		if p.C2S() != nil {
			if len(msg.Data) > 0 {
				err := easyutil.MsgPackUnPack(msg.Data, p.C2S())
				if err != nil {
					return nil, err
				}
			}
			p.parser = r
			return &p, nil
		}
	}

	return nil, ErrMsgPackUnPack
}

func (r *MsgpackParser) PackMsg(v interface{}) []byte {
	data, _ := easyutil.MsgPackPack(v)
	return data
}
