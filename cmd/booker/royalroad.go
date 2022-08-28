package main

// Copyright 2022 Hal Canary
// Use of this program is governed by the file LICENSE.

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/HalCanary/booker/dom"
	"github.com/HalCanary/booker/download"
	"github.com/HalCanary/booker/ebook"
)

type node = dom.Node

var (
	stripRe      = regexp.MustCompile("(?:^\\s+)|(?:\\s+$)")
	whitespaceRe = regexp.MustCompile("\\s+")
)

func init() {
	ebook.Register(func(mainUrl string, populate bool) (ebook.EbookInfo, error) {
		var info ebook.EbookInfo
		mainUrlUrl, err := url.Parse(mainUrl)
		if err != nil {
			return info, err
		}
		if mainUrlUrl.Host != "www.royalroad.com" {
			return info, ebook.UnsupportedUrlError
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
		info.Title = doc.FindOneMatchingNode2("meta", "name", "twitter:title").GetAttribute("content")
		if info.Title == "" {
			return info, fmt.Errorf("no title found: %q", mainUrl)
		}

		info.CoverURL = doc.FindOneMatchingNode2("meta", "property", "og:image").GetAttribute("content")
		info.Authors = doc.FindOneMatchingNode2("meta", "name", "twitter:creator").GetAttribute("content")

		descriptionNode := doc.FindOneMatchingNode2("div", "property", "description")
		info.Comments = descriptionNode.ExtractText()

		chapterTables := doc.FindOneMatchingNode2("table", "id", "chapters")

		for _, row := range chapterTables.FindAllMatchingNodes("tr") {
			link := row.FindOneMatchingNode("a")
			path := link.GetAttribute("href")
			title := stripRe.ReplaceAllString(whitespaceRe.ReplaceAllString(link.ExtractText(), " "), "")
			if path == "" || title == "" {
				continue
			}
			pathUrl, err := url.Parse(path)
			if err != nil {
				log.Println(err)
				continue
			}
			var modified time.Time
			if t, _ := strconv.ParseInt(row.FindOneMatchingNode("time").GetAttribute("unixtime"), 10, 64); t != 0 {
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
		stderrStat, _ := os.Stderr.Stat()
		charDevice := stderrStat.Mode()&os.ModeCharDevice != 0
		for i, chapter := range info.Chapters {
			if charDevice {
				fmt.Fprintf(os.Stderr, "\r[%d/%d]   ", i+1, len(info.Chapters))
			}
			chData, err := download.GetUrl(chapter.Url, mainUrl, false)
			if err != nil {
				return info, err
			}
			ch, err := dom.Parse(chData)
			chData.Close()
			if err != nil {
				return info, err
			}
			info.Chapters[i].Content = ch.FindOneMatchingNode2(
				"div", "class", "chapter-inner chapter-content").Remove()

		}
		if charDevice {
			fmt.Fprint(os.Stderr, "\r           \r")
		}
		info.Language = "en"
		return info, nil
	})
}
