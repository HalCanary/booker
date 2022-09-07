// Copyright 2022 Hal Canary
// Use of this program is governed by the file LICENSE.
package main

import (
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/HalCanary/booker/dom"
)

const (
	icon  = "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAIAAAACAAQAAAADrRVxmAAAACXBIWXMAAAMfAAADHwHmEQywAAAAGXRFWHRTb2Z0d2FyZQB3d3cuaW5rc2NhcGUub3Jnm+48GgAAAVZJREFUSMfd1EuOwyAMAFAQ0rAZNUfIUTga5GbsZtkrcINhyQLBYMgHG6ZddJcoSpOXDzY2ZZls7F5Q9iAzhyu/XsAomCm4AdRHYPUUNAH7GTgdRbZd+k6FCZgGZQPw/DV45dgIWw+rc3rrRvGrHcBrjmDzqocAIBCIsr+ERcYFwYNAfMj0QPC9pC+JIWNgpfJvQGXTQwJYCFgMegSHIWPIA/DaplO49UIuEy7qUR5gWakJHA+oXZhNZYBYQNQnlgaBKSthrRjRwJXfcu+ZA28A5fmF7yXWwKg+BgB9AJyxPtgGib8FQSCKjEcZQR7XqUUaTgiCgJME9vRPMEwREJk+McCKodUBDUvjiDTSfU4v2NPvwCqc7QHif3Bz6Ga9vZIYga5yVs9rO1Qf+uOnSw6yhnKu2Yurx+C/Ru8TVLuQl9HPtmx9CqCvTlYwxxz3eqhv3Htl/wEokJpySHNGkgAAAABJRU5ErkJggg=="
	style = `
@media (prefers-color-scheme:dark) {body {background-color:#000;color:#FFF;}
a:visited {color:#C0F;}
a:link, a:hover, a:active {color:#0CF;}
}
@media print {body {max-width:8in;font-size:12px;margin:0;}
}
@media screen {body {font-family:sans-serif;max-width:35em;margin:22px auto 64px auto;padding:0 8px;}
}
body {overflow-wrap:break-word;}
@page {size:auto;margin:0.25in 0.5in 0.5in 0.5in;}
svg {fill:currentColor;}
img {max-width:100%;height:auto;}
hr {border-style:solid none;}
.content {margin:1em 0;}
.content hr {padding:0;margin:0;border:none;text-align:center;}
.content hr:after {font-size:150%;content:"* \A0 * \A0 *";display:block;position:relative;}
.rightside {text-align:right;}
.centered {text-align:center;}
pre {overflow-x:auto;}
.byline > * {display:inline-block;border-style:solid;border-width:thin;padding:3px 8px;border-radius:5px;text-align:initial;}
.byline {text-align:right;}
.box {border-style:solid;border-width:thin;margin:8px 0;padding:0 8px;}
a.hiddenlink:link {background:inherit;color:inherit;text-decoration:none;}
a.hiddenlink:visited {background:inherit;color:inherit;text-decoration:none;}
a.hiddenlink:active {background:inherit;color:inherit;text-decoration:none;}
ul,ol {padding-left:30px;}
table.border {border-collapse:collapse;margin:8px auto;}
table.border tr > * {border-style:solid;border-width:thin;padding:3px 8px;border-radius:5px;}
.tophead {text-align:center;margin:1ex auto 0 auto;max-width:35em;}
.tightmargins li > ul > li {list-style-type:square;}
.tightmargins h1,
.tightmargins h2,
.tightmargins p,
.tightmargins ul {margin:0.5ex 0;}
.tightmargins li > ul {margin:0 0 0.5ex 0;}
.tightmargins ul {padding-left:30px;}
.tightmargins li {margin:0 0 0.5ex 0;}
.plainlink a:link,
.plainlink a:visited,
.plainlink a:hover,
.plainlink a:active {color:inherit;text-decoration:underline;}
.nolink a:link,
.nolink a:visited,
.nolink a:hover,
.nolink a:active {color:inherit;text-decoration:none;}
div.lcr {display:grid;grid-template-columns:auto auto auto;}
ul.flat {list-style-type:none;margin:16px 0;padding:0;}
ul.flat li {display:inline;}
ul.flat li::after {content:"]";}
ul.flat li::before {content:"[";}`
)

func concat(strs ...string) string {
	return strings.Join(strs, "")
}

func link(dst, text string) *dom.Node {
	return dom.Element("a", dom.Attr{"href": dst}, dom.TextNode(text))
}

func header(level int, attributes dom.Attr, children ...*dom.Node) *dom.Node {
	return dom.Element("h"+strconv.Itoa(level), attributes, children...)
}

func main() {
	log.SetFlags(0)

	if len(os.Args) < 2 {
		log.Fatal("usage?")
		return
	}
	directory := os.Args[1]
	matches, err := filepath.Glob(directory + "/src/BlogSrc/*.md")
	if err != nil {
		log.Fatal(err)
	}
	allPosts, err := getAllPosts(matches)
	if err != nil {
		log.Fatal(err)
	}
	for _, post := range allPosts {
		f, err := os.Create(concat(directory+"/docs/vv/", post.LongId(), "/index.html"))
		if err != nil {
			log.Fatal(err)
		}
		makeIndividualPost(post).RenderHTML(f)
	}
}

func makeHead(title string) *dom.Node {
	return dom.Elem("head",
		dom.TextNode("\n"),
		dom.Element("meta", dom.Attr{"charset": "utf-8"}),
		dom.TextNode("\n"),
		dom.Element("meta", dom.Attr{
			"name": "viewport", "content": "width=device-width, initial-scale=1.0"}),
		dom.TextNode("\n"),
		dom.Elem("title", dom.TextNode(title)),
		dom.TextNode("\n"),
		dom.Element("link", dom.Attr{"rel": "icon", "href": icon}),
		dom.TextNode("\n"),
		dom.Elem("style", dom.TextNode(style+"\n")),
		dom.TextNode("\n"),
	)
}

func makeIndividualPost(post Post) *dom.Node {
	const titlePrefix = "Voder-Vocoder: "
	head := makeHead(titlePrefix + post.Title)
	head.Append(
		dom.Element("link", dom.Attr{"rel": "alternate", "type": "application/atom+xml", "title": "/vv/", "href": "/vv/rss.rss"}),
		dom.TextNode("\n"),
		dom.Comment(" Copyright 2002-2022 Hal Canary. ALL RIGHTS RESERVED. "),
		dom.TextNode("\n"),
	)
	return dom.Element("html", dom.Attr{"lang": "en"},
		dom.TextNode("\n"),
		head,
		dom.TextNode("\n"),
		dom.Elem("body",
			dom.TextNode("\n"),
			post.Article(1),
			dom.TextNode("\n"),
			dom.Elem("hr"),
			dom.TextNode("\n"),
			dom.Elem("nav",
				dom.TextNode("\n"),
				dom.Element("div", dom.Attr{"class": "lcr"},
					dom.TextNode("\n"),
					dom.Elem("div", post.PrevLink()),
					dom.TextNode("\n"),
					dom.Element("div", dom.Attr{"class": "centered"},
						dom.Elem("p",
							dom.TextNode("("),
							link("/vv/archives/", "back"),
							dom.TextNode(")"),
						),
					),
					dom.TextNode("\n"),
					dom.Element("div", dom.Attr{"class": "rightside"}, post.NextLink()),
					dom.TextNode("\n"),
				),
				dom.TextNode("\n"),
			),
			dom.TextNode("\n"),
		),
		dom.TextNode("\n"),
	)
}
