package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var newlineRegexp *regexp.Regexp = regexp.MustCompile("\n")

// Call Calibre's `ebook-convert` command with metadata from `info`.
func EbookConvert(src, dst string, info EbookInfo) error {
	os.MkdirAll(filepath.Dir(dst), 0o755)
	cmd := exec.Command(
		"ebook-convert",
		src,
		dst,
		"--title", info.Title,
		"--authors", info.Authors,
		"--cover", info.CoverPath,
		"--language", info.Language,
		"--pubdate", info.Modified.Format(time.RFC3339),
		"--comments", newlineRegexp.ReplaceAllString(strings.TrimSpace(info.Comments), " ¶ "),
	)
	stdoutStderr, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Error: %w\nCmd:%#v\nCombinedOutput:\n%s", err, cmd.Args, string(stdoutStderr))
	}
	return nil
}
