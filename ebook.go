package main

import (
	"archive/zip"
	"encoding/xml"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
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

// Return the time of most recently modified chapter.
func CalculateLastModified(chapters []Chapter) time.Time {
	var result time.Time
	for _, ch := range chapters {
		if !ch.Modified.IsZero() && ch.Modified.After(result) {
			result = ch.Modified
		}
	}
	return result
}

const bookStyle = `
div p{text-indent:2em;margin-top:0;margin-bottom:0}
div p:first-child{text-indent:0;}
table, th, td { border:2px solid #808080; padding:3px; }
table { border-collapse:collapse; margin:3px; }
nav ul li { list-style: none; }
`

var (
	re           = regexp.MustCompile("[^A-Za-z0-9.-]+")
	whitespaceRe = regexp.MustCompile("\\s+")
	stripRe      = regexp.MustCompile("(?:^\\s+)|(?:\\s+$)")
)

var BookAlreadyExists = errors.New("Book Already Exists")

func head(title, style, comment string) *Node {
	return Elem("head",
		//Element("meta", map[string]string{"charset": "utf-8"}),
		Element("meta", map[string]string{
			"http-equiv": "Content-Type", "content": "text/html; charset=utf-8"}),
		Comment(comment),
		Element("meta", map[string]string{
			"name": "viewport", "content": "width=device-width, initial-scale=1.0"}),
		Elem("title", TextNode(title)),
		Elem("style", TextNode(style)),
	)
}

func makeChapter(chapter Chapter, last bool) *Node {
	r := Elem("body")
	if chapter.Url != "" {
		Append(r, Comment(fmt.Sprintf("\n%s\n", chapter.Url)))
	}
	Append(r, Element("h2", map[string]string{"class": "chapter"}, TextNode(chapter.Title)))
	if !chapter.Modified.IsZero() {
		Append(r, Elem("p", Elem("em", TextNode(chapter.Modified.Format("2006-01-02")))))
	}
	Append(r, Elem("hr"), chapter.Content, Elem("hr"))
	if last {
		Append(r, Elem("div", link(chapter.Url, chapter.Url)), Elem("hr"))
	}
	return r
}

func writeDocument(path string, htmlNode *Node) error {
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()
	return RenderXHTMLDoc(out, htmlNode)
}

func getCover(url, dst string) error {
	rc, err := GetUrl(url, "", false)
	if err != nil {
		return err
	}
	src, _ := io.ReadAll(rc)
	rc.Close()
	jpeg, err := saveJpegWithScale(src, 400, 600)
	if err != nil {
		return err
	}
	if err := os.WriteFile(dst, jpeg, 0o644); err != nil {
		return err
	}
	return nil
}

func bookName(info EbookInfo) string {
	name := re.ReplaceAllString(NormalizeString(info.Title), "_")
	if info.Modified.IsZero() {
		return name
	}
	return name + info.Modified.Format("_2006-01-02_150405")
}

// Write the ebook into given directory as HTML5 documents.}|
func (info *EbookInfo) Write(directory string) (string, error) {
	if info.Title == "" {
		return "", errors.New("title missing")
	}
	name := bookName(*info)
	dstDir := filepath.Join(directory, name)
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		return "", err
	}
	if info.CoverURL != "" && info.CoverPath == "" {
		fn := filepath.Join(dstDir, "cover.jpg")
		err := getCover(info.CoverURL, fn)
		if err != nil {
			log.Printf("error: %v", err)
		} else {
			info.CoverPath, err = filepath.Abs(fn)
			if err != nil {
				log.Printf("error: %v", err)
				info.CoverPath = ""
			}
		}
	}
	// 	dst := filepath.Join(dstDir)
	// 	if exists(dst) {
	// 		return dst, BookAlreadyExists
	// 	}
	err := writeBook(*info, dstDir)
	if err != nil {
		os.RemoveAll(dstDir)
		return "", err
	}

	zpath := filepath.Join(dstDir, name+".epub")
	zfile, err := os.Create(zpath)
	if err != nil {
		os.RemoveAll(dstDir)
		return "", err

	}
	defer zfile.Close()
	zw := zip.NewWriter(zfile)

	zipRaw(zw, "mimetype", []byte("application/epub+zip"))
	zipFile(zw, "META-INF/container.xml", filepath.Join(dstDir, "META-INF/container.xml"))
	zipFile(zw, "content.opf", filepath.Join(dstDir, "content.opf"))
	zipFile(zw, "cover.xhtml", filepath.Join(dstDir, "cover.xhtml"))
	zipFile(zw, "toc.xhtml", filepath.Join(dstDir, "toc.xhtml"))

	if info.CoverPath != "" {
		zipFile(zw, "cover.jpg", filepath.Join(dstDir, "cover.jpg"))
	}
	for i, _ := range info.Chapters {
		p := fmt.Sprintf("%04d.xhtml", i)
		zipFile(zw, p, filepath.Join(dstDir, p))
	}
	zw.Close()
	return zpath, nil
}

