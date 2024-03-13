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
	"time"

	"github.com/HalCanary/facility/dom"
	"github.com/HalCanary/facility/download"
	"github.com/HalCanary/facility/ebook"
	"github.com/HalCanary/facility/spinner"
)

var (
	stripRe       = regexp.MustCompile("(?:^\\s+)|(?:\\s+$)")
	whitespaceRe  = regexp.MustCompile("\\s+")
	sWarningRe    = regexp.MustCompile("^c[^n].{42}$")
	labelClassRe  = regexp.MustCompile("\\blabel\\b")
	isCompletedRe = regexp.MustCompile("\\s*COMPLETED\\s*")
	isStubRe      = regexp.MustCompile("\\s*STUB\\s*")
)

func init() {
	ebook.RegisterEbookGenerator(generateRREbook)
}

func generateRREbook(mainUrl string, populate bool) (ebook.EbookInfo, error) {
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
	ebook.PopulateInfo(&info, doc)
	coverURL := dom.GetAttribute(dom.FindNodeByTagAndAttrib(doc, "meta", "property", "og:image"), "content")
	if coverURL != "" {
		rc, err := download.GetUrl(coverURL, "", false)
		if err != nil {
			log.Printf("Error: cover download: %s\n", err)
		} else {
			info.Cover, _ = io.ReadAll(rc)
			rc.Close()
		}
	}

	if info.Title == "" {
		return info, fmt.Errorf("Error: no title found: %q", mainUrl)
	}

	for _, label := range dom.FindNodesByTagAndAttribRe(doc, "", "class", labelClassRe) {
		labelString := dom.ExtractText(label)
		if isStubRe.MatchString(labelString) {
			info.Title += " [STUB]"
		} else if isCompletedRe.MatchString(labelString) {
			info.Title += " [COMPLETE]"
		}
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
		for _, swn := range dom.FindNodesByTagAndAttribRe(info.Chapters[i].Content,
			"p", "class", sWarningRe) {
			dom.Remove(swn)
		}
	}
	spin.Printf("")
	return info, nil
}
