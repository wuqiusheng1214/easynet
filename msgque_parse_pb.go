/*
@Time       : 2022/1/3
@Author     : wuqiusheng
@File       : msgque_parse_pb.go
@Description: pb解析器
*/
package easynet

import (
	"easyutil"
)

type PBParser struct {
	*Parser
}

func (r *PBParser) ParseC2S(msg *Message) (IMsgParser, error) {
	if msg == nil {
		return nil, ErrPBUnPack
	}

	if msg.Head == nil {
		if len(msg.Data) == 0 {
			return nil, ErrPBUnPack
		}
		for _, p := range r.typMap {
			if p.C2S() != nil {
				err := easyutil.PBUnPack(msg.Data, p.C2S())
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
				err := easyutil.PBUnPack(msg.Data, p.C2S())
				if err != nil {
					return nil, err
				}
			}
			p.parser = r
			return &p, nil
		}
	}

	return nil, ErrPBUnPack
}

func (r *PBParser) PackMsg(v interface{}) []byte {
	data, _ := easyutil.PBPack(v)
	return data
}
