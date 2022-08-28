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
	"strings"

	"github.com/HalCanary/booker/ebook"
	"github.com/HalCanary/booker/email"
	"github.com/HalCanary/booker/humanize"
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
			handleUrlFile(arg[1:])
		} else {
			handle(arg, false)
		}
	}
}

func handleUrlFile(path string) {
	f, err := os.Open(path)
	check(err)
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		s := strings.TrimSpace(scanner.Text())
		if s != "" {
			handle(s, false)
		}
	}
	check(scanner.Err())
}

func handle(arg string, pop bool) {
	bk, err := ebook.Download(arg, pop)
	check(err)

	name := bk.Name()
	if name == "" {
		check(errors.New("no name :("))
	}
	path := filepath.Join(destination, name+".epub")

	if !overwrite {
		_, err := os.Stat(path)
		if err == nil {
			log.Printf("%q already exists.\n\n", path)
			return
		}
	}
	if !pop {
		handle(arg, true)
		return
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

// Send a file to a single destination.
func SendFile(dst, path, contentType string, secrets email.EmailSecrets) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	base := filepath.Base(path)
	subject := fmt.Sprintf("(%s) %s", humanize.Humanize(len(data)), base)
	return email.Email{
		From:    secrets.FromAddr,
		To:      []string{dst},
		Subject: subject,
		Content: "☺",
		Attachments: []email.Attachment{
			email.Attachment{
				Data:        data,
				ContentType: contentType,
				Filename:    base,
			},
		},
	}.Send(secrets)
}
