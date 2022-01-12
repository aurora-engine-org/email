package email

import (
	"bytes"
	"encoding/base64"
	"errors"
	"net/smtp"
	"strings"
	"time"
)

/*

 */

type Email interface {
	// Send 发送邮件
	// Body 需要发送的消息体，...string 则是发送的地址信息
	Send(*Body, ...string) error
}

// File 邮件附件信息
type File struct {
	Filename         string   //文件名称
	Type             []string //内容类型
	Disposition      []string //文件描述
	TransferEncoding string   //传输编码
	Encoding         string   //编码后的字符串
	data             []byte   //文件字节
}

type client struct {
	host     string
	username string
	password string
	auth     smtp.Auth
	boundary string //标识该邮件是否为混合消息体，如果是混合消息，则需要用此参数去初始化第一个message的边界符号
	from     string //发件人信息

	//一下为邮件内容设置
	subject string           //邮件主题
	main    *Message         //用于初始化构建消息
	html    string           //html消息
	text    string           //文本消息
	file    map[string]*File //文件信息
}

func (c *client) Subject(title string) {
	c.subject = title
}

// Text 设置发送的文本信息
func (c *client) Text(text string) {
	c.text = text
}

// Html 设置发送的超文本信息
func (c *client) Html(html string) {
	c.html = html
}

// File 设置发送邮件的附件信息
// 文件默认的传输格式采用Base64
func (c *client) File(name string, data []byte) {
	if c.file == nil {
		c.file = make(map[string]*File)
	}
	encodeToString := base64.StdEncoding.EncodeToString(data)
	file := &File{
		Filename:         name,
		Type:             []string{"application/octet-stream", "name=" + name},
		Disposition:      []string{"attachment", "filename=" + name + CRLF},
		TransferEncoding: Base64,
		data:             data,
		Encoding:         encodeToString,
	}
	c.file[name] = file
}

func (c *client) SendEmail(addr ...string) (bool, error) {
	c.build()
	if c.main == nil {
		return false, errors.New("email content is empty")
	}
	c.main.header = append(c.main.header[:1], append([]*Header{NewHeader(To, addr...)}, c.main.header[1:]...)...)
	message := parseMessage(c.main)
	if message == nil {
		return false, errors.New("email content is empty")
	}
	err := smtp.SendMail(c.host+":25", c.auth, c.from, addr, message)
	if err != nil {
		return false, err
	}
	return true, nil
}

// 构建消息, 该解析 暂时对一个消息的嵌套多个同级消息做支持
func (c *client) build() {
	message := &Message{
		header: []*Header{
			NewHeader(From, c.from),
			NewHeader(Data, time.Now().Format("2006/01/02 15:04:05")),
			NewHeader(Subject, c.subject),
			defaultHeader["MIME"],
		},
	}
	if (c.text != "" && c.html != "" && c.file != nil) || (c.text != "" && c.html != "") || (c.html != "" && c.file != nil) || (c.text != "" && c.file != nil) {
		//设置多媒体消息混合头,此处待后续修改解析，以支持 普通文本，当前默认只支持 html(alternative) text(media)
		message.header = append(message.header, NewHeader(ContentType, "multipart/alternative", "boundary=main body"))
	}
	//开始封装 文本 超文本 以及文件 消息
	if c.text != "" {
		text := &Message{
			header: []*Header{
				NewHeader(ContentType, "text/plain", "charset=utf-8"),
				NewHeader(ContentTransferEncoding, "quoted-printable"+CRLF),
			},
			body: c.text + CRLF,
		}
		message.Next(text)
	}

	if c.html != "" {
		html := &Message{
			header: []*Header{
				NewHeader(ContentType, "text/html", "charset=utf-8"),
				NewHeader(ContentTransferEncoding, "quoted-printable"+CRLF),
			},
			body: c.html,
		}
		message.Next(html)
	}

	if c.file != nil {
		for _, v := range c.file {
			f := &Message{
				header: []*Header{
					NewHeader(ContentType, v.Type...),
					NewHeader(ContentTransferEncoding, v.TransferEncoding),
					NewHeader(ContentDisposition, v.Disposition...),
				},
				body: v.Encoding,
			}
			message.Next(f)
		}
	}
	c.main = message
}

// Message 邮件中的消息部分
type Message struct {
	boundaryStart string     //边界分割
	header        []*Header  //消息头
	body          string     //正文,一般为字符串，HTML，或者编码后的字符串
	msg           []*Message //嵌套消息
	boundaryEed   string     //边界结尾
}

// 附加一个消息
func (m *Message) Next(msg *Message) {
	if m.msg == nil {
		m.msg = make([]*Message, 0)
	}
	m.msg = append(m.msg, msg)
}

// 对消息进行解析，生成最终传输的消息体，该解析实现了 rfc2046 的多部份消息混合解析
func parseMessage(message *Message) []byte {
	buf := &bytes.Buffer{}

	//解析邮件边界分割
	if message.boundaryStart != "" && message.body != "" { //如果当前 message 没有任何消息正文，则不添加分割符
		buf.WriteString(CRLF + message.boundaryStart + CRLF)
	}
	//解析邮件头 检验本身是否属于一个多部份混合消息,如果是多部份混合消息 则自己本身也要使用自己的消息分割符
	if message.header != nil {
		for i := 0; i < len(message.header); i++ {
			if message.header[i].Name == ContentType { //检索是否有内容类型头
				for _, value := range message.header[i].Value { //检索是否需要分段
					if strings.HasPrefix(value, "boundary") {
						//如果有该属性，那么后续的 msg           []*Message //嵌套消息 都需要使用这个作为分割
						//message.boundaryStart = "--" + value[9:]
						message.boundaryEed = "--" + value[9:] + "--"
						if message.msg != nil {
							for j := 0; j < len(message.msg); j++ {
								message.msg[j].boundaryStart = "--" + value[9:] //此处初始化被嵌套的消息起始分割符号
							}
						}
					}
				}
			}
			buf.Write(message.header[i].Encoding())
		}
	}

	if message.body != "" {
		buf.WriteString(message.body)
	}

	if message.msg != nil {
		// 开始递归解析嵌套消息
		for i := 0; i < len(message.msg); i++ {
			s := parseMessage(message.msg[i])
			buf.Write(s)
		}
	}
	if message.boundaryEed != "" {
		buf.WriteString(CRLF + message.boundaryEed + CRLF)
	}
	return buf.Bytes()
}

func (c *client) Send(body *Body, addr ...string) error {
	err := smtp.SendMail(c.host+":25", c.auth, c.from, addr, body.Message())
	if err != nil {
		return err
	}
	return nil
}

// NewEmail 生成一个Email客户端
func NewEmail(user, password, host string) *client {
	c := &client{host: host, username: user, password: password, from: user}
	c.auth = smtp.PlainAuth("", user, password, host)
	return c
}
