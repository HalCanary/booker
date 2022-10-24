package email

// Copyright 2022 Hal Canary
// Use of this program is governed by the file LICENSE.

import (
	"bytes"
	"io"
	"mime"
	"mime/quotedprintable"
	"net/mail"
	"reflect"
	"strings"
	"testing"
	"time"
)

const testdata = `Lorem ipsum dolor sit amet, consectetur adipiscing elit.  Nunc imperdiet elit
eu sapien accumsan, quis accumsan nisl porta. Donec congue dui in dignissim
tincidunt. Phasellus vel ligula lobortis tortor iaculis vulputate. Proin non
augue quis est molestie dignissim eget sit amet enim.  Donec ut purus ac enim
hendrerit ornare. Pellentesque egestas tempor sodales.  Pellentesque eget
auctor mauris.`

const expected = `Date: Sat, 01 Jan 2022 00:00:00 +0000
Subject: a quick note =?utf-8?q?(=E2=99=A0=E2=99=A5=E2=99=A6=E2=99=A3)?=
From: =?utf-8?q?Z_=E2=86=90=E2=86=91=E2=86=92=E2=86=93?= <z@example.com>
To: "A" <a@example.com>,
 "B" <b@example.com>
Cc: "C" <c@example.com>,
 "D" <d@example.com>
Mime-Version: 1.0
Content-Type: multipart/mixed; boundary="================"

--================
Content-Transfer-Encoding: quoted-printable
Content-Type: text/plain; charset="UTF-8"

Hello, World!
--================
Content-Disposition: attachment; filename="foo.txt"
Content-Transfer-Encoding: base64
Content-Type: text/plain; charset=utf-8
MIME-Version: 1.0

TG9yZW0gaXBzdW0gZG9sb3Igc2l0IGFtZXQsIGNvbnNlY3RldHVyIGFkaXBpc2NpbmcgZWxpdC4g
IE51bmMgaW1wZXJkaWV0IGVsaXQKZXUgc2FwaWVuIGFjY3Vtc2FuLCBxdWlzIGFjY3Vtc2FuIG5p
c2wgcG9ydGEuIERvbmVjIGNvbmd1ZSBkdWkgaW4gZGlnbmlzc2ltCnRpbmNpZHVudC4gUGhhc2Vs
bHVzIHZlbCBsaWd1bGEgbG9ib3J0aXMgdG9ydG9yIGlhY3VsaXMgdnVscHV0YXRlLiBQcm9pbiBu
b24KYXVndWUgcXVpcyBlc3QgbW9sZXN0aWUgZGlnbmlzc2ltIGVnZXQgc2l0IGFtZXQgZW5pbS4g
IERvbmVjIHV0IHB1cnVzIGFjIGVuaW0KaGVuZHJlcml0IG9ybmFyZS4gUGVsbGVudGVzcXVlIGVn
ZXN0YXMgdGVtcG9yIHNvZGFsZXMuICBQZWxsZW50ZXNxdWUgZWdldAphdWN0b3IgbWF1cmlzLg==

--================--
`

func TestEmail(t *testing.T) {
	mail := Email{
		Date:    time.Date(2022, time.January, 1, 0, 0, 0, 0, time.UTC),
		To:      []Address{Address{"A", "a@example.com"}, Address{"B", "b@example.com"}},
		Cc:      []Address{Address{"C", "c@example.com"}, Address{"D", "d@example.com"}},
		Bcc:     []Address{Address{"E", "e@example.com"}, Address{"F", "f@example.com"}},
		From:    Address{"Z ←↑→↓", "z@example.com"},
		Subject: "a quick note (♠♥♦♣)",
		Content: "Hello, World!",
		Attachments: []Attachment{
			Attachment{
				Filename: "foo.txt",
				Data:     []byte(testdata),
			},
		},
		Headers: map[string]string{},
	}
	expectedS := strings.ReplaceAll(expected, "\n", "\r\n")
	var buffer bytes.Buffer
	mail.Make(&buffer)
	result := buffer.String()
	if result != expectedS {
		t.Errorf("\n%s\n!=\n%s\n", result, expectedS)
	}
}

