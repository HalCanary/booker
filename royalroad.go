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
	Register(func(mainUrl string, cachePath string) (EbookInfo, error) {
		var info EbookInfo
		mainUrlUrl, err := url.Parse(mainUrl)
		if err != nil {
			return info, err
		}
		if mainUrlUrl.Host != "www.royalroad.com" {
			return info, UnsupportedUrlError
		}
		main, _, err := GetUrl(mainUrl, cachePath, "", true)
		if err != nil {
			return info, err
		}
		info.Source = mainUrl

		doc, err := html.Parse(main)
		main.Close()
		if err != nil {
			return info, err
		}
		info.Title = GetAttribute(findOneMatchingNode2(doc, "meta", "name", "twitter:title"), "content")
		info.Cover = GetAttribute(findOneMatchingNode2(doc, "meta", "property", "og:image"), "content")
		info.Authors = GetAttribute(findOneMatchingNode2(doc, "meta", "name", "twitter:creator"), "content")

		descriptionNode := findOneMatchingNode2(doc, "div", "property", "description")
		info.Comments = ExtractText(descriptionNode)

		chapterTables := findOneMatchingNode2(doc, "table", "id", "chapters")

		for _, row := range findAllMatchingNodes(chapterTables, "tr") {
			link := findOneMatchingNode(row, "a")
			path := GetAttribute(link, "href")
			title := stripRe.ReplaceAllString(whitespaceRe.ReplaceAllString(ExtractText(link), " "), "")
			if path == "" || title == "" {
				continue
			}
			pathUrl, err := url.Parse(path)
			if err != nil {
				log.Println(err)
				continue
			}
			var modified time.Time
			if t, _ := strconv.ParseInt(GetAttribute(findOneMatchingNode(row, "time"), "unixtime"), 10, 64); t != 0 {
				modified = time.Unix(t, 0)
			}
			info.Chapters = append(info.Chapters, Chapter{
				Title:    title,
				Url:      mainUrlUrl.ResolveReference(pathUrl).String(),
				Modified: modified,
			})
		}
		info.Modified = CalculateLastModified(info.Chapters)
		log.Printf("%q -> discovered %d chapters (%s)\n", info.Title, len(info.Chapters), info.Modified)
		for i, chapter := range info.Chapters {
			fmt.Fprintf(os.Stderr, "%d ", i+1)
			chData, _, err := GetUrl(chapter.Url, cachePath, mainUrl, false)
			if err != nil {
				return info, err
			}
			ch, err := html.Parse(chData)
			if err != nil {
				return info, err
			}
			info.Chapters[i].Content =
				Cleanup(Remove(
					findOneMatchingNode2(ch, "div", "class", "chapter-inner chapter-content")))
		}
		fmt.Fprintln(os.Stderr)
		return info, nil
	})
}
