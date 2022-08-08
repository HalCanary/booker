package main

import (
	"errors"
)

var UnsupportedUrlError = errors.New("unsupported url")

type DownloadFunction = func(url, cachePath string) (EbookInfo, error)

var registerdFunctions []DownloadFunction

func Register(fn DownloadFunction) {
	registerdFunctions = append(registerdFunctions, fn)
}

func Download(url, cachePath string) (EbookInfo, error) {
	for _, fn := range registerdFunctions {
		info, err := fn(url, cachePath)
		if err != UnsupportedUrlError {
			return info, err
		}
	}
	return EbookInfo{}, UnsupportedUrlError
}