func zipFile(zw *zip.Writer, name, path string) {
	w, _ := zw.Create(name)
	f, _ := os.Open(path)
	_, _ = io.Copy(w, f)
}

func zipRaw(zw *zip.Writer, name string, data []byte) {
	w, _ := zw.CreateRaw(&zip.FileHeader{
		Name:               name,
		CompressedSize64:   uint64(len(data)),
		UncompressedSize64: uint64(len(data)),
		CRC32:              crc32.ChecksumIEEE(data),
	})
	w.Write(data)
}

func filepathRel(basepath, targpath string) string {
	if targpath == "" {
		return ""
	}
	var err error
	basepath, err = filepath.Abs(basepath)
	if err != nil {
		log.Printf("error: %s\n", err)
		return ""
	}
	result, err := filepath.Rel(basepath, targpath)
	if err != nil {
		log.Printf("error: %s\n", err)
		return ""
	}
	return result
}

const conatainer_xml = xml.Header + `<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
<rootfiles>
<rootfile full-path="content.opf" media-type="application/oebps-package+xml"/>
</rootfiles>
</container>
`

func writeCover(info EbookInfo, dstDir string) error {
	description := Elem("div")
	for _, p := range strings.Split(info.Comments, "\n\n") {
		pnode := Elem("p")
		for i, c := range strings.Split(p, "\n\n") {
			if i > 0 {
				pnode.AppendChild(Elem("br"))
			}
			pnode.AppendChild(TextNode(c))
		}
		description.AppendChild(pnode)
	}

	htmlNode := Element("html", map[string]string{"xmlns": "http://www.w3.org/1999/xhtml", "xml:lang": info.Language},
		head(info.Title, bookStyle, ""),
		//head(info.Title, bookStyle, infoComment(info)),
		Elem("body",
			Elem("h1", TextNode(info.Title)),
			img(filepathRel(dstDir, info.CoverPath), "[COVER]"),
			Elem("div", TextNode(info.Authors)),
			Elem("div", TextNode(info.Source)),
			Elem("div", Elem("em", TextNode(info.Modified.Format("2006-01-02")))),
			description,
		),
	)
	return writeDocument(filepath.Join(dstDir, "cover.xhtml"), htmlNode)
}

func writeToc(info EbookInfo, dstDir string) error {
	links := Elem("ul")
	for i, chapter := range info.Chapters {
		chapterFile := fmt.Sprintf("%s/%04d.xhtml", dstDir, i)
		Append(links,
			Element("li", map[string]string{"style": "list-style: none"}, link(filepathRel(dstDir, chapterFile), chapter.Title)))
		htmlNode := Element("html", map[string]string{"xmlns": "http://www.w3.org/1999/xhtml", "xml:lang": info.Language},
			head(chapter.Title, bookStyle, ""),
			makeChapter(chapter, i+1 == len(info.Chapters)),
		)
		if err := writeDocument(chapterFile, htmlNode); err != nil {
			return err
		}
	}
	htmlNode := Element("html", map[string]string{"xmlns": "http://www.w3.org/1999/xhtml", "xml:lang": info.Language},
		//head(info.Title, bookStyle, infoComment(info)),
		head(info.Title, bookStyle, ""),
		Elem("body",
			Element("h2", map[string]string{"class": "chapter"}, TextNode(info.Title)),
			img(filepathRel(dstDir, info.CoverPath), "[COVER]"),
			Element("nav", map[string]string{"epub:type": "toc"}, Elem("h2", TextNode("Contents")), links),
		),
	)
	return writeDocument(filepath.Join(dstDir, "toc.xhtml"), htmlNode)
}

func writeBook(info EbookInfo, dstDir string) error {
	if err := os.MkdirAll(filepath.Join(dstDir, "META-INF"), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dstDir, "META-INF/container.xml"), []byte(conatainer_xml), 0o644); err != nil {
		return err
	}
	if err := makePackage(info, filepath.Join(dstDir, "content.opf")); err != nil {
		return err
	}
	if err := writeCover(info, dstDir); err != nil {
		return err
	}
	return writeToc(info, dstDir)
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

func wsb(builder *strings.Builder, strs ...string) {
	for _, s := range strs {
		builder.WriteString(s)
	}
}

func infoComment(info EbookInfo) string {
	var builder strings.Builder
	builder.WriteString("\n\n")
	modified := ""
	if !info.Modified.IsZero() {
		modified = info.Modified.Format(time.RFC3339)
	}
	for _, v := range [][2]string{
		{"src", info.Source},
		{"title", info.Title},
		{"authors", info.Authors},
		{"chapter_count", strconv.Itoa(len(info.Chapters))},
		{"cover", info.CoverURL},
		{"last_modified", modified},
		{"comments", info.Comments},
	} {
		if v[1] != "" {
			wsb(&builder, v[0], ": ", v[1], "\n")
		}
	}
	builder.WriteString("\n")
	return builder.String()
}

