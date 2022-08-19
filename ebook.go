package main

import (
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"math/rand"
	"regexp"
	"strings"
	"time"
)

type Chapter struct {
	Title    string
	Url      string
	Content  *Node
	Modified time.Time
}

// Ebook content and metadata.
type EbookInfo struct {
	Authors   string
	CoverURL  string
	CoverPath string
	Comments  string
	Title     string
	Source    string
	Language  string
	Chapters  []Chapter
	Modified  time.Time
}

var (
	re           = regexp.MustCompile("[^A-Za-z0-9.-]+")
	whitespaceRe = regexp.MustCompile("\\s+")
	stripRe      = regexp.MustCompile("(?:^\\s+)|(?:\\s+$)")
)

const bookStyle = `
div p{text-indent:2em;margin-top:0;margin-bottom:0}
div p:first-child{text-indent:0;}
table, th, td { border:2px solid #808080; padding:3px; }
table { border-collapse:collapse; margin:3px; }
ol.flat {list-style-type:none;}
ol.flat li {list-style:none; display:inline;}
ol.flat li::after {content:"]";}
ol.flat li::before {content:"[";}
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
	return Elem("head",
		Element("meta", map[string]string{
			"http-equiv": "Content-Type", "content": "text/html; charset=utf-8"}),
		Comment(comment),
		Element("meta", map[string]string{
			"name": "viewport", "content": "width=device-width, initial-scale=1.0"}),
		Elem("title", TextNode(title)),
		Elem("style", TextNode(style)),
	)
}

func (info EbookInfo) Name() string {
	name := re.ReplaceAllString(NormalizeString(info.Title), "_")
	if info.Modified.IsZero() {
		return name
	}
	return name + info.Modified.UTC().Format("_2006-01-02_150405")
}

// Write the ebook as an Epub.
func (info EbookInfo) Write(dst io.Writer) error {
	var (
		uid      string = randomUUID()
		jpegData []byte
		cover    string
	)
	if info.CoverURL != "" {
		rc, err := GetUrl(info.CoverURL, "", false)
		if err != nil {
			log.Printf("error: %v", err)
		} else {
			src, _ := io.ReadAll(rc)
			rc.Close()
			jpegData, err = saveJpegWithScale(src, 400, 600)
			if err != nil {
				log.Printf("error: %v", err)
			} else {
				cover = "cover.jpg"
			}
		}
	}

	zw := MakeZipper(dst)
	defer zw.Close()

	if w := zw.CreateStore("mimetype", info.Modified); w != nil {
		_, zw.Error = w.Write([]byte("application/epub+zip"))
	}
	if w := zw.CreateDeflate("META-INF/container.xml", info.Modified); w != nil {
		_, zw.Error = w.Write([]byte(conatainer_xml))
	}
	if w := zw.CreateDeflate("book/toc.ncx", info.Modified); w != nil {
		zw.Error = makeNCX(info, uid, w)
	}
	if w := zw.CreateDeflate("book/content.opf", info.Modified); w != nil {
		zw.Error = makePackage(info, uid, w, cover)
	}
	if w := zw.CreateDeflate("book/frontmatter.xhtml", info.Modified); w != nil {
		zw.Error = writeFrontmatter(info, w, cover)
	}
	if w := zw.CreateDeflate("book/toc.xhtml", info.Modified); w != nil {
		zw.Error = writeToc(info, w)
	}
	if cover != "" {
		if w := zw.CreateStore("book/"+cover, info.Modified); w != nil {
			_, zw.Error = w.Write(jpegData)
		}
	}
	for i, chapter := range info.Chapters {
		if w := zw.CreateDeflate(fmt.Sprintf("book/%04d.xhtml", i), chapter.Modified); w != nil {
			var churl string
			if i+1 == len(info.Chapters) {
				churl = chapter.Url
			}
			zw.Error = writeChapter(chapter, churl, info.Language, w)
		}
	}
	return zw.Error
}

func writeFrontmatter(info EbookInfo, dst io.Writer, cover string) error {
	description := Elem("div")
	for _, p := range strings.Split(info.Comments, "\n\n") {
		pnode := Elem("p")
		for i, c := range strings.Split(p, "\n\n") {
			if i > 0 {
				pnode.Append(Elem("br"))
			}
			pnode.Append(TextNode(c))
		}
		description.Append(pnode)
	}
	htmlNode := Element("html", map[string]string{"xmlns": "http://www.w3.org/1999/xhtml", "xml:lang": info.Language},
		head(info.Title, bookStyle, ""),
		Elem("body",
			Elem("h1", TextNode(info.Title)),
			img(cover, "[COVER]"),
			Elem("div", TextNode(info.Authors)),
			Elem("div", TextNode(info.Source)),
			Elem("div", Elem("em", TextNode(info.Modified.Format("2006-01-02")))),
			description,
		),
	)
	return htmlNode.RenderXHTMLDoc(dst)
}

func writeChapter(chapter Chapter, url, lang string, dst io.Writer) error {
	body := Elem("body")
	if chapter.Url != "" {
		body.Append(Comment(fmt.Sprintf("\n%s\n", chapter.Url)))
	}
	body.Append(Element("h2", map[string]string{"class": "chapter"}, TextNode(chapter.Title)))
	if !chapter.Modified.IsZero() {
		body.Append(Elem("p", Elem("em", TextNode(chapter.Modified.Format("2006-01-02")))))
	}
	body.Append(Elem("hr"), chapter.Content, Elem("hr"))
	if url != "" {
		body.Append(Elem("div", link(url, url)), Elem("hr"))
	}
	htmlNode := Element("html",
		map[string]string{"xmlns": "http://www.w3.org/1999/xhtml", "xml:lang": lang},
		head(chapter.Title, bookStyle, ""),
		body,
	)
	return htmlNode.RenderXHTMLDoc(dst)
}

func writeToc(info EbookInfo, dst io.Writer) error {
	links := Element("ol", map[string]string{"class": "flat"})
	for i, ch := range info.Chapters {
		label := fmt.Sprintf("%d. %s", i+1, ch.Title)
		links.Append(Elem("li", link(fmt.Sprintf("%04d.xhtml", i), label)))
	}
	htmlNode := Element("html",
		map[string]string{
			"xmlns":      "http://www.w3.org/1999/xhtml",
			"xml:lang":   info.Language,
			"xmlns:epub": "http://www.idpf.org/2007/ops",
		},
		head(info.Title, bookStyle, ""),
		Elem("body",
			Element("nav", map[string]string{"epub:type": "toc"}, Elem("h2", TextNode("Contents")), links),
		),
	)
	return htmlNode.RenderXHTMLDoc(dst)
}

func link(url, text string) *Node {
	if url == "" {
		return nil
	}
	return Element("a", map[string]string{"href": url}, TextNode(text))
}

func img(url, alt string) *Node {
	if url == "" {
		return nil
	}
	return Element("img", map[string]string{"src": url, "alt": alt})
}

func randomUUID() string {
	var v [16]byte
	rand.Read(v[:])
	return fmt.Sprintf("%x-%x-%x-%x-%x", v[0:4], v[4:6], v[6:8], v[8:10], v[10:16])
}
