// Copyright 2022 Hal Canary
// Use of this program is governed by the file LICENSE.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func check(err error) {
	if err != nil {
		log.Output(2, err.Error())
		os.Exit(1)
	}
}

func main() {
	var send bool
	flag.BoolVar(&send, "send", false, "also send via email")
	flag.Parse()

	var secrets EmailSecrets
	var address string
	if send {
		homeDir, err := os.UserHomeDir()
		check(err)

		secrets, err = GetSecrets(filepath.Join(homeDir, ".email_secrets.json"))
		check(err)

		addressData, err := os.ReadFile(filepath.Join(homeDir, ".ebook_address"))
		check(err)
		address = strings.TrimSpace(string(addressData))
	}

	var cache = "../cache" // filepath.Join(os.UserCacheDir(), "download")

	for _, arg := range flag.Args() {
		bk, err := Download(arg, cache)
		check(err)

		path, err := bk.Write("./dst", cache)
		if err == BookAlreadyExists {
			log.Printf("%q already exists.\n", path)
		} else {
			check(err)
			log.Println(path)
		}

		epubPath := path[:len(path)-len(filepath.Ext(path))] + ".epub"
		epubBase := filepath.Base(epubPath)

		if !exists(epubPath) {
			start := time.Now()
			err = EbookConvert(path, epubPath, bk)
			check(err)
			log.Printf("EbookConvert took %s\n", time.Now().Sub(start))
		}
		log.Println(epubPath)

		if send {
			data, err := os.ReadFile(epubPath)
			check(err)

			subject := fmt.Sprintf("(%s) %s", Humanize(len(data)), epubBase)
			err = Email{
				From:    secrets.FromAddr,
				To:      []string{address},
				Subject: subject,
				Attachments: []Attachment{
					Attachment{
						Data:        data,
						ContentType: "application/epub+zip",
						Filename:    epubBase,
					},
				},
			}.Send(secrets)
			check(err)
			log.Printf("Send message %q to %q.", subject, address)
		}
	}
}
