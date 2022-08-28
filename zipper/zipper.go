package zipper

import (
	"archive/zip"
	"io"
	"time"
)

type Zipper struct {
	ZipWriter *zip.Writer
	Error     error
}

func Make(dst io.Writer) Zipper {
	return Zipper{zip.NewWriter(dst), nil}
}

func (zw *Zipper) Close() {
	err := zw.ZipWriter.Close()
	if zw.Error == nil {
		zw.Error = err
	}
}

func (zw *Zipper) create(name string, method uint16, mod time.Time) io.Writer {
	if !mod.IsZero() {
		mod = mod.UTC()
	}
	if zw.Error == nil {
		var w io.Writer
		if w, zw.Error = zw.ZipWriter.CreateHeader(&zip.FileHeader{
			Name:     name,
			Modified: mod,
			Method:   method,
		}); zw.Error == nil {
			return w
		}
	}
	return nil
}

func (zw *Zipper) CreateDeflate(name string, mod time.Time) io.Writer {
	return zw.create(name, zip.Deflate, mod)
}

func (zw *Zipper) CreateStore(name string, mod time.Time) io.Writer {
	return zw.create(name, zip.Store, mod)
}
