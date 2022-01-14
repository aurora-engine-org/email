package email

import (
	"bytes"
	"strings"
)

/*
	Message 嵌套设计:
	本文件实现MIME中对 邮件消息体的嵌套设计，思路如下：
	没有混合消息的机制下，一个Message 即可作为一封邮件的完整消息
	存在多部份混合的消息机制下，比如在本库中同时设置了 文本，HTML，文件，3项或者任意2项
	将最外层的 Message 作为容器来存放 文本，HTML，文件信息
*/

// Message 邮件中的消息部分
type Message struct {
	boundaryStart string     //不为空表示本消息开头需要添加边界分割
	header        []*Header  //消息头
	body          string     //正文,一般为字符串，HTML，或者编码后的字符串
	msg           []*Message //嵌套消息
	boundaryEed   string     //边界结尾
}

// Next 附加一个消息
func (m *Message) Next(msg *Message) {
	if m.msg == nil {
		m.msg = make([]*Message, 0)
	}
	m.msg = append(m.msg, msg)
}

// 对消息进行解析，生成最终传输的消息体，该解析实现了 rfc2046 的多部份消息混合解析
func parseMessage(message *Message) []byte {
	buf := &bytes.Buffer{}
	//解析邮件头 检验本身是否属于一个多部份混合消息,如果是多部份混合消息 则自己本身也要使用自己的消息分割符
	if message.header != nil {
		for i := 0; i < len(message.header); i++ {
			if message.header[i].Name == ContentType { //检索是否有内容类型头
				for _, value := range message.header[i].Value { //检索是否需要分段
					if strings.HasPrefix(value, "boundary") {
						//如果有该属性，那么后续的 msg           []*Message //嵌套消息 都需要使用这个作为分割
						message.boundaryStart = "--" + value[9:]
						message.boundaryEed = "--" + value[9:] + "--"
						break
					}
				}
			}
			buf.Write(message.header[i].Encoding())
		}
	}

	if message.body != "" {
		if message.boundaryStart != "" { //当前不是混合邮件，将不会添加分割符号
			buf.WriteString(CRLF + message.boundaryStart + CRLF)
		}
		buf.WriteString(message.body)
	}

	if message.msg != nil {
		// 开始递归解析嵌套消息
		for i := 0; i < len(message.msg); i++ {
			if message.boundaryStart != "" { //由于Message的嵌套机制，如果分割符为为空，可能写入2个换行标识，导致无法解析。
				buf.WriteString(CRLF + message.boundaryStart + CRLF) //每个嵌套的消息模块采用边界分割开来
			}
			s := parseMessage(message.msg[i])
			buf.Write(s)
		}
	}
	if message.boundaryEed != "" {
		buf.WriteString(CRLF + message.boundaryEed + CRLF)
	}
	return buf.Bytes()
}
