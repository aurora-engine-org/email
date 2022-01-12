package email

import (
	"encoding/base64"
	"fmt"
	"github.com/awensir/project/times"
	"io"
	"log"
	"net/mail"
	"net/smtp"
	"os"
	"strings"
	"testing"
)

func TestEmail(t *testing.T) {
	msg := `Date: Mon, 23 Jun 2015 11:40:36 -0400
From: Gopher <from@example.com>
To: Another Gopher <to@example.com>
Subject: Gophers at Gophercon

Message body
`
	r := strings.NewReader(msg)
	m, err := mail.ReadMessage(r)
	if err != nil {
		log.Fatal(err)
	}

	header := m.Header
	fmt.Println("Date:", header.Get("Date"))
	fmt.Println("From:", header.Get("From"))
	fmt.Println("To:", header.Get("To"))
	fmt.Println("Subject:", header.Get("Subject"))

	body, err := io.ReadAll(m.Body)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s", body)
}

//地址解析

func TestAddress(t *testing.T) {
	e, err := mail.ParseAddress("Alice <alice@example.com>")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("name: ", e.Name, "address: ", e.Address)
}

func TestAddressList(t *testing.T) {
	const list = "Alice <alice@example.com>, Bob <bob@example.com>, Eve <eve@example.com>"
	emails, err := mail.ParseAddressList(list)
	if err != nil {
		log.Fatal(err)
	}

	for _, v := range emails {
		fmt.Println("name: ", v.Name, "address: ", v.Address)
	}
}

func TestClient_Send(t *testing.T) {
	//	email := NewEmail("zhiwen_der@qq.com", "qwjuttqhgsfnbaad", "smtp.qq.com")
	//	msg := Message{
	//		Subject: []string{"test_subject"},
	//		From:    []string{"test", "zhiwen_der@qq.com"},
	//		To:      []string{"3391853042@qq.com"},
	//		Data:    []string{times.TimeFormat(times.YYYY_MM_DD_HH_MM_SS_1)},
	//		Header:  []string{"Content-Type: text/html; charset=UTF-8", "Content-Transfer-Encoding: quoted-printable"},
	//		Body: []string{`
	//<!DOCTYPE html>
	//<html lang="en">
	//<head>
	//   <meta http-equiv="Content-Type" content="text/html;charset=utf-8" />
	//   <title>Title</title>
	//</head>
	//<body>
	//<p style="font-size:30px;color:orange">p标签设置字体颜色</p>
	//</body>
	//</html>
	//`},
	//	}
	//	err := email.Send(msg)
	//	if err != nil {
	//		log.Fatal(err.Error())
	//		return
	//	}
}

// 生成 HTML 信息测试
func NewHtmlBlock() *Block {
	block := &Block{}
	html := NewHeader(ContentType, "text/html", "charset=UTF-8")
	encoding := NewHeader(ContentTransferEncoding, "quoted-printable")
	block.SetHeader(html, encoding)
	block.SetContent(`
	<!DOCTYPE html>
	<html lang="en">
		<head>
           <meta http-equiv="Content-Type" content="text/html;charset=utf-8" />
		   <title>Title</title>
		</head>
		<body>
			<p style="font-size:30px;color:orange">p标签设置字体颜色</p>
		</body>
	</html>
`)
	return block
}

// 生成pdf附件信息测试
func NewPdfBlock() *Block {
	block := &Block{}
	pdf := NewHeader(ContentType, "application/pdf", "name=test.pdf", "boundary=\"--123--\"")
	encoding := NewHeader(ContentTransferEncoding, "Base64")
	n := NewHeader("Content-Disposition", "attachment", "filename=test.pdf")
	block.SetHeader(pdf, encoding, n)
	file, err := os.ReadFile("E:\\space\\src\\project\\email\\1111.pdf")
	if err != nil {
		return nil
	}
	toString := base64.StdEncoding.EncodeToString(file)
	fmt.Println("附件编码")
	fmt.Println(toString)
	block.SetContent(toString)
	return block
}

func NewTextHtml() []*Block {
	text := &Block{}
	//texth := NewHeader(ContentType, "text/plain", "charset=UTF-8")
	//tencoding := NewHeader(ContentTransferEncoding, "quoted-printable")
	//text.SetHeader(texth, tencoding)
	text.SetContent("测试文本信息")

	block := &Block{}
	html := NewHeader(ContentType, "text/html", "charset=UTF-8")
	htmlh := NewHeader(ContentTransferEncoding, "quoted-printable")
	block.SetHeader(html, htmlh)
	block.SetContent(`
	<!DOCTYPE html>
	<html lang="en">
		<head>
           <meta http-equiv="Content-Type" content="text/html;charset=utf-8" />
		   <title>Title</title>
		</head>
		<body>
			<p style="font-size:30px;color:orange">测试HTML 信息</p>
		</body>
	</html>
`)
	return []*Block{text, block}
}

