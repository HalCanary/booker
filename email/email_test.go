package email

// Copyright 2022 Hal Canary
// Use of this program is governed by the file LICENSE.

import (
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
Subject: a quick note
From: z@example.com
To: a@example.com
To: b@example.com
Cc: c@example.com
Cc: d@example.com
MIME-Version: 1.0
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

func TestEbook(t *testing.T) {
	mail := Email{
		Date:    time.Date(2022, time.January, 1, 0, 0, 0, 0, time.UTC),
		To:      []string{"a@example.com", "b@example.com"},
		Cc:      []string{"c@example.com", "d@example.com"},
		Bcc:     []string{"e@example.com", "f@example.com"},
		From:    "z@example.com",
		Subject: "a quick note",
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
	result := string(mail.Make())
	if result != expectedS {
		t.Errorf("\n%q\n!=\n%q\n", result, expectedS)
	}
}
