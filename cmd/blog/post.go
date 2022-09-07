package main

import (
	"bufio"
	"bytes"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/HalCanary/booker/dom"
	"gitlab.com/golang-commonmark/markdown"
)

type Post struct {
	Source     string
	Title      string
	Author     string
	Time       time.Time
	TZone      string
	Id         string
	Summary    string
	Categories []string
	Markdown   []byte
	Next       *Post
	Prev       *Post
}

func (p Post) LongId() string {
	return p.Time.Format("2006/01/02/") + p.Id
}

func (p Post) Location() string {
	return p.LongId() + "/index.html"
}

func (p Post) Link() *dom.Node {
	return link(concat("/vv/", p.LongId(), "/"), p.Title)
}

func (p *Post) longLink(description string) *dom.Node {
	if p == nil {
		return dom.RawHtml("&nbsp;") // return dom.TextNode("\u00a0")
	}
	return dom.Elem("p",
		dom.TextNode(concat("(", description, ": ")),
		p.Link(),
		dom.TextNode(")"),
	)
}

func (p Post) NextLink() *dom.Node { return p.Next.longLink("newer") }

func (p Post) PrevLink() *dom.Node { return p.Prev.longLink("older") }

func (post Post) Article(level int) *dom.Node {
	categoryPrefix := "/vv/category/"
	url := concat("https://halcanary.org/vv/", post.LongId(), "/")
	var cats *dom.Node
	if len(post.Categories) > 0 {
		cats = dom.Elem("div")
		for i, c := range post.Categories {
			if i != 0 {
				cats.Append(dom.TextNode("; "))
			}
			cats.Append(
				dom.Element("a", dom.Attr{
					"href":  concat(categoryPrefix, c, "/"),
					"class": "p-category",
				}, dom.TextNode("#"+c)),
			)
		}
	}

	tokens := Markdowner().Parse(post.Markdown)
	lowestLevel := math.MaxInt
	for _, token := range tokens {
		if h, ok := token.(*markdown.HeadingOpen); ok && h != nil {
			if h.HLevel < lowestLevel {
				lowestLevel = h.HLevel
			}
		}
	}
	if lowestLevel < math.MaxInt {
		levelChange := level + 1 - lowestLevel
		for _, token := range tokens {
			if h, ok := token.(*markdown.HeadingOpen); ok && h != nil {
				h.HLevel += levelChange
			} else if h, ok := token.(*markdown.HeadingClose); ok && h != nil {
				h.HLevel += levelChange
			}
		}
	}
	postHtml := Markdowner().RenderTokensToString(tokens)

	var summary *dom.Node
	var summary2 *dom.Node
	if post.Summary != "" {
		summary = dom.Element("p", dom.Attr{"class": "p-summary"}, dom.TextNode(post.Summary))
		summary2 = dom.Element("div", dom.Attr{"style": "display:none;"}, dom.TextNode(post.Summary))
	}
	return dom.Element("article", dom.Attr{"id": post.LongId(), "class": "h-entry"},
		dom.TextNode("\n"),
		dom.Elem("header",
			dom.TextNode("\n"),
			dom.Comment(concat(" SRC= ", post.Source, " ")),
			dom.TextNode("\n"),
			header(level, dom.Attr{"class": "blogtitle p-name"}, dom.TextNode(post.Title)),
			dom.TextNode("\n"),
			summary,
			dom.TextNode("\n"),
			dom.Element("div", dom.Attr{"class": "byline plainlink"},
				dom.TextNode("\n"),
				dom.Elem("div",
					dom.TextNode("\n"),
					dom.Element("div", dom.Attr{"class": "p-author"}, dom.TextNode(post.Author)),
					dom.TextNode("\n"),
					dom.Elem("div",
						dom.Element("time",
							dom.Attr{"datetime": post.Time.Format("2006-01-02T15:04:05-07:00"), "class": "dt-published"},
							dom.TextNode(post.Time.Format("2006-01-02 15:04:05-07:00 (MST)")),
						),
					),
					dom.TextNode("\n"),
					dom.Elem("div",
						dom.Element("a", dom.Attr{"href": concat("/vv/", post.LongId(), "/"), "class": "u-url u-uid"}, dom.TextNode(url)),
					),
					dom.TextNode("\n"),
					cats,
					dom.TextNode("\n"),
				),
				dom.TextNode("\n"),
			),
			dom.TextNode("\n"),
		),
		dom.TextNode("\n"),
		dom.Element("div", dom.Attr{"class": "content e-content"},
			dom.TextNode("\n"),
			summary2,
			dom.TextNode("\n"),
			dom.RawHtml(postHtml),
		),
		dom.TextNode("\n"),
	)
}

type PostList []Post

func (a PostList) Len() int           { return len(a) }
func (a PostList) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a PostList) Less(i, j int) bool { return a[i].Time.Before(a[j].Time) }

func ReadPost(path string) (post Post, err error) {
	var f *os.File
	f, err = os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		splt := strings.SplitN(scanner.Text(), "=", 2)
		if len(splt) != 2 {
			break
		}
		value := strings.TrimSpace(splt[1])
		switch strings.TrimSpace(splt[0]) {
		case "TITLE":
			post.Title = value
		case "DATE":
			post.Time, err = time.Parse("2006-01-02 15:04:05Z07:00 (MST)", value)
			if err != nil {
				return
			}
		case "POSTID":
			post.Id = value
		case "SUMMARY":
			post.Summary = value
		case "CATEGORIES":
			cats := strings.Split(value, ";")
			for _, c := range cats {
				if c != "" {
					post.Categories = append(post.Categories, c)
				}
			}
		}
	}
	var buffer bytes.Buffer
	for scanner.Scan() {
		buffer.Write(scanner.Bytes())
		buffer.Write([]byte{'\n'})
	}
	post.Markdown = buffer.Bytes()
	err = scanner.Err()
	post.Source = filepath.Base(path)
	return
}

func getAllPosts(paths []string) ([]Post, error) {
	var allPosts []Post
	for _, path := range paths {
		post, err := ReadPost(path)
		if err != nil {
			return allPosts, err
		}
		post.Author = "Hal Canary"
		allPosts = append(allPosts, post)
	}
	sort.Sort(PostList(allPosts))
	for i, _ := range allPosts {
		if i > 0 {
			allPosts[i].Prev = &allPosts[i-1]
			allPosts[i-1].Next = &allPosts[i]
		}
	}
	return allPosts, nil
}
