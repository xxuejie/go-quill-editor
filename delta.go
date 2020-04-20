package editor

import (
	"bytes"
	"io"

	"github.com/fmpwizard/go-quilljs-delta/delta"
)

type DeltaFile struct {
	delta.Delta
	start, end int64
}

func NewDeltaFile(d delta.Delta) *DeltaFile {
	return &DeltaFile{
		Delta: d,
	}
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

func (e *DeltaFile) Len() (int64, error) {
	return int64(e.Length()), nil
}

func (e *DeltaFile) Reader(start, end int64) io.ReadSeeker {
	l, err := e.Len()
	if err != nil {
		return nil
	}
	if end < start || start > l {
		return nil
	}
	if end > l {
		end = l
	}
	return bytes.NewReader(e.Bytes()[start:end])
}

func (e *DeltaFile) Compose(d delta.Delta) error {
	e.Delta = *e.Delta.Compose(d)
	return nil
}

func (e *DeltaFile) Bytes() []byte {
	result := make([]byte, 0)
	for _, op := range e.Ops {
		if op.Insert != nil {
			result = append(result, []byte(string(op.Insert))...)
		} else if op.InsertEmbed != nil {
			result = append(result, 0)
		}
	}
	return result
}
