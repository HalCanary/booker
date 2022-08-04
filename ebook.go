package main

import (
	"os/exec"
	"os"
)

type EbookInfo struct {
	Authors  string
	Cover    string
	Comments string
	Title    string
}

func EbookConvert(src, dst string, info EbookInfo) error {
	cmd := exec.Command(
		"ebook-convert",
		src,
		dst,
		"--authors", info.Authors,
		"--cover", info.Cover,
		"--comments", info.Comments,
		"--title", info.Title,
	)
	stdoutStderr, err := cmd.CombinedOutput()
	if err != nil {
		os.Stderr.Write(stdoutStderr)
	}
	return err
}