const testmessage2 = "Lorem ipsum dolor sit amet, consectetur adipiscing elit.  Nunc imperdiet elit eu sapien accumsan, quis accumsan nisl porta. Donec congue dui in dignissim tincidunt. Phasellus vel ligula lobortis tortor iaculis vulputate.\n\nProin non augue quis est molestie dignissim eget sit amet enim.  Donec ut purus ac enim hendrerit ornare. Pellentesque egestas tempor sodales.  Pellentesque eget auctor mauris."

const expected2 = `Date: Sat, 01 Jan 2022 00:00:00 +0000
Subject: test2
From: "Z" <z@example.com>
To: "A" <a@example.com>
Mime-Version: 1.0
Content-Type: text/plain; charset="UTF-8"
Content-Transfer-Encoding: quoted-printable

Lorem ipsum dolor sit amet, consectetur adipiscing elit.  Nunc imperdiet el=
it eu sapien accumsan, quis accumsan nisl porta. Donec congue dui in dignis=
sim tincidunt. Phasellus vel ligula lobortis tortor iaculis vulputate.

Proin non augue quis est molestie dignissim eget sit amet enim.  Donec ut p=
urus ac enim hendrerit ornare. Pellentesque egestas tempor sodales.  Pellen=
tesque eget auctor mauris.
`

func TestEmail2(t *testing.T) {
	m := Email{
		Date:    time.Date(2022, time.January, 1, 0, 0, 0, 0, time.UTC),
		To:      []Address{Address{"A", "a@example.com"}},
		From:    Address{"Z", "z@example.com"},
		Subject: "test2",
		Content: testmessage2,
		Headers: map[string]string{},
	}
	expectedS2 := strings.ReplaceAll(expected2, "\n", "\r\n")
	var buffer bytes.Buffer
	m.Make(&buffer)
	result := buffer.String()
	if result != expectedS2 {
		t.Fatalf("\n%q\n!=\n%q\n", result, expectedS2)
	}
	//		expectedHeader := mail.Header{
	//				"Content-Transfer-Encoding":[]string{"quoted-printable"},
	//				"Content-Type":[]string{"text/plain; charset=\"UTF-8\""},
	//				"Date":[]string{"Sat, 01 Jan 2022 00:00:00 +0000"},
	//				"From":[]string{"\"Z\" <z@example.com>"},
	//				"Mime-Version":[]string{"1.0"},
	//				"Subject":[]string{"test2"},
	//				"To":[]string{"\"A\" <a@example.com>"},
	//			}
	//	 || !reflect.DeepEqual(&expectedHeader, &(msg.Header))
}

func addressList(header mail.Header, key string) ([]Address, error) {
	var result []Address
	list, err := header.AddressList(key)
	if err == nil {
		result = make([]Address, 0, len(list))
		for _, a := range list {
			if a != nil {
				result = append(result, *a)
			}
		}
	}
	return result, err
}

func TestEmail3(t *testing.T) {
	m := Email{
		Date:    time.Date(2022, time.January, 1, 0, 0, 0, 0, time.UTC),
		To:      []Address{Address{"A ← ↑ → ↓", "a@example.com"}},
		From:    Address{"Z ← ↑ → ↓", "z@example.com"},
		Subject: "test2 ←↑ →↓",
		Content: testmessage2,
		Headers: map[string]string{},
	}
	e, _ := decodeMessage(encodeBytes(m))

	expectEqualDate(t, m.Date, e.Date)
	expectEqual(t, m.To, e.To)
	expectEqual(t, m.From, e.From)
	expectEqual(t, m.Cc, e.Cc)
	expectEqual(t, m.Subject, e.Subject)
	expectEqual(t, m.Content, e.Content)
	expectEqual(t, m.Attachments, e.Attachments)
	expectEqualMap(t, m.Headers, e.Headers)
	if !e.Equal(m) {
		t.Error()
	}
}

func equalAddressArrays(u, v []Address) bool {
	if len(u) != len(v) {
		return false
	}
	for i, a := range u {
		if a != v[i] {
			return false
		}
	}
	return true
}

func (e Email) Equal(m Email) bool {
	return e.Date.Equal(m.Date) &&
		equalAddressArrays(e.To, m.To) &&
		equalAddressArrays(e.Cc, m.Cc) &&
		equalAddressArrays(e.Bcc, m.Bcc) &&
		e.From == m.From &&
		e.Subject == m.Subject &&
		e.Content == m.Content
}

func encodeBytes(m Email) []byte {
	var buffer bytes.Buffer
	m.Make(&buffer)
	return buffer.Bytes()
}

