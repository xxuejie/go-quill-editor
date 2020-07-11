package editor

import (
	"fmt"
	"io"
	"os"

	"github.com/fmpwizard/go-quilljs-delta/delta"
)

type GoFileFile struct {
	file       *os.File
	start, end int64
}

func NewGoFile(file *os.File) *GoFileFile {
	return &GoFileFile{
		file: file,
	}
}

func (f *GoFileFile) Select(start, end int64) {
	f.start = start
	f.end = end
}

func (f *GoFileFile) Dot() (start, end int64) {
	start = f.start
	end = f.end
	return
}

// For now, we would only use  GoFileFile in a readonly fashion, later we
// might visit again to see if we can provide proper Compose implementation.
func (f *GoFileFile) Compose(d delta.Delta) error {
	if len(d.Ops) > 0 {
		return fmt.Errorf("GoFileFile is read only for now!")
	}
	return nil
}

func (f *GoFileFile) Len() (int64, error) {
	stat, err := f.file.Stat()
	if err != nil {
		return -1, err
	}
	return stat.Size(), nil
}

type goFileReader struct {
	file   *GoFileFile
	offset int64
	start  int64
	end    int64
}

func (f *GoFileFile) Reader(start, end int64) io.ReadSeeker {
	return &goFileReader{
		file:   f,
		offset: 0,
		start:  start,
		end:    end,
	}
}

func (r *goFileReader) Len() int64 {
	return r.end - r.start
}

func (r *goFileReader) Read(p []byte) (int, error) {
	offset := r.offset + r.start
	_, err := r.file.file.Seek(offset, io.SeekStart)
	if err != nil {
		return -1, err
	}
	remaining := r.end - offset
	originalLen := int64(len(p))
	if remaining < originalLen {
		p = p[0:remaining]
	}
	n, err := r.file.file.Read(p)
	if err == nil && remaining < originalLen {
		err = io.EOF
	}
	if n > 0 {
		r.offset += int64(n)
	}
	return n, err
}

func (r *goFileReader) Seek(offset int64, whence int) (int64, error) {
	var newOffset int64
	switch whence {
	case io.SeekStart:
		newOffset = offset
	case io.SeekCurrent:
		newOffset = r.offset + offset
	case io.SeekEnd:
		newOffset = r.Len() + offset
	}
	if newOffset < 0 {
		return -1, io.EOF
	}
	r.offset = newOffset
	return r.offset, nil
}
