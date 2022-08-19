package main

import (
	"archive/zip"
	"bytes"
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
nav ol li { list-style: none; }
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
	files := []string{
		"META-INF/container.xml",
		"toc.ncx",
		"content.opf",
		"frontmatter.xhtml",
		"toc.xhtml",
	}
	if info.CoverPath != "" {
		files = append(files, "cover.jpg")
	}
	for i, _ := range info.Chapters {
		files = append(files, fmt.Sprintf("%04d.xhtml", i))
	}
	for _, f := range files {
		err := zipFile(zw, f, dstDir)
		if err != nil {
			return "", err
		}
	}

	zw.Close()
	return zpath, nil
}

func zipFile(zw *zip.Writer, name, dir string) error {
	w, err := zw.Create(name)
	if err != nil {
		return err
	}
	f, err := os.Open(filepath.Join(dir, name))
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(w, f)
	return err
}

func zipRaw(zw *zip.Writer, name string, data []byte) error {
	w, err := zw.CreateRaw(&zip.FileHeader{
		Name:               name,
		CompressedSize64:   uint64(len(data)),
		UncompressedSize64: uint64(len(data)),
		CRC32:              crc32.ChecksumIEEE(data),
	})
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
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
	return writeDocument(filepath.Join(dstDir, "frontmatter.xhtml"), htmlNode)
}

