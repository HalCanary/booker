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
	out.WriteString("\n")
}

// Make, but do not send an email message.
func (mail Email) Make() []byte {
	const boundary = "================"
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
	mixedContentType := fmt.Sprintf("multipart/mixed; boundary=%q", boundary)
	encodeHeader(&buffer, "Content-Type", mixedContentType)
	buffer.WriteString("\n") // end of header

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
