package email

import (
	"bytes"
	"strings"
)

// Block 用于构建每个正文的结构
type Block struct {
	header   []*Header //该头主要标识正文内容该如何显示
	boundary string    //标识该部分是否存在边界，辅助解析
	content  string
}

// Body 用于构建正文内容
type Body struct {
	must     []*Header //基础头
	boundary string    //标识该部分是否存在边界，辅助解析
	text     []*Block
}

func NewBlock(header []*Header, msg string) *Block {
	return &Block{
		header:  header,
		content: msg,
	}
}
func NewBody(m []*Header, blocks ...*Block) *Body {
	return &Body{
		must: m,
		text: blocks,
	}
}

// SetHeader 设置消息头
func (b *Block) SetHeader(header ...*Header) {
	if b.header == nil {
		b.header = make([]*Header, 0)
	}
	b.header = append(b.header, header...)
}

// SetContent 设置消息
func (b *Block) SetContent(msg string) {
	b.content = msg
}

// Message 把每个 Block 格式化为字节以便传输
func (b *Block) Message() []byte {
	buffer := bytes.Buffer{}
	for _, v := range b.header {
		//检查是否需要对本 block编码
		if buf := v.Encoding(); buf != nil {
			buffer.Write(buf)
		}
	}
	buffer.WriteString(b.content + CRLF)
	return buffer.Bytes()
}

func (b *Body) Write(block ...*Block) {
	if b.text == nil {
		b.text = make([]*Block, len(block))
	}
	b.text = append(b.text, block...)
}

func (b *Body) Message() []byte {
	buffer := bytes.Buffer{}
	for _, h := range b.must {
		for _, v := range h.Value { //遍历Content-Type 是否包含 multipart/mixed
			//解析该邮件是否是混合邮件
			if strings.HasPrefix(v, "boundary") {
				b.boundary = CRLF + "--" + v[9:] //解析边界分割
				//h.Value[i] += CRLF + b.boundary + CRLF
			}
		}
		if buf := h.Encoding(); buf != nil {
			buffer.Write(buf)
		}
	}

	tlen := len(b.text)
	for i := 0; i < tlen; i++ {
		if b.boundary != "" {
			buffer.Write([]byte(b.boundary + CRLF))
		}
		if buf := b.text[i].Message(); buf != nil {
			buffer.Write(buf)
		}
		if i == tlen-1 {
			buffer.Write([]byte(b.boundary + "--" + CRLF)) //标记结尾
		}
	}
	return buffer.Bytes()
}