func writeToc(info EbookInfo, dstDir string) error {
	links := Elem("ol")
	for i, chapter := range info.Chapters {
		chapterFile := fmt.Sprintf("%s/%04d.xhtml", dstDir, i)
		Append(links,
			Elem("li", link(filepathRel(dstDir, chapterFile), chapter.Title)))
		htmlNode := Element("html", map[string]string{"xmlns": "http://www.w3.org/1999/xhtml", "xml:lang": info.Language},
			head(chapter.Title, bookStyle, ""),
			makeChapter(chapter, i+1 == len(info.Chapters)),
		)
		if err := writeDocument(chapterFile, htmlNode); err != nil {
			return err
		}
	}
	htmlNode := Element("html",
		map[string]string{
			"xmlns":      "http://www.w3.org/1999/xhtml",
			"xml:lang":   info.Language,
			"xmlns:epub": "http://www.idpf.org/2007/ops",
		},
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
	uid := randomUUID()
	if err := makeNCX(info, uid, filepath.Join(dstDir, "toc.ncx")); err != nil {
		return err
	}
	if err := makePackage(info, uid, filepath.Join(dstDir, "content.opf")); err != nil {
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

func makePackage(info EbookInfo, uuid, dst string) error {
	cover := filepath.Base(info.CoverPath)
	manifestItems := []xmlItem{
		xmlItem{Id: "frontmatter", Href: "frontmatter.xhtml", MediaType: "application/xhtml+xml"},
		xmlItem{Id: "toc", Href: "toc.xhtml", MediaType: "application/xhtml+xml",
			Attributes: []xml.Attr{xml.Attr{Name: xml.Name{Local: "properties"}, Value: "nav"}}},
		xmlItem{Id: "ncx", Href: "toc.ncx", MediaType: "application/x-dtbncx+xml"},
	}
	itemrefs := []xmlItemref{
		xmlItemref{Idref: "frontmatter"},
		xmlItemref{Idref: "toc"},
	}
	if cover != "" {
		manifestItems = append(manifestItems, xmlItem{Id: cover, Href: cover, MediaType: "image/jpeg",
			Attributes: []xml.Attr{xml.Attr{Name: xml.Name{Local: "properties"}, Value: "cover-image"}}})
	}
	for i, _ := range info.Chapters {
		fn := fmt.Sprintf("%04d", i)
		id := "ch" + fn
		manifestItems = append(manifestItems, xmlItem{Id: id, Href: fn + ".xhtml", MediaType: "application/xhtml+xml"})
		itemrefs = append(itemrefs, xmlItemref{Idref: id})
	}
	modified := info.Modified.UTC().Format("2006-01-02T15:04:05Z")
	p := xmlPackage{
		Xmlns:            "http://www.idpf.org/2007/opf",
		XmlnsOpf:         "http://www.idpf.org/2007/opf",
		Version:          "3.0",
		UniqueIdentifier: "BookID",
		Metadata: xmlMetaData{
			XmlnsDC: "http://purl.org/dc/elements/1.1/",
			Properties: []xmlMetaProperty{
				xmlMetaProperty{Property: "dcterms:modified", Value: modified},
			},
			MetaItems: []xmlMetaItems{
				xmlMetaItems{
					XMLName:    xml.Name{Local: "dc:identifier"},
					Value:      uuid,
					Attributes: []xml.Attr{xml.Attr{Name: xml.Name{Local: "id"}, Value: "BookID"}},
				},
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
			xmlGuideReference{Title: "Cover page", Type: "cover", Href: "frontmatter.xhtml"},
			xmlGuideReference{Title: "Table of content", Type: "toc", Href: "toc.xhtml"},
		},
	}
	encoded, err := xml.MarshalIndent(&p, "", " ")
	if err != nil {
		return err
	}
	encoded = bytes.ReplaceAll(encoded, []byte("></item>"), []byte("/>"))
	encoded = bytes.ReplaceAll(encoded, []byte("></itemref>"), []byte("/>"))
	encoded = bytes.ReplaceAll(encoded, []byte("></reference>"), []byte("/>"))
	return os.WriteFile(dst, encoded, 0o644)
	// 	f, err := os.Create(dst)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	defer f.Close()
	// 	f.Write([]byte(xml.Header))
	// 	enc := xml.NewEncoder(f)
	// 	enc.Indent("", " ")
	// 	return enc.Encode(&p)
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
	XMLName    xml.Name
	Value      string     `xml:",chardata"`
	Attributes []xml.Attr `xml:",attr,any"`
}

type xmlMetaProperty struct {
	Property string `xml:"property,attr"`
	Value    string `xml:",chardata"`
}

type xmlItem struct {
	Id         string     `xml:"id,attr"`
	Href       string     `xml:"href,attr"`
	MediaType  string     `xml:"media-type,attr"`
	Attributes []xml.Attr `xml:",attr,any"`
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

//////
// <?xml version="1.0" encoding="UTF-8"?>
// <!DOCTYPE ncx PUBLIC "-//NISO//DTD ncx 2005-1//EN"
// "http://www.daisy.org/z3986/2005/ncx-2005-1.dtd">
// <ncx version="2005-1" xml:lang="en" xmlns="http://www.daisy.org/z3986/2005/ncx/">
//   <head>
//     <meta name="dtb:uid" content="123456789X"/> <!-- same as in .opf -->
//     <meta name="dtb:depth" content="1"/> <!-- 1 or higher -->
//     <meta name="dtb:totalPageCount" content="0"/> <!-- must be 0 -->
//     <meta name="dtb:maxPageNumber" content="0"/> <!-- must be 0 -->
//   </head>
//   <docTitle>
//     <text>Pride and Prejudice</text>
//   </docTitle>
//   <docAuthor>
//     <text>Austen, Jane</text>
//   </docAuthor>
//   <navMap>
//     <navPoint class="chapter" id="chapter1" playOrder="1">
//       <navLabel><text>Chapter 1</text></navLabel>
//       <content src="chapter1.xhtml"/>
//     </navPoint>
//   </navMap>
// </ncx>

func makeNCX(info EbookInfo, uid string, dst string) error {
	nav := []navPointXml{
		navPointXml{Class: "chapter", Id: "frontmatter", PlayOrder: 0, Label: "Front Matter", Content: contentXml{Src: "frontmatter.xhtml"}},
	}
	for i, ch := range info.Chapters {
		fn := fmt.Sprintf("%04d", i)
		id := "ch" + fn
		nav = append(nav, navPointXml{Class: "chapter", Id: id, PlayOrder: i + 1, Label: ch.Title, Content: contentXml{Src: fn + ".xhtml"}})
	}
	ncx := ncxXml{
		Xmlns:   "http://www.daisy.org/z3986/2005/ncx/",
		Version: "2005-1",
		Lang:    "en",
		Metas: []metaNcxXml{
			metaNcxXml{Name: "dtb:uid", Content: uid},
			metaNcxXml{Name: "dtb:depth", Content: "1"},
			metaNcxXml{Name: "dtb:totalPageCount", Content: "0"},
			metaNcxXml{Name: "dtb:maxPageNumber", Content: "0"},
		},
		Title:     info.Title,
		Author:    info.Authors,
		NavPoints: nav,
	}
	encoded, err := xml.MarshalIndent(&ncx, "", " ")
	if err != nil {
		return err
	}
	encoded = bytes.ReplaceAll(encoded, []byte("></meta>"), []byte("/>"))
	encoded = bytes.ReplaceAll(encoded, []byte("></content>"), []byte("/>"))
	return os.WriteFile(dst, encoded, 0o644)
}

type ncxXml struct {
	XMLName   xml.Name      `xml:"ncx"`
	Xmlns     string        `xml:"xmlns,attr"`
	Version   string        `xml:"version,attr"`
	Lang      string        `xml:"xml:lang,attr"`
	Metas     []metaNcxXml  `xml:"head>meta"`
	Title     string        `xml:"docTitle>text"`
	Author    string        `xml:"docAuthor>text"`
	NavPoints []navPointXml `xml:"navMap>navPoint"`
}
type metaNcxXml struct {
	Name    string `xml:"name,attr"`
	Content string `xml:"content,attr"`
}
type navPointXml struct {
	Class     string     `xml:"class,attr"`
	Id        string     `xml:"id,attr"`
	PlayOrder int        `xml:"playOrder,attr"`
	Label     string     `xml:"navLabel>text"`
	Content   contentXml `xml:"content"`
}
type contentXml struct {
	Src string `xml:"src,attr"`
}
