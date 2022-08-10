package main

import (
	"fmt"
	"io"
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
	Authors  string
	Cover    string
	Comments string
	Title    string
	Source   string
	Language string
	Chapters []Chapter
	Modified time.Time
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

// Write the ebook into given directory as HTML5 documents.}|
func (info *EbookInfo) Write(directory string, cacheDir string) (string, error) {
	if info.Title == "" {
		return "", nil
	}
	name := re.ReplaceAllString(NormalizeString(info.Title), "_")
	if !info.Modified.IsZero() {
		name = name + info.Modified.Format("_2006-01-02_150405")
	}
	dstDir := filepath.Join(directory, name)
	os.MkdirAll(dstDir, 0o755)
	tocPath := filepath.Join(dstDir, name+".html")
	informationComment := infocomment(*info)
	if info.Cover != "" {
		rc, _, err := GetUrl(info.Cover, cacheDir, "", false)
		if err == nil {
			src, _ := io.ReadAll(rc)
			rc.Close()
			fn := filepath.Join(dstDir, name+".jpg")
			if err = saveJpegWithScale(src, fn, 400, 600); err == nil {
				info.Cover = fn
			}
		}
	}
	cover, err := filepath.Rel(dstDir, info.Cover)
	if err != nil {
		cover = info.Cover
	}
	for i, chapter := range info.Chapters {
		var linkDiv *Node
		if i+1 == len(info.Chapters) {
			linkDiv = Elem("div",
				link(chapter.Url, chapter.Url),
				Elem("hr"),
			)
		}
		out, err := os.Create(chapterPath(dstDir, i))
		if err != nil {
			return "", err
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
		RenderDoc(out,
			Element("html", map[string]string{"lang": info.Language},
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
		out.Write([]byte{'\n'})
		out.Close()
	}
	p := make([]*Node, 0, len(info.Chapters)*2+1)
	p = append(p, TextNode("\n| "))
	for i, chapter := range info.Chapters {
		p = append(p,
			link(chapterPath(".", i), chapter.Title),
			TextNode("\n| "),
		)
	}

	out, err := os.Create(tocPath)
	if err != nil {
		return "", err
	}
	RenderDoc(out,
		Element("html", map[string]string{"lang": info.Language},
			head(info.Title, bookStyle, informationComment),
			Elem("body",
				Element("h2", map[string]string{"class": "chapter"}, TextNode(info.Title)),
				img(cover, "[COVER]"),
				Elem("p", p...),
			),
		),
	)
	out.Write([]byte{'\n'})
	out.Close()
	return tocPath, nil
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

func infocomment(info EbookInfo) string {
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
		{"cover", info.Cover},
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
