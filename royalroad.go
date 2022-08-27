package main

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"strconv"
	"time"

	"golang.org/x/net/html"
)

func init() {
	Register(func(mainUrl string, populate bool) (EbookInfo, error) {
		var info EbookInfo
		mainUrlUrl, err := url.Parse(mainUrl)
		if err != nil {
			return info, err
		}
		if mainUrlUrl.Host != "www.royalroad.com" {
			return info, UnsupportedUrlError
		}
		main, err := GetUrl(mainUrl, "", true)
		if err != nil {
			return info, err
		}
		info.Source = mainUrl

		htmldoc, err := html.Parse(main)
		main.Close()
		if err != nil {
			return info, err
		}
		doc := (*Node)(htmldoc)
		info.Title = findOneMatchingNode2(doc, "meta", "name", "twitter:title").GetAttribute("content")
		if info.Title == "" {
			return info, fmt.Errorf("no title found: %q", mainUrl)
		}

		info.CoverURL = findOneMatchingNode2(doc, "meta", "property", "og:image").GetAttribute("content")
		info.Authors = findOneMatchingNode2(doc, "meta", "name", "twitter:creator").GetAttribute("content")

		descriptionNode := findOneMatchingNode2(doc, "div", "property", "description")
		info.Comments = descriptionNode.ExtractText()

		chapterTables := findOneMatchingNode2(doc, "table", "id", "chapters")

		for _, row := range findAllMatchingNodes(chapterTables, "tr") {
			link := findOneMatchingNode(row, "a")
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
			if t, _ := strconv.ParseInt(findOneMatchingNode(row, "time").GetAttribute("unixtime"), 10, 64); t != 0 {
				modified = time.Unix(t, 0)
			}
			info.Chapters = append(info.Chapters, Chapter{
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
			chData, err := GetUrl(chapter.Url, mainUrl, false)
			if err != nil {
				return info, err
			}
			ch, err := html.Parse(chData)
			if err != nil {
				return info, err
			}
			info.Chapters[i].Content = (*Node)(ch)
		}
		if charDevice {
			fmt.Fprint(os.Stderr, "\r           \r")
		}
		for i, chapter := range info.Chapters {
			info.Chapters[i].Content =
				Cleanup(
					findOneMatchingNode2(chapter.Content, "div", "class", "chapter-inner chapter-content").Remove())
		}
		info.Language = "en"
		return info, nil
	})
}
