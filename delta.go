package editor

import (
	"bytes"
	"io"

	"github.com/fmpwizard/go-quilljs-delta/delta"
)

type DeltaFile struct {
	current    delta.Delta
	changes    delta.Delta
	start, end int64
}

func NewDeltaFile(d delta.Delta) *DeltaFile {
	return &DeltaFile{
		current: d,
	}
}

func (e *DeltaFile) Changes() delta.Delta {
	return e.changes
}

func (e *DeltaFile) Commit() error {
	e.current = *e.current.Compose(e.changes)
	e.changes = *delta.New(nil)
	return nil
}

func (e *DeltaFile) Bytes(applyChange bool) []byte {
	c := e.current
	if applyChange {
		c = *c.Compose(e.changes)
	}
	result := make([]byte, 0)
	for _, op := range c.Ops {
		if op.Insert != nil {
			result = append(result, []byte(string(op.Insert))...)
		} else if op.InsertEmbed != nil {
			result = append(result, 0)
		}
	}
	return result
}

func (e *DeltaFile) String(applyChange bool) string {
	return string(e.Bytes(applyChange))
}

func (e *DeltaFile) Insert(p []byte, at int64) int64 {
	at = int64(e.changes.TransformPosition(int(at), true))
	if at < 0 || len(p) == 0 {
		return 0
	}
	if at > e.appliedLen() {
		at = e.appliedLen()
	}
	change := delta.New(nil).Retain(int(at), nil).Insert(string(p), nil)
	e.changes = *e.changes.Compose(*change)
	e.start = at
	e.end = at + int64(len(p))
	return int64(len(p))
}

func (e *DeltaFile) Delete(start, end int64) int64 {
	start = int64(e.changes.TransformPosition(int(start), true))
	end = int64(e.changes.TransformPosition(int(end), true))
	if end > e.appliedLen() {
		end = e.appliedLen()
	}
	if start < 0 || end <= start {
		return 0
	}
	l := int(end - start)
	change := delta.New(nil).Retain(int(start), nil).Delete(l)
	e.changes = *e.changes.Compose(*change)
	e.start = start
	e.end = start
	return int64(l)
}

func (e *DeltaFile) Select(start, end int64) {
	e.start = start
	e.end = end
}

func (e *DeltaFile) Dot() (start, end int64) {
	start = e.start
	end = e.end
	return
}

func (e *DeltaFile) Len() int64 {
	return int64(e.current.Length())
}

func (e *DeltaFile) appliedLen() int64 {
	return int64(e.current.Compose(e.changes).Length())
}

func (e *DeltaFile) Reader(start, end int64) io.ReadSeeker {
	if end < start || start > e.Len() {
		return nil
	}
	if end > e.Len() {
		end = e.Len()
	}
	return bytes.NewReader(e.Bytes(false)[start:end])
}
