// Copyright 2022 Hal Canary
// Use of this program is governed by the file LICENSE.
package ebook

import (
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"math/rand"
	"strings"
	"time"

	"github.com/HalCanary/booker/dom"
	"github.com/HalCanary/booker/zipper"
)

type Chapter struct {
	Title    string
	Url      string
	Content  *Node
	Modified time.Time
}

// Ebook content and metadata.
type EbookInfo struct {
	Authors  string
	Comments string
	Title    string
	Source   string
	Language string
	Chapters []Chapter
	Modified time.Time
	Cover    []byte
}

const bookStyle = `
div p{text-indent:2em;margin-top:0;margin-bottom:0}
div p:first-child{text-indent:0;}
table, th, td { border:2px solid #808080; padding:3px; }
table { border-collapse:collapse; margin:3px; }
ol.flat {list-style-type:none;}
ol.flat li {list-style:none; display:inline;}
ol.flat li::after {content:"]";}
ol.flat li::before {content:"[";}
div.mid {margin: 0 auto;}
div.mid p {text-indent:0;}
`

const conatainer_xml = xml.Header + `<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
<rootfiles>
<rootfile full-path="book/content.opf" media-type="application/oebps-package+xml"/>
</rootfiles>
</container>
`

// Return the time of most recently modified chapter.
func (info EbookInfo) CalculateLastModified() time.Time {
	var result time.Time = info.Modified
	for _, ch := range info.Chapters {
		if !ch.Modified.IsZero() && ch.Modified.After(result) {
			result = ch.Modified
		}
	}
	return result
}

func head(title, style, comment string) *Node {
	return dom.Elem("head",
		dom.Element("meta", dom.Attr{
			"http-equiv": "Content-Type", "content": "text/html; charset=utf-8"}),
		dom.Comment(comment),
		dom.Element("meta", dom.Attr{
			"name": "viewport", "content": "width=device-width, initial-scale=1.0"}),
		dom.Elem("title", dom.TextNode(title)),
		dom.Elem("style", dom.TextNode(style)),
	)
}

// Write the ebook as an Epub.
func (info EbookInfo) Write(dst io.Writer) error {
	var (
		uid   string = randomUUID()
		cover []byte
	)
	if len(info.Cover) > 0 {
		var err error
		cover, err = saveJpegWithScale(info.Cover, 400, 600)
		if err != nil {
			log.Printf("Cover error: %v", err)
			cover = nil
		}
	}
	for i, chapter := range info.Chapters {
		info.Chapters[i].Content = Cleanup(chapter.Content)
	}

	zw := zipper.Make(dst)
	defer zw.Close()

	if w := zw.CreateStore("mimetype", time.Time{}); w != nil {
		_, zw.Error = w.Write([]byte("application/epub+zip"))
	}
	if w := zw.CreateDeflate("META-INF/container.xml", info.Modified); w != nil {
		_, zw.Error = w.Write([]byte(conatainer_xml))
	}
	if w := zw.CreateDeflate("book/"+"toc.ncx", info.Modified); w != nil {
		zw.Error = makeNCX(info, uid, w)
	}
	if w := zw.CreateDeflate("book/"+"content.opf", info.Modified); w != nil {
		zw.Error = makePackage(info, uid, w, len(cover) > 0)
	}
	if w := zw.CreateDeflate("book/"+"frontmatter.xhtml", info.Modified); w != nil {
		zw.Error = writeFrontmatter(info, w, len(cover) > 0)
	}
	if w := zw.CreateDeflate("book/"+"toc.xhtml", info.Modified); w != nil {
		zw.Error = writeToc(info, w)
	}
	if len(cover) > 0 {
		if w := zw.CreateStore("book/"+"cover.jpg", info.Modified); w != nil {
			_, zw.Error = w.Write(cover)
		}
	}
	for i, chapter := range info.Chapters {
		if w := zw.CreateDeflate(fmt.Sprintf("book/"+"%04d.xhtml", i), chapter.Modified); w != nil {
			var churl string
			if i+1 == len(info.Chapters) {
				churl = chapter.Url
			}
			zw.Error = writeChapter(chapter, churl, info.Language, w)
		}
	}
	return zw.Error
}

