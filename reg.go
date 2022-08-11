package main

import (
	"errors"
)

// Returned by a downloadFunction when the URL can not be handled.
var UnsupportedUrlError = errors.New("unsupported url")

var registerdFunctions []func(url string) (EbookInfo, error)

// Register the given function.
func Register(downloadFunction func(url string) (EbookInfo, error)) {
	registerdFunctions = append(registerdFunctions, downloadFunction)
}

// Return the result of the first registered download function that does not
// return UnsupportedUrlError.
func Download(url string) (EbookInfo, error) {
	for _, fn := range registerdFunctions {
		info, err := fn(url)
		if err != UnsupportedUrlError {
			return info, err
		}
	}
	return EbookInfo{}, UnsupportedUrlError
}
