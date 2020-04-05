package editor

import (
	"github.com/as/edit"
	"github.com/fmpwizard/go-quilljs-delta/delta"
)

type DeltaEditor struct {
	current    delta.Delta
	changes    delta.Delta
	start, end int64
}

func NewDeltaEditor(d delta.Delta) *DeltaEditor {
	return &DeltaEditor{
		current: d,
	}
}

func (e *DeltaEditor) Changes() delta.Delta {
	return e.changes
}

func (e *DeltaEditor) String() string {
	return string(e.Bytes())
}

func Run(command string, e *DeltaEditor) error {
	cmd, err := edit.Compile(command)
	if err != nil {
		return err
	}
	return cmd.Run(e)
}

func (e *DeltaEditor) Insert(p []byte, at int64) int {
	if len(p) == 0 {
		return 0
	}
	if at < 0 {
		return 0
	}
	if at > e.Len() {
		at = e.Len()
	}
	change := delta.New(nil).Retain(int(at), nil).Insert(string(p), nil)
	e.current = *e.current.Compose(*change)
	e.changes = *e.changes.Compose(*change)
	return len(p)
}

func (e *DeltaEditor) Delete(start, end int64) int {
	if end <= start || start < 0 {
		return 0
	}
	l := int(end - start)
	change := delta.New(nil).Retain(int(start), nil).Delete(l)
	e.current = *e.current.Compose(*change)
	e.changes = *e.changes.Compose(*change)
	return l
}

func (e *DeltaEditor) Select(start, end int64) {
	if start < 0 {
		return
	}
	if end > e.Len() {
		end = e.Len()
	}
	e.start = start
	e.end = end
}

func (e *DeltaEditor) Dot() (start, end int64) {
	start = e.start
	end = e.end
	return
}

func (e *DeltaEditor) Len() int64 {
	return int64(e.current.Length())
}

func (e *DeltaEditor) Bytes() []byte {
	result := make([]byte, 0)
	for _, op := range e.current.Ops {
		if op.Insert != nil {
			result = append(result, []byte(string(op.Insert))...)
		} else if op.InsertEmbed != nil {
			result = append(result, 0)
		}
	}
	return result
}

func (e *DeltaEditor) Close() error {
	return nil
}
