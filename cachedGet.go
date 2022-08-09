package main

// Copyright 2022 Hal Canary
// Use of this program is governed by the file LICENSE.

import (
	"crypto/md5"
	"encoding/hex"
	"io"
	"net/http"
	"os"
)

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

const (
	accept    = "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9"
	userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/103.0.0.0 Safari/537.36"
)

// Fetch the content of a URL, using a cache if possible and if force is fakse.
func GetUrl(url, cacheDir, ref string, force bool) (io.ReadCloser, string, error) {
	uhashbytes := md5.Sum([]byte(url))
	uhash := hex.EncodeToString(uhashbytes[:])
	cache := cacheDir + "/" + uhash
	tcache := cacheDir + "/" + uhash + "_type"
	var contentType string
	if force || !exists(cache) || !exists(tcache) {
		if err := os.MkdirAll(cacheDir, 0o755); err != nil {
			return nil, "", err
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
			return nil, "", err
		}
		contentType = resp.Header.Get("Content-Type")
		if err = os.WriteFile(tcache, []byte(contentType), 0o644); err != nil {
			return nil, "", err
		}
		bodyWriter, err := os.Create(cache)
		if err != nil {
			return nil, "", err
		}
		_, err = io.Copy(bodyWriter, resp.Body)
		if err != nil {
			return nil, "", err
		}
		resp.Body.Close()
		bodyWriter.Close()
	}
	if contentType == "" {
		typeBytes, err := os.ReadFile(tcache)
		if err != nil {
			return nil, "", err
		}
		contentType = string(typeBytes)
	}
	body, err := os.Open(cache)
	return body, contentType, err
}
