package main

// Copyright 2022 Hal Canary
// Use of this program is governed by the file LICENSE.

import (
	"errors"
	"sync"

	"github.com/HalCanary/facility/ebook"
)

// Returned by a downloadFunction when the URL can not be handled.
var UnsupportedUrlError = errors.New("unsupported url")

var registerdFunctions []func(url string, pop bool) (ebook.EbookInfo, error)
var registerdFunctionsMutex sync.Mutex

// Register the given function.
func Register(downloadFunction func(url string, pop bool) (ebook.EbookInfo, error)) {
	registerdFunctionsMutex.Lock()
	registerdFunctions = append(registerdFunctions, downloadFunction)
	registerdFunctionsMutex.Unlock()
}

// Return the result of the first registered download function that does not
// return UnsupportedUrlError.
// @param url - the URL of the title page of the book.
// @param pop - set to true to download and populate the entire EbookInfo data
//
//	structure, not just it's metadata.
func Download(url string, pop bool) (ebook.EbookInfo, error) {
	for _, fn := range registerdFunctions {
		info, err := fn(url, pop)
		if err != UnsupportedUrlError {
			return info, err
		}
	}
	return ebook.EbookInfo{}, UnsupportedUrlError
}
