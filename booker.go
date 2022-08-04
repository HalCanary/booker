// Copyright 2022 Hal Canary
// Use of this program is governed by the file LICENSE.
package main

import (
	"io"
	"log"
	"os"
)

func head(title, style string) *Node {
	return Elem("head",
		Comment(" Hello World "),
		Element("meta", map[string]string{"charset": "utf-8"}),
		Element("meta", map[string]string{
			"name": "viewport", "content": "width=device-width, initial-scale=1.0"}),
		Elem("title", TextNode(title)),
		Elem("style", TextNode(style)),
	)
}

func main() {
	lang := "en"
	title := "HELLO WORLD"
	style := "body{max-width:35em;margin:22px auto 64px auto;padding:0 8px;}"
	err := writeHtml(os.Stdout, lang, title, style, Elem("h1", TextNode("HELLO!")),
		Elem("p", TextNode("world")))
	if err != nil {
		log.Fatal(err)
	}
}

func writeHtml(out io.Writer, lang, title, style string, contents ...*Node) error {
	return RenderDoc(out,
		Element("html", map[string]string{"lang": lang}, head(title, style), Elem("body", contents...)))

}
