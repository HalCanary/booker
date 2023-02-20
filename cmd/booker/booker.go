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

	"github.com/HalCanary/booker/email"
	"github.com/HalCanary/booker/humanize"
	"github.com/HalCanary/booker/tmpwriter"
	"github.com/HalCanary/booker/unorm"
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
	badfileRe = regexp.MustCompile("[/\\?*|\"<>]+")
	//badfileRe = regexp.MustCompile("[^A-Za-z0-9._+-]+")
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
			lines, err := readFile(arg[1:])
			check(err)
			for _, s := range lines {
				s = strings.TrimSpace(s)
				if s != "" {
					handle(s, false)
				}
			}
		} else {
			handle(arg, false)
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

func handle(arg string, pop bool) error {
	bk, err := Download(arg, pop)
	if err != nil {
		return err
	}

	name := badfileRe.ReplaceAllString(unorm.Normalize(bk.Title), "_")
	if name == "" {
		return errors.New("bad or missing book title")
	}
	if !bk.Modified.IsZero() {
		name = name + bk.Modified.UTC().Format(" [2006-01-02 15:04:05]")
	}
	path := filepath.Join(destination, name+".epub")

	if !overwrite {
		_, err := os.Stat(path)
		if err == nil {
			log.Printf("Already exists: %q\n", path)
			return nil
		}
	}
	if !pop {
		return handle(arg, true)
	}

	f, err := tmpwriter.Make(path)
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
	log.Printf("%7s written to %q\n", humanize.Humanize(int(size)), path)

	if send {
		const epubContentType = "application/epub+zip"
		if err = email.SendFile(email.Address{Address: address}, path, epubContentType, secrets); err != nil {
			return err
		}
		log.Printf("Sent message to %q.\n\n", address)
	}
	return nil
}
