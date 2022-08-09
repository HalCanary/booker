package main

import (
	"fmt"
	"os/exec"
	"regexp"
)

var newlineRegexp *regexp.Regexp = regexp.MustCompile("\n")

// Call Calibre's `ebook-convert` command with metadata from `info`.
func EbookConvert(src, dst string, info EbookInfo) error {
	cmd := exec.Command(
		"ebook-convert",
		src,
		dst,
		"--title", info.Title,
		"--authors", info.Authors,
		"--cover", info.Cover,
		"--comments", newlineRegexp.ReplaceAllString(info.Comments, " ¶ "),
	)
	stdoutStderr, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Error: %w\nCombinedOutput:\n%s", err, string(stdoutStderr))
	}
	return nil
}
