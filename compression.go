package q

import (
	"io"
	"sync"

	"github.com/klauspost/compress/gzip"
)

var gzipWriterPool sync.Pool // used for several methods, usually inside context

// AcquireGzip prepares a gzip writer and returns it
//
// see ReleaseGzip
func AcquireGzip(w io.Writer) *gzip.Writer {
	v := gzipWriterPool.Get()
	if v == nil {
		gzipWriter, err := gzip.NewWriterLevel(w, gzip.DefaultCompression)
		if err != nil {
			return nil
		}
		return gzipWriter
	}
	gzipWriter := v.(*gzip.Writer)
	gzipWriter.Reset(w)
	return gzipWriter
}

// ReleaseGzip called when flush/close and put the gzip writer back to the pool
//
// see AcquireGzip
func ReleaseGzip(gzipWriter *gzip.Writer) {
	gzipWriter.Close()
	gzipWriterPool.Put(gzipWriter)
}

// WriteGzip accepts an io.Writer to write on it and the contents to write
func WriteGzip(w io.Writer, b []byte) (int, error) {
	gzipWriter := AcquireGzip(w)
	n, err := gzipWriter.Write(b)
	ReleaseGzip(gzipWriter)
	return n, err
}
