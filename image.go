package main

// Copyright 2022 Hal Canary
// Use of this program is governed by the file LICENSE.

import (
	"bytes"
	"golang.org/x/image/draw"
	"image"
	"image/color"
	"image/jpeg"
	_ "image/png"
	"os"
)

func decode(src []byte) (image.Image, string, error) {
	var b bytes.Reader
	b.Reset(src)
	return image.Decode(&b)
}

func saveJpeg(src []byte, filename string) error {
	img, fmt, err := decode(src)
	if err != nil {
		return err
	}
	if fmt == "jpeg" {
		return os.WriteFile(filename, src, 0o644)
	}
	return writeJpeg(img, filename, 80)
}

func writeJpeg(img image.Image, filename string, quality int) error {
	options := jpeg.Options{Quality: quality}
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	return jpeg.Encode(file, img, &options)
}

func saveJpegWithScale(src []byte, filename string, minWidth, minHeight int) error {
	img, fmt, err := decode(src)
	if err != nil {
		return err
	}
	imgSize := img.Bounds().Size()
	if fmt == "jpeg" && imgSize.X >= minWidth && imgSize.Y >= minHeight {
		return os.WriteFile(filename, src, 0o644)
	}
	if imgSize.X < minWidth || imgSize.Y < minHeight {
		scale := float64(minWidth) / float64(imgSize.X)
		scaleY := float64(minHeight) / float64(imgSize.Y)
		if scaleY > scale {
			scale = scaleY
		}
		dst := image.NewNRGBA(image.Rectangle{
			Max: image.Point{int(float64(imgSize.X) * scale), int(float64(imgSize.Y) * scale)}})
		draw.Draw(dst, dst.Bounds(), &image.Uniform{&color.Gray{128}}, image.Point{}, draw.Src)
		draw.BiLinear.Scale(dst, dst.Bounds(), img, img.Bounds(), draw.Over, nil)
		img = dst
	}
	return writeJpeg(img, filename, 80)
}
