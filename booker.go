// Copyright 2022 Hal Canary
// Use of this program is governed by the file LICENSE.
package main

import (
	"os"
	"log"
)

func main() {
	var cache = "../cache"
	for _, arg := range os.Args[1:] {
		bk, err := Download(arg, cache)
		if err != nil {
			log.Fatal(err)
		}
		path, e := Write(bk, "./dst", cache)
		if e != nil {
			log.Fatal(e)
		}
		log.Println(path)
		e = EbookConvert(path, path + ".epub", bk)
		if e != nil {
			log.Fatal(e)
		}
		log.Println(path + ".epub")
	}
}