func expectEqual(t *testing.T, u, v interface{}) {
	if !reflect.DeepEqual(u, v) {
		t.Helper()
		t.Errorf("%#v != %#v\n", u, v)
	}
}
func expectEqualDate(t *testing.T, u, v time.Time) {
	if !u.Equal(v) {
		t.Helper()
		t.Errorf("%#v != %#v\n", u, v)
	}
}
func expectEqualMap(t *testing.T, u, v map[string]string) {
	if (len(u) > 0 || len(v) > 0) && !reflect.DeepEqual(u, v) {
		t.Helper()
		t.Errorf("%#v != %#v\n", u, v)
	}
}

func encodeString(m Email) string {
	var buffer bytes.Buffer
	m.Make(&buffer)
	return buffer.String()
}

func decodeMessage(message []byte) (Email, error) {
	var email Email
	msg, err := mail.ReadMessage(bytes.NewReader(message))
	if err != nil || msg == nil {
		return email, err
	}
	email.Date, err = msg.Header.Date()
	if err != nil {
		return email, err
	}
	email.To, err = addressList(msg.Header, "To")
	email.Cc, err = addressList(msg.Header, "Cc")
	email.Bcc, err = addressList(msg.Header, "Bcc")

	fromList, err := msg.Header.AddressList("From")
	if len(fromList) > 0 && fromList[0] != nil {
		email.From = *(fromList[0])
	}
	wordDecoder := mime.WordDecoder{}
	email.Subject, _ = wordDecoder.DecodeHeader(msg.Header.Get("Subject"))

	knownHeaders := map[string]struct{}{
		"Bcc":                       struct{}{},
		"Cc":                        struct{}{},
		"Content-Transfer-Encoding": struct{}{},
		"Content-Type":              struct{}{},
		"Date":                      struct{}{},
		"From":                      struct{}{},
		"Mime-Version":              struct{}{},
		"Subject":                   struct{}{},
		"To":                        struct{}{},
	}

	for k, _ := range msg.Header {
		_, ok := knownHeaders[k]
		if !ok {
			if email.Headers == nil {
				email.Headers = map[string]string{}
			}
			email.Headers[k] = msg.Header.Get(k)
		}
	}

	body, _ := io.ReadAll(msg.Body)
	if msg.Header.Get("Content-Transfer-Encoding") == "quoted-printable" {
		qpr := quotedprintable.NewReader(bytes.NewReader(body))
		body, _ = io.ReadAll(qpr)
	}
	email.Content = strings.ReplaceAll(strings.TrimSpace(string(body)), "\r\n", "\n")

	// 	Content     string
	// 	Attachments []Attachment
	return email, nil
}

func decodeMimeHeader(s string) (string, error) {
	d := mime.WordDecoder{}
	return d.DecodeHeader(s)
}

func qencodeString(s string) string {
	var b bytes.Buffer
	qencode(&b, s)
	return b.String()
}

func TestMimeHeader(t *testing.T) {
	for _, s := range []string{
		"HELLO WORLD",
		"test2 ←↑ →↓",
		"HELLO HELLO HELLO HELLO HELLO HELLO HELLO HELLO HELLO HELLO HELLO HELLO HELLO HELLO HELLO HELLO HELLO HELLO HELLO HELLO HELLO HELLO HELLO HELLO HELLO HELLO ←↑ →↓",
	} {
		q := qencodeString(s)
		v, _ := decodeMimeHeader(q)
		if s != v {
			t.Errorf("%q != %q", s, v)
		}
	}

}

// //
// func foo() {
// 	mediaType, _, err := mime.ParseMediaType(msg.Header.Get("Content-Type"))
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	if strings.HasPrefix(mediaType, "multipart/") {
// 	} else {
// 		switch strings.ToUpper(msg.Header.Get("Content-Transfer-Encoding")) {
// 		case "BASE64":
// 			encoded, _ := io.ReadAll(msg.Body)
// 			content, _ := base64.StdEncoding.DecodeString(string(encoded))
// 			t.Logf("\n%s\n", content)
// 		case "QUOTED-PRINTABLE":
// 			content, _ := io.ReadAll(quotedprintable.NewReader(msg.Body))
// 			t.Logf("\n%s\n", content)
// 		default:
// 			content, _ := io.ReadAll(msg.Body)
// 			t.Logf("\n%s\n", content)
// 		}
// 	}
//
// }
