package main

import (
	"sync"

	"gitlab.com/golang-commonmark/markdown"
)

var (
	mdOnce     sync.Once
	markdowner *markdown.Markdown
)

func Markdowner() *markdown.Markdown {
	mdOnce.Do(func() {
		markdowner = markdown.New(
			markdown.Breaks(false),
			markdown.HTML(true),
			markdown.LangPrefix("language-"),
			markdown.Linkify(false),
			markdown.MaxNesting(20),
			markdown.Nofollow(false),
			markdown.Tables(false),
			markdown.Typographer(false),
			markdown.XHTMLOutput(true),
		)
	})
	return markdowner
}