func writeFrontmatter(info EbookInfo, dst io.Writer, cover bool) error {
	description := dom.Elem("div")
	for _, p := range strings.Split(info.Comments, "\n\n") {
		pnode := dom.Elem("p")
		for i, c := range strings.Split(p, "\n\n") {
			if i > 0 {
				pnode.Append(dom.Elem("br"))
			}
			pnode.Append(dom.TextNode(c))
		}
		description.Append(pnode)
	}
	var img *dom.Node
	if cover {
		img = dom.Element("img", dom.Attr{"src": "cover.jpg", "alt": "[COVER]"})
	}
	htmlNode := dom.Element("html", dom.Attr{"xmlns": "http://www.w3.org/1999/xhtml", "xml:lang": info.Language},
		head(info.Title, bookStyle, ""),
		dom.Elem("body",
			dom.Elem("h1", dom.TextNode(info.Title)),
			img,
			dom.Elem("div", dom.TextNode(info.Authors)),
			dom.Elem("div", dom.TextNode(info.Source)),
			dom.Elem("div", dom.Elem("em", dom.TextNode(info.Modified.Format("2006-01-02")))),
			description,
		),
	)
	return htmlNode.RenderXHTMLDoc(dst)
}

func writeChapter(chapter Chapter, url, lang string, dst io.Writer) error {
	body := dom.Elem("body")
	if chapter.Url != "" {
		body.Append(dom.Comment(fmt.Sprintf("\n%s\n", chapter.Url)))
	}
	body.Append(dom.Element("h2", dom.Attr{"class": "chapter"}, dom.TextNode(chapter.Title)))
	if !chapter.Modified.IsZero() {
		body.Append(dom.Elem("p", dom.Elem("em", dom.TextNode(chapter.Modified.Format("2006-01-02")))))
	}
	body.Append(dom.Elem("hr"), chapter.Content, dom.Elem("hr"))
	if url != "" {
		body.Append(dom.Elem("div", link(url, url)), dom.Elem("hr"))
	}
	htmlNode := dom.Element("html",
		dom.Attr{"xmlns": "http://www.w3.org/1999/xhtml", "xml:lang": lang},
		head(chapter.Title, bookStyle, ""),
		body,
	)
	return htmlNode.RenderXHTMLDoc(dst)
}

func writeToc(info EbookInfo, dst io.Writer) error {
	links := dom.Element("ol", dom.Attr{"class": "flat"})
	for i, ch := range info.Chapters {
		label := fmt.Sprintf("%d. %s", i+1, ch.Title)
		links.Append(dom.Elem("li", link(fmt.Sprintf("%04d.xhtml", i), label)))
	}
	htmlNode := dom.Element("html",
		dom.Attr{
			"xmlns":      "http://www.w3.org/1999/xhtml",
			"xml:lang":   info.Language,
			"xmlns:epub": "http://www.idpf.org/2007/ops",
		},
		head(info.Title, bookStyle, ""),
		dom.Elem("body",
			dom.Element("nav", dom.Attr{"epub:type": "toc"}, dom.Elem("h2", dom.TextNode("Contents")), links),
		),
	)
	return htmlNode.RenderXHTMLDoc(dst)
}

func link(url, text string) *Node {
	if url == "" {
		return nil
	}
	return dom.Element("a", dom.Attr{"href": url}, dom.TextNode(text))
}

func imgElem(url, alt string) *Node {
	if url == "" {
		return nil
	}
	return dom.Element("img", dom.Attr{"src": url, "alt": alt})
}

func randomUUID() string {
	var v [16]byte
	rand.Read(v[:])
	return fmt.Sprintf("%x-%x-%x-%x-%x", v[0:4], v[4:6], v[6:8], v[8:10], v[10:16])
}
