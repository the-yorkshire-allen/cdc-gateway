
package util

import (
	"bytes"
	"compress/gzip"
	"io"
)

func Gzip(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	zw, _ := gzip.NewWriterLevel(&buf, gzip.BestSpeed)
	if _, err := zw.Write(data); err != nil { return nil, err }
	if err := zw.Close(); err != nil { return nil, err }
	return buf.Bytes(), nil
}

func Gunzip(data []byte) ([]byte, error) {
	zr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil { return nil, err }
	defer zr.Close()
	return io.ReadAll(zr)
}
