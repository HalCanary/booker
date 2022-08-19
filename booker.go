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
	//	"time"
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

	homeDir, err := os.UserHomeDir()
	check(err)

	var secrets EmailSecrets
	var address string
	if send {
		secrets, err = GetSecrets(filepath.Join(homeDir, ".email_secrets.json"))
		check(err)

		addressData, err := os.ReadFile(filepath.Join(homeDir, ".ebook_address"))
		check(err)
		address = strings.TrimSpace(string(addressData))
	}

	destination := filepath.Join(homeDir, "ebooks")

	for _, arg := range flag.Args() {
		bk, err := Download(arg)
		check(err)

		path, err := bk.Write(destination)
		if err == BookAlreadyExists {
			log.Printf("%q already exists.\n", path)
		} else {
			check(err)
			log.Println(path)
		}
		if send {
			err = SendFile(address, path, "application/epub+zip", secrets)
			check(err)
			log.Printf("Send message to %q.", address)
		}
	}
}

// Send a file to a single destination.
func SendFile(dst, path, contentType string, secrets EmailSecrets) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	base := filepath.Base(path)
	subject := fmt.Sprintf("(%s) %s", Humanize(len(data)), base)
	return Email{
		From:    secrets.FromAddr,
		To:      []string{dst},
		Subject: subject,
		Content: "☺",
		Attachments: []Attachment{
			Attachment{
				Data:        data,
				ContentType: contentType,
				Filename:    base,
			},
		},
	}.Send(secrets)
}
