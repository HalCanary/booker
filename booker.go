// Copyright 2022 Hal Canary
// Use of this program is governed by the file LICENSE.
package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func check(err error) {
	if err != nil {
		log.Output(2, err.Error())
		os.Exit(1)
	}
}

var (
	send      bool
	overwrite bool
	flagset   flag.FlagSet
)

func init() {
	flagset.Init("", flag.ExitOnError)
	flagset.Usage = func() {
		cmd := os.Args[0]
		fmt.Fprintf(flagset.Output(), "Usage of %s:\n  %s [FLAGS] URL [MORE_URLS]\n\n", cmd, cmd)
		flagset.PrintDefaults()
	}
	flagset.BoolVar(&send, "send", false, "also send via email")
	flagset.BoolVar(&overwrite, "over", false, "force overwrite of output file")
	log.SetFlags(0)
}

func main() {
	flagset.Parse(os.Args[1:])
	if flagset.NArg() == 0 {
		flagset.Usage()
		os.Exit(2)
	}

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
	check(os.MkdirAll(destination, 0o755))

	for _, arg := range flagset.Args() {
		bk, err := Download(arg)
		check(err)

		name := bk.Name()
		if name == "" {
			check(errors.New("no name :("))
		}
		path := filepath.Join(destination, name+".epub")

		if !overwrite && exists(path) {
			log.Printf("%q already exists.\n\n", path)
			continue
		}
		f, err := os.Create(path)
		check(err)
		defer f.Close()

		check(bk.Write(f))
		log.Printf("%q written\n\n", path)

		if send {
			check(SendFile(address, path, "application/epub+zip", secrets))
			log.Printf("Sent message to %q.\n\n", address)
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
