package main

import (
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

var magicTransformerChain = transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)

// Simplify and normalize a Unicode string.
func NormalizeString(v string) string {
	result, _, err := transform.String(magicTransformerChain, v)
	if err != nil {
		return v
	}
	return result
}
