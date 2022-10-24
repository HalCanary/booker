package dom

// Copyright 2022 Hal Canary
// Use of this program is governed by the file LICENSE.

import (
	"bytes"
	"testing"
)

func maketest() *Node {
	return Element("html", Attr{"lang": "en"},
		Elem("head",
			Element("meta", Attr{
				"http-equiv": "Content-Type", "content": "text/html; charset=utf-8"}),
			Element("meta", Attr{
				"name": "viewport", "content": "width=device-width, initial-scale=1.0"}),
			Elem("title", TextNode("TITLE")),
		),
		Elem("body", Elem("p", TextNode("hi"))),
	)
}

const expected = `<!DOCTYPE html>
<html lang="en"><head><meta content="text/html; charset=utf-8" http-equiv="Content-Type"/><meta content="width=device-width, initial-scale=1.0" name="viewport"/><title>TITLE</title></head><body><p>hi</p></body></html>
`

const expected2 = `<!DOCTYPE html>
<html lang="en"><head><meta content="text/html; charset=utf-8" http-equiv="Content-Type"><meta content="width=device-width, initial-scale=1.0" name="viewport"><title>TITLE</title></head><body><p>hi</p></body></html>
`

func TestDom(t *testing.T) {
	var b bytes.Buffer
	x := maketest()
	x.RenderHTML(&b)
	if expected != b.String() {
		t.Error(b.String())
	}
	b = bytes.Buffer{}
	x = maketest()
	x.RenderHTMLExperimental(&b)
	if expected2 != b.String() {
		t.Errorf(b.String())
	}
}
