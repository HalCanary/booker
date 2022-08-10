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
)

func decode(src []byte) (image.Image, string, error) {
	var b bytes.Reader
	b.Reset(src)
	return image.Decode(&b)
}

func saveJpeg(src []byte) ([]byte, error) {
	img, fmt, err := decode(src)
	if err != nil {
		return nil, err
	}
	if fmt == "jpeg" {
		return src, nil
	}
	return writeJpeg(img, 80)
}

func writeJpeg(img image.Image, quality int) ([]byte, error) {
	var buffer bytes.Buffer
	options := jpeg.Options{Quality: quality}
	err := jpeg.Encode(&buffer, img, &options)
	return buffer.Bytes(), err
}

func saveJpegWithScale(src []byte, minWidth, minHeight int) ([]byte, error) {
	img, fmt, err := decode(src)
	if err != nil {
		return nil, err
	}
	imgSize := img.Bounds().Size()
	if fmt == "jpeg" && imgSize.X >= minWidth && imgSize.Y >= minHeight {
		return src, nil
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
	return writeJpeg(img, 80)
}
