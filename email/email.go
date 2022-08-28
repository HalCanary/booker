package email

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"mime/quotedprintable"
	"net/http"
	"net/mail"
	"net/smtp"
	"net/textproto"
	"os"
	"time"
)

// Data structure representing instructions for connecting to SMTP server.
// Headers are additional headers to be added to outgoing email.
type EmailSecrets struct {
	SmtpHost string            `json:"SMTP_HOST"` // example: "smtp.gmail.com"
	SmtpUser string            `json:"SMTP_USER"` // example: "halcanary@gmail.com"
	SmtpPass string            `json:"SMTP_PASS"` // for gmail, is a App Password
	FromAddr string            `json:"FROM_ADDR"`
	Headers  map[string]string `json:"HEADERS"`
}

// Attachment for an email.
type Attachment struct {
	Filename    string
	ContentType string
	Data        []byte
}

// An electric mail message.
type Email struct {
	Date        time.Time
	To          []string
	Cc          []string
	Bcc         []string
	From        string
	Subject     string
	Content     string
	Attachments []Attachment
	Headers     map[string]string
}

// Read email secrets from the given file.
func GetSecrets(path string) (EmailSecrets, error) {
	var v EmailSecrets
	b, err := os.ReadFile(path)
	if err == nil {
		err = json.Unmarshal(b, &v)
	}
	return v, err
}

// Send the given email using the provided SMTP secrets.
func (m Email) Send(secrets EmailSecrets) error {
	to := append(append(m.To, m.Cc...), m.Bcc...)
	for i, a := range to {
		addr, err := mail.ParseAddress(a)
		if err != nil {
			return err
		}
		to[i] = addr.Address
	}
	msg := m.Make()
	auth := smtp.PlainAuth("", secrets.SmtpUser, secrets.SmtpPass, secrets.SmtpHost)
	return smtp.SendMail(secrets.SmtpHost+":587", auth, secrets.SmtpUser, to, msg)
}

// Make, but do not send an email message.
func (mail Email) Make() []byte {
	var buffer bytes.Buffer
	if mail.Date.IsZero() {
		mail.Date = time.Now()
	}
	wb(&buffer, "Date: ", mail.Date.Format(time.RFC1123Z), "\n")
	if mail.Subject != "" {
		wb(&buffer, "Subject: ", mail.Subject, "\n")
	}
	if mail.From != "" {
		wb(&buffer, "From: ", mail.From, "\n")
	}
	for _, to := range mail.To {
		wb(&buffer, "To: ", to, "\n")
	}
	for _, cc := range mail.Cc {
		wb(&buffer, "Cc: ", cc, "\n")
	}
	for key, value := range mail.Headers {
		wb(&buffer, key, ": ", value, "\n")
	}
	wb(&buffer,
		"MIME-Version: 1.0\n",
		"Content-Type: multipart/mixed; boundary=\"================\"\n\n",
	)
	mw := multipart.NewWriter(&buffer)
	mw.SetBoundary("================")
	if mail.Content != "" {
		w, _ := mw.CreatePart(textproto.MIMEHeader{
			"Content-Type":              []string{"text/plain; charset=\"UTF-8\""},
			"Content-Transfer-Encoding": []string{"quoted-printable"},
		})
		quotedprintableWrite(mail.Content, w)
	}
	for _, attachment := range mail.Attachments {
		contentType := attachment.ContentType
		if contentType == "" {
			contentType = http.DetectContentType(attachment.Data)
		}
		w, _ := mw.CreatePart(textproto.MIMEHeader{
			"Content-Type":              []string{contentType},
			"Content-Transfer-Encoding": []string{"base64"},
			"Content-Disposition":       []string{contentDisposition(attachment.Filename)},
			"MIME-Version":              []string{"1.0"},
		})
		base64Write(attachment.Data, w)
	}
	mw.Close()
	return buffer.Bytes()
}

func wb(buffer *bytes.Buffer, strings ...string) {
	for _, s := range strings {
		buffer.Write([]byte(s))
	}
}

func contentDisposition(filename string) string {
	if filename == "" {
		return "attachment"
	}
	return fmt.Sprintf("attachment; filename=%q", filename)
}

func quotedprintableWrite(src string, dst io.Writer) {
	qpw := quotedprintable.NewWriter(dst)
	qpw.Write([]byte(src))
	qpw.Close()
}

func base64Write(src []byte, dst io.Writer) {
	var bb bytes.Buffer
	encoder := base64.NewEncoder(base64.StdEncoding, &bb)
	encoder.Write(src)
	encoder.Close()
	line := [76]byte{}
	for {
		n, _ := bb.Read(line[:])
		if n == 0 {
			break
		}
		dst.Write(line[:n])
		dst.Write([]byte{'\n'})
	}
}
