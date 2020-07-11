package editor

import (
	"bytes"
	"os"
	"testing"
)

func TestFile(t *testing.T) {
	file, err := os.Open("testfile.txt")
	if err != nil {
		t.Fatal(err)
	}
	e := NewGoFile(file)
	buf := bytes.NewBuffer(nil)
	ctx := Context{
		File:    e,
		Printer: buf,
	}
	cmd, err := Compile("2p")
	if err != nil {
		t.Fatal(err)
	}
	err = cmd.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}
	printerContent := buf.String()
	expectedContent := "Emacs sam\n"
	if printerContent != expectedContent {
		t.Fatalf("Invalid printer value! Expected: %s, actual: %s", expectedContent, printerContent)
	}
}

func TestDot(t *testing.T) {
	file, err := os.Open("testfile.txt")
	if err != nil {
		t.Fatal(err)
	}
	e := NewGoFile(file)
	ctx := Context{
		File: e,
	}
	cmd, err := Compile("3=")
	if err != nil {
		t.Fatal(err)
	}
	err = cmd.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}
	q0, q1 := e.Dot()
	if q0 != 14 || q1 != 23 {
		t.Fatalf("Invalid dot! Expected: (14, 23), actual: (%d, %d)", q0, q1)
	}
}
