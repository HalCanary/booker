package main

// Copyright 2022 Hal Canary
// Use of this program is governed by the file LICENSE.

import (
	"crypto/md5"
	"encoding/hex"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
)

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

const (
	accept    = "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9"
	userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/103.0.0.0 Safari/537.36"
)

var (
	cacheDirOnce sync.Once
	cacheDir     string
)

// Fetch the content of a URL, using a cache if possible and if force is fakse.
func GetUrl(url, ref string, force bool) (io.ReadCloser, error) {
	cacheDirOnce.Do(func() {
		cache, err := os.UserCacheDir()
		if err != nil {
			log.Fatal(err)
		}
		cacheDir = cache + "/urlcache"
	})
	uhashbytes := md5.Sum([]byte(url))
	uhash := hex.EncodeToString(uhashbytes[:])
	cache := cacheDir + "/" + uhash
	if force || !exists(cache) {
		if err := os.MkdirAll(cacheDir, 0o755); err != nil {
			return nil, err
		}
		req, err := http.NewRequest("GET", url, nil)
		if ref != "" {
			req.Header.Add("Referer", ref)
		}
		req.Header.Add("accept", accept)
		// req.Header.Add("accept-encoding", "gzip")
		req.Header.Add("accept-language", "en-US,en;q=0.9")
		req.Header.Add("user-agent", userAgent)
		client := http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		if err = os.WriteFile(cacheDir+"/"+uhash+"_type",
			[]byte(resp.Header.Get("Content-Type")), 0o644); err != nil {
			return nil, err
		}
		bodyWriter, err := os.Create(cache)
		if err != nil {
			return nil, err
		}
		_, err = io.Copy(bodyWriter, resp.Body)
		if err != nil {
			return nil, err
		}
		resp.Body.Close()
		bodyWriter.Close()
	}
	return os.Open(cache)
}
