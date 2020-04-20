package editor

import (
	"fmt"
	"io"
	"regexp"

	"github.com/fmpwizard/go-quilljs-delta/delta"
)

type File interface {
	Select(q0, q1 int64)
	Dot() (q0, q1 int64)
	Len() (int64, error)
	Reader(start, end int64) io.ReadSeeker
	Compose(d delta.Delta) error
}

type Context struct {
	File    File
	Printer io.Writer
}

func Compile(cmd string) (Cmd, error) {
	cmd = regexp.MustCompile("\n*$").ReplaceAllString(cmd, "\n")
	c, err := innerParseCmd(newCmdScanner(cmd), 0)
	if err != nil {
		return Cmd{}, err
	}
	if c == nil {
		return Cmd{}, fmt.Errorf("Command is empty!")
	}
	return *c, nil
}

func (c Cmd) Run(context Context) error {
	innerContext, err := newInnerContext(context)
	if err != nil {
		return err
	}
	err = cmdExec(c, innerContext)
	if err != nil {
		return err
	}
	return innerContext.File.Commit()
}