func TestSendBody(t *testing.T) {
	//from := NewHeader(From, "zhiwen_der@qq.com")
	//to := NewHeader(To, "1219449282@qq.com")
	//data := NewHeader(Data, times.TimeFormat(times.YYYY_MM_DD_HH_MM_SS_1))
	//subject := NewHeader(Subject, "test")
	//c := NewHeader(ContentType, "multipart/mixed", "boundary=main body")
	//
	//body := NewBody([]*Header{from, to, data, subject, defaultHeader["MIME"], c}, NewTextHtml()...)
	//message := body.Message()
	//fmt.Println(string(message))
	email := NewEmail("zhiwen_der@qq.com", "qwjuttqhgsfnbaad", "smtp.qq.com")
	email.Text("test 普通文本消息")
}

func TestBuild(t *testing.T) {

	fmt.Println(BuildEmile())
}

func BuildEmile() []byte {
	file, err := os.ReadFile("E:\\space\\src\\project\\email\\1111.pdf")
	if err != nil {
		return nil
	}
	toString := base64.StdEncoding.EncodeToString(file)
	message := &Message{
		header: []*Header{
			NewHeader(From, "zhiwen_der@qq.com"),
			NewHeader(To, "1219449282@qq.com"),
			NewHeader(Data, times.TimeFormat(times.YYYY_MM_DD_HH_MM_SS_1)),
			NewHeader(Subject, "test"),
			defaultHeader["MIME"],
			NewHeader(ContentType, "multipart/alternative", "boundary=main body"),
		},
		msg: []*Message{
			&Message{
				header: []*Header{
					NewHeader(ContentType, "text/plain", "charset=gb2312"),
					//NewHeader(ContentTransferEncoding, "quoted-printable"),
				},
				body: "嵌套TEXT 信息",
			},
			&Message{
				header: []*Header{
					NewHeader(ContentType, "text/html", "charset=UTF-8"),
					//NewHeader(ContentTransferEncoding, "quoted-printable"),
				},
				body: `<!DOCTYPE html>
<html lang="en">
<head>
<meta http-equiv="Content-Type" content="text/html;charset=utf-8" />
<title>Title</title>
</head>
<body>
<p style="font-size:30px;color:orange">嵌套HTML 信息</p>
</body>
</html>
`,
			},
			&Message{
				header: []*Header{
					NewHeader(ContentType, "application/pdf", "name=test.pdf"),
					NewHeader(ContentTransferEncoding, "Base64"),
					NewHeader("Content-Disposition", "attachment", "filename=test.pdf"),
				},
				body: toString,
			},
			&Message{
				header: []*Header{
					NewHeader(ContentType, "application/octet-stream", "name=test2.pdf"),
					NewHeader(ContentTransferEncoding, "Base64"),
					NewHeader("Content-Disposition", "attachment", "filename=test2.pdf"),
				},
				body: toString,
			},
		},
	}
	return parseMessage(message)
}

func TestSendEmail(t *testing.T) {

	auth := smtp.PlainAuth("", "zhiwen_der@qq.com", "qwjuttqhgsfnbaad", "smtp.qq.com")

	to := []string{"1219449282@qq.com"} //收件地址
	s := BuildEmile()
	fmt.Println(s)
	err := smtp.SendMail("smtp.qq.com:25", auth, "zhiwen_der@qq.com", to, s)
	if err != nil {
		log.Fatal(err)
	}
}

func TestText(t *testing.T) {
	email := NewEmail("zhiwen_der@qq.com", "qwjuttqhgsfnbaad", "smtp.qq.com")
	email.Subject("test")
	//email.Text("test 普通文本消息")
	email.Html(`<!DOCTYPE html>
	<html>
	<body>
	<p style="font-size:30px;color:orange">测试HTML 信息</p>
	</body>
	</html>`)
	//file, err := os.ReadFile("E:\\space\\src\\project\\email\\1111.pdf")
	//if err != nil {
	//	fmt.Println(err.Error())
	//	return
	//}
	//email.File("test.pdf", file)
	_, err := email.SendEmail("1219449282@qq.com")
	if err != nil {
		fmt.Println(err.Error())
		return
	}
}
