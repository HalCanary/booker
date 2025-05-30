// Copyright 2022 Hal Canary
// Use of this program is governed by the file LICENSE.
package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/HalCanary/facility/ebook"
	"github.com/HalCanary/facility/email"
	"github.com/HalCanary/facility/humanize"
	"github.com/HalCanary/facility/tmpwriter"
	"github.com/HalCanary/facility/unorm"
	"github.com/HalCanary/facility/zipper"
)

func check(err error) {
	if err != nil {
		log.Output(2, err.Error())
		os.Exit(1)
	}
}

var (
	send                bool
	overwrite           bool
	htmlOut             bool
	convert             bool
	compress            bool
	maximumChapterCount int
	firstChapterIndex   int
	flagset             flag.FlagSet
	badfileRe           = regexp.MustCompile("[/\\?*|\"<>]+")
	apostropheRe        = regexp.MustCompile("[ʼ’‘:]")
)

func normalize(s string) string {
	return apostropheRe.ReplaceAllString(badfileRe.ReplaceAllString(unorm.Normalize(s), "_"), "'")
}

func init() {
	flagset.Init("", flag.ExitOnError)
	flagset.Usage = func() {
		cmd := os.Args[0]
		fmt.Fprintf(flagset.Output(), "Usage of %s:\n  %s [FLAGS] URL [MORE_URLS]\n\n", cmd, cmd)
		flagset.PrintDefaults()
	}
	flagset.BoolVar(&send, "send", false, "also send via email")
	flagset.BoolVar(&overwrite, "over", false, "force overwrite of output file")
	flagset.BoolVar(&htmlOut, "html", false, "output html only")
	flagset.BoolVar(&convert, "convert", false, "convert html to epub")
	flagset.BoolVar(&compress, "compress", false, "compress html to zip")
	flagset.IntVar(&maximumChapterCount, "maximumChapterCount", 0, "0 is unlimited")
	flagset.IntVar(&firstChapterIndex, "firstChapterIndex", 0, "")
	log.SetFlags(0)
}

var (
	secrets     email.EmailSecrets
	address     string
	destination string
)

func main() {
	flagset.Parse(os.Args[1:])
	if flagset.NArg() == 0 {
		flagset.Usage()
		os.Exit(2)
	}

	homeDir, err := os.UserHomeDir()
	check(err)

	if send {
		secrets, err = email.GetSecrets(filepath.Join(homeDir, ".email_secrets.json"))
		check(err)

		addressData, err := os.ReadFile(filepath.Join(homeDir, ".ebook_address"))
		check(err)
		address = strings.TrimSpace(string(addressData))
	}

	destination = filepath.Join(homeDir, "ebooks")
	check(os.MkdirAll(destination, 0o755))

	for _, arg := range flagset.Args() {
		if strings.HasPrefix(arg, "@") {
			lines, err := readFile(arg[1:])
			check(err)
			for _, s := range lines {
				s = strings.TrimSpace(s)
				if s != "" {
					err = handle(s, false)
					if err != nil {
						log.Println("Error: ", err)
					}
				}
			}
		} else {
			err = handle(arg, false)
			if err != nil {
				log.Println("Error: ", err)
			}
		}
	}
}

func readFile(path string) ([]string, error) {
	var result []string
	f, err := os.Open(path)
	if err == nil {
		defer f.Close()
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			result = append(result, scanner.Text())
		}
		err = scanner.Err()
	}
	return result, err
}

func fileSize(p string) int64 {
	fileInfo, err := os.Stat(p)
	if err != nil {
		return 0
	}
	return fileInfo.Size()
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func handle(arg string, pop bool) error {
	bk, err := ebook.DownloadEbook(arg, pop)
	if err != nil {
		return err
	}
	name := normalize(bk.Title)
	if name == "" {
		return errors.New("bad or missing book title")
	}
	if !bk.Modified.IsZero() {
		name = name + bk.Modified.UTC().Format(" [2006-01-02 150405]")
	}

	var epubPath string
	var htmlPath string
	var chapterCount = len(bk.Chapters)
	var cut = maximumChapterCount > 0 && maximumChapterCount+firstChapterIndex < chapterCount
	if cut || firstChapterIndex > 0 {
		if cut {
			chapterCount = maximumChapterCount + firstChapterIndex
			bk.Chapters = bk.Chapters[:chapterCount]
		}
		if firstChapterIndex > 0 && len(bk.Chapters) > firstChapterIndex {
			bk.Chapters = bk.Chapters[firstChapterIndex:]
		}
		epubPath = filepath.Join(destination,
			fmt.Sprintf("%s [%d-%d].epub", name, firstChapterIndex+1, chapterCount))
		htmlPath = filepath.Join(destination,
			fmt.Sprintf("%s [%d-%d].html", name, firstChapterIndex+1, chapterCount))

	} else {
		epubPath = filepath.Join(destination, name+".epub")
		htmlPath = filepath.Join(destination, name+".html")
	}

	if !overwrite {
		path := epubPath
		if htmlOut {
			path = htmlPath
		}
		if fileExists(path) {
			log.Printf("Already exists: %q\n", path)
			return nil
		}
	}

	if !pop {
		return handle(arg, true)
	}
	bk.Cleanup()

	if htmlOut {
		f, err := os.Create(htmlPath)
		if err != nil {
			return err
		}
		err = bk.WriteHtml(f)
		f.Close()
		if err != nil {
			return err
		}
		log.Printf("%7s written to %q\n", humanize.Humanize(fileSize(htmlPath)), htmlPath)

		if compress {
			zipPath := filepath.Join(destination, name+".zip")
			f, err = os.Create(zipPath)
			if err != nil {
				return err
			}
			zp := zipper.Make(f)
			bk.WriteHtml(zp.CreateDeflate(name+".html", bk.Modified))
			zp.Close()
			f.Close()
			log.Printf("%7s written to %q\n", humanize.Humanize(fileSize(zipPath)), zipPath)
		}
		var convertArgs []string
		if len(bk.Cover) > 0 {
			o, err := os.CreateTemp("", "")
			if err == nil {
				convertArgs = append(convertArgs, "--cover", o.Name())
				o.Write(bk.Cover)
				o.Close()
			}
		}
		if convert {
			if !overwrite && fileExists(epubPath) {
				log.Printf("Already exists: %q\n", epubPath)
				return nil
			}
			if err = ebook.ConvertToEbook(htmlPath, epubPath, convertArgs...); err != nil {
				return err
			}
			log.Printf("%7s written to %q\n", humanize.Humanize(fileSize(epubPath)), epubPath)
		}
	} else {
		f, err := tmpwriter.Make(epubPath)
		if err != nil {
			return err
		}
		if err = bk.Write(&f); err != nil {
			f.Reset()
			return err
		}
		size := f.Len()
		if err = f.Close(); err != nil {
			return err
		}
		log.Printf("%7s written to %q\n", humanize.Humanize(int64(size)), epubPath)
	}

	if send {
		const epubContentType = "application/epub+zip"
		if err = email.SendFile(email.Address{Address: address}, epubPath, epubContentType, secrets); err != nil {
			return err
		}
		log.Printf("Sent message to %q.\n\n", address)
	}
	return nil
}
