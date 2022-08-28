package ebook

import (
	"errors"
)

// Returned by a downloadFunction when the URL can not be handled.
var UnsupportedUrlError = errors.New("unsupported url")

var registerdFunctions []func(url string, pop bool) (EbookInfo, error)

// Register the given function.
func Register(downloadFunction func(url string, pop bool) (EbookInfo, error)) {
	registerdFunctions = append(registerdFunctions, downloadFunction)
}

// Return the result of the first registered download function that does not
// return UnsupportedUrlError.
// @param url - the URL of the title page of the book.
// @param pop - set to true to download and populate the entire EbookInfo data
//              structure, not just it's metadata.
func Download(url string, pop bool) (EbookInfo, error) {
	for _, fn := range registerdFunctions {
		info, err := fn(url, pop)
		if err != UnsupportedUrlError {
			return info, err
		}
	}
	return EbookInfo{}, UnsupportedUrlError
}