func randomUUID() string {
	var v [16]byte
	rand.Read(v[:])
	return fmt.Sprintf("%x-%x-%x-%x-%x", v[0:4], v[4:6], v[6:8], v[8:10], v[10:16])
}

func makePackage(info EbookInfo, dst string) error {
	cover := filepath.Base(info.CoverPath)
	manifestItems := []xmlItem{
		xmlItem{Id: "cover.xhtml", Href: "cover.xhtml", MediaType: "application/xhtml+xml"},
		xmlItem{Id: "toc.xhtml", Href: "toc.xhtml", MediaType: "application/xhtml+xml", Properties: "nav"},
	}
	itemrefs := []xmlItemref{
		xmlItemref{Idref: "cover.xhtml"},
		xmlItemref{Idref: "toc.xhtml"},
	}
	if cover != "" {
		manifestItems = append(manifestItems, xmlItem{Id: cover, Href: cover, MediaType: "image/jpeg", Properties: "cover-image"})
	}
	for i, _ := range info.Chapters {
		fn := fmt.Sprintf("%04d.xhtml", i)
		manifestItems = append(manifestItems, xmlItem{Id: fn, Href: fn, MediaType: "application/xhtml+xml"})
		itemrefs = append(itemrefs, xmlItemref{Idref: fn})
	}
	p := xmlPackage{
		Xmlns:            "http://www.idpf.org/2007/opf",
		XmlnsOpf:         "http://www.idpf.org/2007/opf",
		Version:          "3.0",
		UniqueIdentifier: "BookID",
		Metadata: xmlMetaData{
			XmlnsDC: "http://purl.org/dc/elements/1.1/",
			Properties: []xmlMetaProperty{
				xmlMetaProperty{Property: "dcterms:modified", Value: info.Modified.Format(time.RFC3339)},
			},
			MetaItems: []xmlMetaItems{
				xmlMetaItems{XMLName: xml.Name{Local: "dc:identifier"}, Value: randomUUID()},
				xmlMetaItems{XMLName: xml.Name{Local: "dc:title"}, Value: info.Title},
				xmlMetaItems{XMLName: xml.Name{Local: "dc:language"}, Value: info.Language},
				xmlMetaItems{XMLName: xml.Name{Local: "dc:creator"}, Value: info.Authors},
				xmlMetaItems{XMLName: xml.Name{Local: "dc:description"}, Value: info.Comments},
			},
		},
		ManifestItems: manifestItems,
		Spine: xmlSpine{
			Toc:      "ncx",
			Itemrefs: itemrefs,
		},
		GuideRefs: []xmlGuideReference{
			xmlGuideReference{Title: "Cover page", Type: "cover", Href: "cover.xhtml"},
			xmlGuideReference{Title: "Table of content", Type: "toc", Href: "toc.xhtml"},
		},
	}
	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer f.Close()
	f.Write([]byte(xml.Header))
	enc := xml.NewEncoder(f)
	return enc.Encode(&p)
}

type xmlPackage struct {
	XMLName          xml.Name            `xml:"package"`
	Xmlns            string              `xml:"xmlns,attr"`
	XmlnsOpf         string              `xml:"xmlns:opf,attr"`
	Version          string              `xml:"version,attr"`
	UniqueIdentifier string              `xml:"unique-identifier,attr"`
	Metadata         xmlMetaData         `xml:"metadata"`
	ManifestItems    []xmlItem           `xml:"manifest>item"`
	Spine            xmlSpine            `xml:"spine"`
	GuideRefs        []xmlGuideReference `xml:"guide>reference"`
}

type xmlMetaData struct {
	XmlnsDC    string            `xml:"xmlns:dc,attr"`
	Properties []xmlMetaProperty `xml:"meta"`
	MetaItems  []xmlMetaItems    `xml:",any"`
}

type xmlMetaItems struct {
	XMLName xml.Name
	Value   string `xml:",chardata"`
}

type xmlMetaProperty struct {
	Property string `xml:"property,attr"`
	Value    string `xml:",chardata"`
}

type xmlItem struct {
	Id         string `xml:"id,attr"`
	Href       string `xml:"href,attr"`
	MediaType  string `xml:"media-type,attr"`
	Properties string `xml:"properties,omitempy,attr"`
}

type xmlSpine struct {
	Toc      string       `xml:"toc,attr"`
	Itemrefs []xmlItemref `xml:"itemref"`
}

type xmlItemref struct {
	Idref string `xml:"idref,attr"`
}

type xmlGuideReference struct {
	Title string `xml:"title,attr"`
	Type  string `xml:"type,attr"`
	Href  string `xml:"href,attr"`
}
