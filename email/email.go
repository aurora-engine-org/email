package email

import (
	"bytes"
	"encoding/base64"
	"errors"
	"net/smtp"
	"strings"
	"time"
)

// File 邮件附件信息
type File struct {
	Filename         string   //文件名称
	Type             []string //内容类型
	Disposition      []string //文件描述
	TransferEncoding string   //传输编码
	Encoding         string   //编码后的字符串
	data             []byte   //文件字节
}

type Client struct {
	host     string
	username string
	password string
	auth     smtp.Auth
	boundary string //标识该邮件是否为混合消息体，如果是混合消息，则需要用此参数去初始化第一个message的边界符号
	from     string //发件人信息

	//一下为邮件内容设置
	subject string           //邮件主题
	main    *Message         //用于初始化构建消息
	html    string           //html消息 和 text 消息 在不同的平台上的兼容性存在一定的差异，同附件一起传输存在一定的bug，建议和附件传输经可能采用text
	text    string           //文本消息
	file    map[string]*File //文件信息
}

// Subject 设置邮件标题信息
func (c *Client) Subject(title string) {
	c.subject = title
}

// Text 设置发送的文本信息
func (c *Client) Text(text string) {
	c.text = text
}

// Html 设置发送的超文本信息
func (c *Client) Html(html string) {
	c.html = html
}

// File 设置发送邮件的附件信息
// 文件默认的传输格式采用Base64
// File 可以多次添加并不会产生覆盖
func (c *Client) File(name string, data []byte) {
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

// SendEmail 发送邮件信息 可选多个地址
func (c *Client) SendEmail(addr ...string) (bool, error) {
	if addr == nil || len(addr) == 0 {
		return false, errors.New("pass at least one address information")
	}
	c.build()
	if c.main == nil {
		return false, errors.New("email content is empty")
	}
	c.main.header = append(c.main.header[:1], append([]*Header{NewHeader(To, addr...)}, c.main.header[1:]...)...) //设置收件人信息
	message := parseMessage(c.main)                                                                               //开始解析消息体
	if message == nil {
		return false, errors.New("email content is empty")
	}
	err := smtp.SendMail(c.host+":25", c.auth, c.from, addr, message)
	if err != nil {
		return false, err
	}
	//清空内容
	c.text = ""
	c.html = ""
	c.file = nil
	return true, nil
}

// 构建消息, 该解析 暂时对一个消息的嵌套多个同级消息做支持
func (c *Client) build() {
	message := &Message{
		header: []*Header{
			NewHeader(From, c.from),
			NewHeader(Data, time.Now().Format("2006/01/02 15:04:05")),
			NewHeader(Subject, c.subject),
			defaultHeader["MIME"],
		},
	}
	if (c.text != "" && c.html != "" && c.file != nil) || (c.text != "" && c.html != "") || (c.html != "" && c.file != nil) || (c.text != "" && c.file != nil) {
		//设置多媒体消息混合头,此处待后续修改解析，以支持 普通文本，当前默认只支持 html(alternative)兼容性相对好一写,   text(media)
		//此处的 boundary 可以采取随机生成，根据 https://www.rfc-editor.org/rfc/rfc2046#section-5.1.1 中的要素，不要携带特殊符号即可，此处暂时固定不变，后续需要在进行调整
		message.header = append(message.header, NewHeader(ContentType, "multipart/alternative", "boundary=main body"))
	}
	//开始封装 文本 超文本 以及文件 消息
	if c.text != "" {
		text := &Message{
			header: []*Header{
				NewHeader(ContentType, "text/plain", "charset=utf-8"),
				NewHeader(ContentTransferEncoding, "quoted-printable"+CRLF), //此处的 CRLF作为多部份混合所必须要的
			},
			body: c.text,
		}
		message.Next(text)
	}

	if c.html != "" {
		html := &Message{
			header: []*Header{
				NewHeader(ContentType, "text/html", "charset=utf-8"),
				NewHeader(ContentTransferEncoding, "quoted-printable"+CRLF), //此处的 CRLF作为多部份混合所必须要的
			},
			body: c.html,
		}
		message.Next(html)
	}

	//此处的对多个附件的处理方式，全部放在最末尾
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
			buf.WriteString(CRLF + message.boundaryStart + CRLF) //每个嵌套的消息模块采用边界分割开来
			s := parseMessage(message.msg[i])
			buf.Write(s)
		}
	}
	if message.boundaryEed != "" {
		buf.WriteString(CRLF + message.boundaryEed + CRLF)
	}
	return buf.Bytes()
}

// NewClient 生成一个Email客户端
func NewClient(user, password, host string) *Client {
	c := &Client{host: host, username: user, password: password, from: user}
	c.auth = smtp.PlainAuth("", user, password, host)
	return c
}
