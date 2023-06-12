package main

// Copyright 2022 Hal Canary
// Use of this program is governed by the file LICENSE.

import (
	"fmt"
	"io"
	"log"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/HalCanary/facility/spinner"
	"github.com/HalCanary/facility/dom"
	"github.com/HalCanary/facility/download"
	"github.com/HalCanary/facility/ebook"
)

var (
	stripRe      = regexp.MustCompile("(?:^\\s+)|(?:\\s+$)")
	whitespaceRe = regexp.MustCompile("\\s+")
)

func getAttribute(dst *string, root *dom.Node, tag, key, value, attribute string) {
	if *dst == "" {
		*dst = dom.GetAttribute(dom.FindNodeByTagAndAttrib(root, tag, key, value), attribute)
	}
}

func getTextByTagAndAttrib(dst *string, root *dom.Node, tag, key, value string) {
	if *dst == "" {
		*dst = strings.TrimSpace(dom.ExtractText(dom.FindNodeByTagAndAttrib(root, tag, key, value)))
	}
}

func populateInfo(info *ebook.EbookInfo, doc *dom.Node) {
	if info == nil || doc == nil {
		return
	}
	var coverURL string
	getAttribute(&info.Title, doc, "meta", "name", "twitter:title", "content")
	getTextByTagAndAttrib(&info.Title, doc, "h1", "", "")
	getAttribute(&info.Authors, doc, "meta", "property", "books:author", "content")
	getAttribute(&info.Authors, doc, "meta", "name", "twitter:creator", "content")
	getTextByTagAndAttrib(&info.Authors, doc, "a", "rel", "author")
	getTextByTagAndAttrib(&info.Comments, doc, "div", "property", "description")
	getTextByTagAndAttrib(&info.Comments, doc, "div", "class", "description")
	getAttribute(&info.Comments, doc, "meta", "property", "og:description", "content")
	getAttribute(&info.Language, doc, "html", "", "", "lang")
	getAttribute(&coverURL, doc, "meta", "property", "og:image", "content")
	if coverURL != "" {
		rc, err := download.GetUrl(coverURL, "", false)
		if err != nil {
			log.Printf("Error: cover download: %s\n", err)
		} else {
			info.Cover, _ = io.ReadAll(rc)
			rc.Close()
		}
	}
}

func init() {
	Register(func(mainUrl string, populate bool) (ebook.EbookInfo, error) {
		var info ebook.EbookInfo
		mainUrlUrl, err := url.Parse(mainUrl)
		if err != nil {
			return info, err
		}
		if mainUrlUrl.Host != "www.royalroad.com" {
			return info, UnsupportedUrlError
		}
		main, err := download.GetUrl(mainUrl, "", true)
		if err != nil {
			return info, err
		}
		info.Source = mainUrl

		doc, err := dom.Parse(main)
		main.Close()
		if err != nil {
			return info, err
		}
		populateInfo(&info, doc)

		if info.Title == "" {
			return info, fmt.Errorf("Error: no title found: %q", mainUrl)
		}

		chapterTables := dom.FindNodeByTagAndAttrib(doc, "table", "id", "chapters")
		for _, row := range dom.FindNodesByTagAndAttrib(chapterTables, "tr", "", "") {
			link := dom.FindNodeByTag(row, "a")
			path := dom.GetAttribute(link, "href")
			title := stripRe.ReplaceAllString(whitespaceRe.ReplaceAllString(dom.ExtractText(link), " "), "")
			if path == "" || title == "" {
				continue
			}
			pathUrl, err := url.Parse(path)
			if err != nil {
				log.Println(err)
				continue
			}
			var modified time.Time

			unixtime := dom.GetAttribute(dom.FindNodeByTagAndAttrib(row, "time", "", ""), "unixtime")
			if t, _ := strconv.ParseInt(unixtime, 10, 64); t != 0 {
				modified = time.Unix(t, 0)
			}
			info.Chapters = append(info.Chapters, ebook.Chapter{
				Title:    title,
				Url:      mainUrlUrl.ResolveReference(pathUrl).String(),
				Modified: modified,
			})
		}
		info.Modified = info.CalculateLastModified()
		if !populate {
			info.Chapters = nil
			return info, nil
		}
		log.Printf("%q -> discovered %d chapters (%s)\n", info.Title, len(info.Chapters), info.Modified)

		spin := spinner.NewTerminalSpinner()
		for i, chapter := range info.Chapters {
			spin.Printf("[%d/%d] ", i+1, len(info.Chapters))
			chData, err := download.GetUrl(chapter.Url, mainUrl, false)
			if err != nil {
				return info, err
			}
			ch, err := dom.Parse(chData)
			chData.Close()
			if err != nil {
				return info, err
			}
			content := dom.FindNodeByTagAndAttrib(ch, "div", "class", "chapter-inner chapter-content")
			if content == nil {
				return info, fmt.Errorf("Missing chapter content: %q", chapter.Url)
			}
			info.Chapters[i].Content = dom.Remove(content)
		}
		spin.Printf("")
		return info, nil
	})
}
