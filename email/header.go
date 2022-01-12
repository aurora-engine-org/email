package email

import "strings"

const (
	CRLF                    = "\r\n"                      //解析分隔标记
	Subject                 = "Subject"                   //邮件主题
	From                    = "From"                      //发件人信息
	To                      = "To"                        //收件人信息
	Data                    = "Data"                      //指定邮件发送时间
	ContentType             = "Content-Type"              //指定消息类型
	ContentTransferEncoding = "Content-Transfer-Encoding" //指定消息编码格式
	ContentDisposition      = "Content-Disposition"
	MIME                    = "MIME-Version"
	Base64                  = "Base64"
)

// defaultHeader 常用的内置头类型
var defaultHeader = map[string]*Header{
	"MIME":         NewHeader(MIME, "1.0"), //MIME 版本信息
	"HTML":         NewHeader(ContentType, "text/html", "charset=UTF-8"),
	"TEXT":         NewHeader(ContentType, "text/plain", "charset=UTF-8"),
	"TextEncoding": NewHeader(ContentTransferEncoding, "quoted-printable"),
	"HtmlEncoding": NewHeader(ContentTransferEncoding, "quoted-printable"),
	"FileEncoding": NewHeader(ContentTransferEncoding, Base64),
}

// Header Email 头信息结构
type Header struct {
	Name  string   //头名
	Value []string //头的内容属性
}

// NewHeader 创建新的 Header
func NewHeader(name string, attr ...string) *Header {
	h := &Header{
		Name:  name,
		Value: attr,
	}
	return h
}

// AddAttr 设置 Header 的属性内容
func (h *Header) AddAttr(name string, value ...string) {
	if h.Value == nil {
		return
	}
	h.Value = append(h.Value, value...)
}

// Encoding Header 编码生成指定格式的消息组合部分
func (h *Header) Encoding() []byte {
	if h.Value == nil || len(h.Value) == 0 {
		return nil
	}
	s := h.Name + ": "
	s += strings.Join(h.Value, ";") + CRLF
	return []byte(s)
}
