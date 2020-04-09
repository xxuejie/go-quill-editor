package editor

import (
	"fmt"
	"io"
	"regexp"
)

type File interface {
	Insert(p []byte, at int64) (n int64)
	Delete(q0, q1 int64) (n int64)
	Select(q0, q1 int64)
	Dot() (q0, q1 int64)
	Len() int64
	Reader(q0, q1 int64) io.ReadSeeker
	Commit() error
}

type Context struct {
	File    File
	Printer io.Writer
	Commit  bool
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
	err := cmdExec(c, context)
	if err != nil {
		return err
	}
	if context.Commit {
		err = context.File.Commit()
		if err != nil {
			return err
		}
	}
	return nil
}
