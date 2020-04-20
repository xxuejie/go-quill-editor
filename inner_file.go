package editor

import (
	"io"

	"github.com/fmpwizard/go-quilljs-delta/delta"
)

type innerFile struct {
	file        File
	changes     delta.Delta
	originalLen int64
	appliedLen  int64
}

func newInnerFile(file File) (*innerFile, error) {
	l, err := file.Len()
	if err != nil {
		return nil, err
	}
	return &innerFile{
		file:        file,
		originalLen: l,
		appliedLen:  l,
	}, nil
}

func (f *innerFile) updateAppliedLen(d *delta.Delta) {
	f.appliedLen = int64(d.TransformPosition(int(f.appliedLen), false))
}

func (f *innerFile) Commit() error {
	err := f.file.Compose(f.changes)
	if err != nil {
		return err
	}
	f.changes = *delta.New(nil)
	f.originalLen, err = f.file.Len()
	if err != nil {
		return nil
	}
	f.appliedLen = f.originalLen
	return nil
}

func (f *innerFile) Insert(p []byte, at int64) int64 {
	at = int64(f.changes.TransformPosition(int(at), true))
	if at < 0 || len(p) == 0 {
		return 0
	}
	if at > f.appliedLen {
		at = f.appliedLen
	}
	change := delta.New(nil).Retain(int(at), nil).Insert(string(p), nil)
	f.updateAppliedLen(change)
	f.changes = *f.changes.Compose(*change)
	f.Select(at, at+int64(len(p)))
	return int64(len(p))
}

func (f *innerFile) Delete(start, end int64) int64 {
	start = int64(f.changes.TransformPosition(int(start), true))
	end = int64(f.changes.TransformPosition(int(end), true))
	if end > f.appliedLen {
		end = f.appliedLen
	}
	if start < 0 || end <= start {
		return 0
	}
	l := int(end - start)
	change := delta.New(nil).Retain(int(start), nil).Delete(l)
	f.updateAppliedLen(change)
	f.changes = *f.changes.Compose(*change)
	f.Select(start, start)
	return int64(l)
}

func (f *innerFile) Select(start, end int64) {
	f.file.Select(start, end)
}

func (f *innerFile) Dot() (int64, int64) {
	return f.file.Dot()
}

func (f *innerFile) Len() int64 {
	return f.originalLen
}

func (f *innerFile) Reader(start, end int64) io.ReadSeeker {
	if end < start || start > f.Len() {
		return nil
	}
	if end > f.Len() {
		end = f.Len()
	}
	return f.file.Reader(start, end)
}
