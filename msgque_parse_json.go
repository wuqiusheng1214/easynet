/*
@Time       : 2022/1/3
@Author     : wuqiusheng
@File       : msgque_parse_json.go
@Description: json解析器
*/
package easynet

import (
	"easyutil"
)

type JsonParser struct {
	*Parser
}

func (r *JsonParser) ParseC2S(msg *Message) (IMsgParser, error) {
	if msg == nil {
		return nil, ErrJsonUnPack
	}

	if msg.Head == nil {
		if len(msg.Data) == 0 {
			return nil, ErrJsonUnPack
		}
		for _, p := range r.typMap {
			if p.C2S() != nil {
				err := easyutil.JsonUnPack(msg.Data, p.C2S())
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
				err := easyutil.JsonUnPack(msg.Data, p.C2S())
				if err != nil {
					return nil, err
				}
			}
			p.parser = r
			return &p, nil
		}
	}

	return nil, ErrJsonUnPack
}

func (r *JsonParser) PackMsg(v interface{}) []byte {
	data, _ := easyutil.JsonPack(v)
	return data
}
