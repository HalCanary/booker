package main

import (
	"errors"
	"fmt"
	"io"
	"log"
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
`

var (
	re           = regexp.MustCompile("[^A-Za-z0-9.-]+")
	whitespaceRe = regexp.MustCompile("\\s+")
	stripRe      = regexp.MustCompile("(?:^\\s+)|(?:\\s+$)")
)

func head(title, style, comment string) *Node {
	return Elem("head",
		Element("meta", map[string]string{"charset": "utf-8"}),
		Comment(comment),
		Element("meta", map[string]string{
			"name": "viewport", "content": "width=device-width, initial-scale=1.0"}),
		Elem("title", TextNode(title)),
		Elem("style", TextNode(style)),
	)
}

var BookAlreadyExists = errors.New("Book Already Exists")

func writeChapter(path string, chapter Chapter, language string, last bool) error {
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()

	var linkDiv *Node
	if last {
		linkDiv = Elem("div",
			link(chapter.Url, chapter.Url),
			Elem("hr"),
		)
	}
	comment := ""
	if chapter.Url != "" {
		comment = fmt.Sprintf("\n%s\n", chapter.Url)
	}

	var modified *Node
	if !chapter.Modified.IsZero() {
		modified = Elem("div",
			Elem("p", Elem("em", TextNode(chapter.Modified.Format("2006-01-02")))),
			Elem("hr"),
		)
	}
	return RenderDoc(out,
		Element("html", map[string]string{"lang": language},
			head(chapter.Title, bookStyle, comment),
			Elem("body",
				Element("h2", map[string]string{"class": "chapter"}, TextNode(chapter.Title)),
				modified,
				chapter.Content,
				Elem("hr"),
				linkDiv,
			),
		),
	)
}

func getCover(url, dst, cacheDir string) error {
	rc, _, err := GetUrl(url, cacheDir, "", false)
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
func (info *EbookInfo) Write(directory string, cacheDir string) (string, error) {
	if info.Title == "" {
		return "", errors.New("title missing")
	}
	name := bookName(*info)
	dstDir := filepath.Join(directory, name)
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		return "", err
	}
	if info.CoverURL != "" && info.CoverPath == "" {
		fn := filepath.Join(dstDir, name+".jpg")
		err := getCover(info.CoverURL, fn, cacheDir)
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
	dst := filepath.Join(dstDir, name+".html")
	if exists(dst) {
		return dst, BookAlreadyExists
	}
	err := writeBook(*info, dst, cacheDir)
	if err != nil {
		os.RemoveAll(dstDir)
		return "", err
	}
	return dst, nil
}

func makeChapterLinks(chapters []Chapter) *Node {
	p := Elem("p")
	p.AppendChild(TextNode("\n| "))
	for i, chapter := range chapters {
		p.AppendChild(link(chapterPath(".", i), chapter.Title))
		p.AppendChild(TextNode("\n| "))
	}
	return p
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

func writeBook(info EbookInfo, tocPath, cacheDir string) error {
	dstDir := filepath.Dir(tocPath)
	for i, chapter := range info.Chapters {
		path := chapterPath(dstDir, i)
		last := i+1 == len(info.Chapters)
		if err := writeChapter(path, chapter, info.Language, last); err != nil {
			return err
		}
	}
	out, err := os.Create(tocPath)
	if err != nil {
		return err
	}
	defer out.Close()
	return RenderDoc(out,
		Element("html", map[string]string{"lang": info.Language},
			head(info.Title, bookStyle, infoComment(info)),
			Elem("body",
				Element("h2", map[string]string{"class": "chapter"}, TextNode(info.Title)),
				img(filepathRel(dstDir, info.CoverPath), "[COVER]"),
				makeChapterLinks(info.Chapters),
			),
		),
	)
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

func chapterPath(dir string, i int) string { return fmt.Sprintf("%s/%04d.html", dir, i) }

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
