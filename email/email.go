// Copyright 2022 Hal Canary
// Use of this program is governed by the file LICENSE.
package email

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/http"
	"net/mail"
	"net/smtp"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/HalCanary/booker/humanize"
)

const crlf = "\r\n"

// Data structure representing instructions for connecting to SMTP server.
// Headers are additional headers to be added to outgoing email.
type EmailSecrets struct {
	SmtpHost string            // example: "smtp.gmail.com"
	SmtpUser string            // example: "halcanary@gmail.com"
	SmtpPass string            // for gmail, is a App Password
	FromAddr string            // example: "Hal Canary <halcanary@gmail.com>"
	Headers  map[string]string // extra headers to be added to email.
}

// Attachment for an email.
type Attachment struct {
	Filename    string
	ContentType string // If empty, determined via http.DetectContentType.
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

func encodeHeader(out io.StringWriter, key, s string) {
	if s == "" {
		return
	}
	out.WriteString(key)
	out.WriteString(": ")
	for s != "" {
		word, next, found := strings.Cut(s, " ")
		s = next
		out.WriteString(mime.QEncoding.Encode("utf-8", word))
		if found {
			out.WriteString(" ")
		}
	}
	out.WriteString(crlf)
}

// Make, but do not send an email message.
func (mail Email) Make() []byte {
	const boundary = "================"
	const mixedContentType = "multipart/mixed; boundary=\"" + boundary + "\""
	var buffer bytes.Buffer
	if mail.Date.IsZero() {
		mail.Date = time.Now()
	}
	encodeHeader(&buffer, "Date", mail.Date.Format(time.RFC1123Z))
	encodeHeader(&buffer, "Subject", mail.Subject)
	encodeHeader(&buffer, "From", mail.From)
	for _, to := range mail.To {
		encodeHeader(&buffer, "To", to)
	}
	for _, cc := range mail.Cc {
		encodeHeader(&buffer, "Cc", cc)
	}
	for key, value := range mail.Headers {
		encodeHeader(&buffer, key, value)
	}
	encodeHeader(&buffer, "MIME-Version", "1.0")
	encodeHeader(&buffer, "Content-Type", mixedContentType)
	buffer.WriteString(crlf) // end of header

	mw := multipart.NewWriter(&buffer)
	mw.SetBoundary(boundary)
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

// Send a file to a single destination.
func SendFile(dst, path, contentType string, secrets EmailSecrets) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	base := filepath.Base(path)
	subject := fmt.Sprintf("(%s) %s", humanize.Humanize(len(data)), base)
	return Email{
		From:    secrets.FromAddr,
		To:      []string{dst},
		Subject: subject,
		Content: "☺",
		Attachments: []Attachment{
			Attachment{
				Data:        data,
				ContentType: contentType,
				Filename:    base,
			},
		},
	}.Send(secrets)
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
	endofline := []byte(crlf)
	const linelength = 57
	const bufferlength = 78 // base64.StdEncoding.EncodedLen(57) + 2
	var buffer [bufferlength]byte
	for len(src) > 0 {
		l := len(src)
		if l > linelength {
			l = linelength
		}
		el := base64.StdEncoding.EncodedLen(l)
		base64.StdEncoding.Encode(buffer[:el], src[:l])
		src = src[l:]
		copy(buffer[el:el+2], endofline)
		dst.Write(buffer[:el+2])
	}
}
