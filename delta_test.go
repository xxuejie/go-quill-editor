package editor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"testing"

	"github.com/fmpwizard/go-quilljs-delta/delta"
)

func run(command string, f File) error {
	cmd, err := Compile(command)
	if err != nil {
		return err
	}
	err = cmd.Run(Context{
		File:    f,
		Printer: os.Stderr,
	})
	if err != nil {
		return err
	}
	return nil
}

func debugDeltaString(t *testing.T, d delta.Delta) string {
	data, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestDelta(t *testing.T) {
	content := *delta.New(nil).Insert("Code Emacs Vim Sam ed", nil)
	e := NewDeltaFile(content)
	err := run("/Emacs/a/ is not so great/", e)
	if err != nil {
		t.Fatal(err)
	}
	expectedContent := "Code Emacs is not so great Vim Sam ed"
	actualContent := string(e.Bytes())
	if actualContent != expectedContent {
		t.Fatalf("Invalid result: expected: \"%s\", actual: \"%s\"", expectedContent, actualContent)
	}
}

type testDelta struct {
	content    delta.Delta
	changes    delta.Delta
	start, end int64
}

func newTestDelta(d delta.Delta) *testDelta {
	return &testDelta{
		content: d,
	}
}

func (t *testDelta) Changes() delta.Delta {
	return t.changes
}

func (t *testDelta) Select(start, end int64) {
	t.start = start
	t.end = end
}

func (t *testDelta) Dot() (start, end int64) {
	start = t.start
	end = t.end
	return
}

func (t *testDelta) Len() (int64, error) {
	return (&DeltaFile{
		Delta: *t.content.Compose(t.changes),
	}).Len()
}

func (t *testDelta) Reader(start, end int64) io.ReadSeeker {
	return (&DeltaFile{
		Delta: *t.content.Compose(t.changes),
	}).Reader(start, end)
}

func (t *testDelta) Compose(d delta.Delta) error {
	t.changes = *t.changes.Compose(d)
	return nil
}

func (t *testDelta) String() string {
	return string((&DeltaFile{
		Delta: *t.content.Compose(t.changes),
	}).Bytes())
}

func TestNormalCommand(t *testing.T) {
	content := *delta.New(nil).Insert("Code Emacs Vim Sam ed", nil)
	e := newTestDelta(content)
	err := run("/Emacs/a/ is not so great/", e)
	if err != nil {
		t.Fatal(err)
	}
	expectedContent := "Code Emacs is not so great Vim Sam ed"
	actualContent := e.String()
	if actualContent != expectedContent {
		t.Fatalf("Invalid result: expected: \"%s\", actual: \"%s\"", expectedContent, actualContent)
	}
	expectedChange := *delta.New(nil).Retain(10, nil).Insert(" is not so great", nil)
	if !reflect.DeepEqual(e.Changes(), expectedChange) {
		t.Fatalf("Invalid change, expected: %s, actual: %s",
			debugDeltaString(t, expectedChange), debugDeltaString(t, e.Changes()))
	}
}

func TestContentWithEmbed(t *testing.T) {
	content := *delta.New(nil).Insert("Code Em", nil).
		InsertEmbed(delta.Embed{
			Key:   "image",
			Value: "image-uri",
		}, nil).
		Insert("acs Emacs Vim Sam ed", nil)
	e := newTestDelta(content)
	err := run("/Emacs/a/ is not so great/", e)
	if err != nil {
		t.Fatal(err)
	}
	expectedContent := "Code Em\x00acs Emacs is not so great Vim Sam ed"
	actualContent := e.String()
	if actualContent != expectedContent {
		t.Fatalf("Invalid result: expected: \"%s\", actual: \"%s\"", expectedContent, actualContent)
	}
	expectedChange := *delta.New(nil).Retain(17, nil).Insert(" is not so great", nil)
	if !reflect.DeepEqual(e.Changes(), expectedChange) {
		t.Fatalf("Invalid change, expected: %s, actual: %s",
			debugDeltaString(t, expectedChange), debugDeltaString(t, e.Changes()))
	}
}

func TestDeleteCommand(t *testing.T) {
	content := *delta.New(nil).Insert("Code Emacs Vim Sam ed", nil)
	e := newTestDelta(content)
	err := run(",x/m /d", e)
	if err != nil {
		t.Fatal(err)
	}
	expectedContent := "Code Emacs ViSaed"
	actualContent := e.String()
	if actualContent != expectedContent {
		t.Fatalf("Invalid result: expected: \"%s\", actual: \"%s\"", expectedContent, actualContent)
	}
	expectedChange := *delta.New(nil).Retain(13, nil).Delete(2).Retain(2, nil).Delete(2)
	if !reflect.DeepEqual(e.Changes(), expectedChange) {
		t.Fatalf("Invalid change, expected: %s, actual: %s",
			debugDeltaString(t, expectedChange), debugDeltaString(t, e.Changes()))
	}
}

func TestDeleteFromMultiLineCommand(t *testing.T) {
	content := *delta.New(nil).Insert("1 45 1\n2 48 21\n3 45 1\n4 48 43\n5 45 1\n6 48 20\n", nil)
	e := newTestDelta(content)
	err := run(",x/^(5|6)/+-d", e)
	if err != nil {
		t.Fatal(err)
	}
	expectedContent := "1 45 1\n2 48 21\n3 45 1\n4 48 43\n"
	actualContent := e.String()
	if actualContent != expectedContent {
		t.Fatalf("Invalid result: expected: \"%s\", actual: \"%s\"", expectedContent, actualContent)
	}
}

type testCaseRun struct {
	command string
	result  string
	print   string
}

type testCase struct {
	source string
	runs   []testCaseRun
}

const DefaultSource = `This manual is organized in a rather haphazard manner. The first
several sections were written hastily in an attempt to provide a
general introduction to the commands in Emacs and to try to show
the method in the madness that is the Emacs command structure.
`

var testCases = []testCase{
	{
		source: "",
		runs: []testCaseRun{
			{
				command: fmt.Sprintf("a\n%s.", DefaultSource),
				result:  DefaultSource,
				print:   "",
			},
			{
				command: "p",
				result:  DefaultSource,
				print:   DefaultSource,
			},
		},
	},
	{
		source: DefaultSource,
		runs: []testCaseRun{
			{
				command: "2c\nchanged\n.",
				result: `This manual is organized in a rather haphazard manner. The first
changed
general introduction to the commands in Emacs and to try to show
the method in the madness that is the Emacs command structure.
`,
				print: "",
			},
			{
				command: "p",
				result: `This manual is organized in a rather haphazard manner. The first
changed
general introduction to the commands in Emacs and to try to show
the method in the madness that is the Emacs command structure.
`,
				print: "changed\n",
			},
		},
	},
	{
		source: DefaultSource,
		runs: []testCaseRun{
			{
				command: "#1,g/manual/p",
				result:  DefaultSource,
				print: `his manual is organized in a rather haphazard manner. The first
several sections were written hastily in an attempt to provide a
general introduction to the commands in Emacs and to try to show
the method in the madness that is the Emacs command structure.
`,
			},
		},
	},
	{
		source: DefaultSource,
		runs: []testCaseRun{
			{
				command: "#2,v/manual/p",
				result:  DefaultSource,
				print:   "",
			},
		},
	},
	{
		source: DefaultSource,
		runs: []testCaseRun{
			{
				command: "$i/thisisend/",
				result: `This manual is organized in a rather haphazard manner. The first
several sections were written hastily in an attempt to provide a
general introduction to the commands in Emacs and to try to show
the method in the madness that is the Emacs command structure.
thisisend`,
				print: "",
			},
		},
	},
	{
		source: DefaultSource,
		runs: []testCaseRun{
			{
				command: "/manual/m/haphazard/",
				result: `This  is organized in a rather haphazardmanual manner. The first
several sections were written hastily in an attempt to provide a
general introduction to the commands in Emacs and to try to show
the method in the madness that is the Emacs command structure.
`,
				print: "",
			},
			{
				command: "p",
				result: `This  is organized in a rather haphazardmanual manner. The first
several sections were written hastily in an attempt to provide a
general introduction to the commands in Emacs and to try to show
the method in the madness that is the Emacs command structure.
`,
				print: "manual",
			},
		},
	},
	{
		source: DefaultSource,
		runs: []testCaseRun{
			{
				command: "/manual/t/haphazard/",
				result: `This manual is organized in a rather haphazardmanual manner. The first
several sections were written hastily in an attempt to provide a
general introduction to the commands in Emacs and to try to show
the method in the madness that is the Emacs command structure.
`,
				print: "",
			},
		},
	},
	{
		source: DefaultSource,
		runs: []testCaseRun{
			{
				command: "1,$s/haphazard/thoughtless/",
				result: `This manual is organized in a rather thoughtless manner. The first
several sections were written hastily in an attempt to provide a
general introduction to the commands in Emacs and to try to show
the method in the madness that is the Emacs command structure.
`,
				print: "",
			},
		},
	},
	{
		source: DefaultSource,
		runs: []testCaseRun{
			{
				command: "1,$s/haphazard/thoughtless&/",
				result: `This manual is organized in a rather thoughtlesshaphazard manner. The first
several sections were written hastily in an attempt to provide a
general introduction to the commands in Emacs and to try to show
the method in the madness that is the Emacs command structure.
`,
				print: "",
			},
		},
	},
	{
		source: DefaultSource,
		runs: []testCaseRun{
			{
				command: "1,$s/hapha(zard)/\\1/",
				result: `This manual is organized in a rather zard manner. The first
several sections were written hastily in an attempt to provide a
general introduction to the commands in Emacs and to try to show
the method in the madness that is the Emacs command structure.
`,
				print: "",
			},
		},
	},
	{
		source: DefaultSource,
		runs: []testCaseRun{
			{
				command: "1,$s/Emacs/vi/g",
				result: `This manual is organized in a rather haphazard manner. The first
several sections were written hastily in an attempt to provide a
general introduction to the commands in vi and to try to show
the method in the madness that is the vi command structure.
`,
				print: "",
			},
		},
	},
	{
		source: DefaultSource,
		runs: []testCaseRun{
			{
				command: "1,$s2/Emacs/vi/",
				result: `This manual is organized in a rather haphazard manner. The first
several sections were written hastily in an attempt to provide a
general introduction to the commands in Emacs and to try to show
the method in the madness that is the vi command structure.
`,
				print: "",
			},
		},
	},
	{
		source: DefaultSource,
		runs: []testCaseRun{
			{
				command: "2=",
				result:  DefaultSource,
				print:   "2\n",
			},
		},
	},
	{
		source: DefaultSource,
		runs: []testCaseRun{
			{
				command: "2=#",
				result:  DefaultSource,
				print:   "#65,#130\n",
			},
		},
	},
	{
		source: DefaultSource,
		runs: []testCaseRun{
			{
				command: "#5,#100=+",
				result:  DefaultSource,
				print:   "1+#5,2+#35\n",
			},
		},
	},
	{
		source: DefaultSource,
		runs: []testCaseRun{
			{
				command: `,x/Emacs|vi/{
g/Emacs/ c/vi/
g/vi/ c/Emacs/
}`,
				result: `This manual is organized in a rather haphazard manner. The first
several sections were written hastily in an attempt to proEmacsde a
general introduction to the commands in vi and to try to show
the method in the madness that is the vi command structure.
`,
				print: "",
			},
		},
	},
}

func TestMultipleCases(t *testing.T) {
	for i, c := range testCases {
		f := newTestDelta(*delta.New(nil).Insert(c.source, nil))
		ctx := Context{
			File: f,
		}
		for j, run := range c.runs {
			cmd, err := Compile(run.command)
			if err != nil {
				t.Fatalf("Error compiling command %s at (%d, %d): %v", run.command, i, j, err)
			}
			buf := bytes.NewBuffer(nil)
			ctx.Printer = buf
			err = cmd.Run(ctx)
			if err != nil {
				t.Fatalf("Error running command %s at (%d, %d): %v", run.command, i, j, err)
			}
			actualContent := f.String()
			if actualContent != run.result {
				t.Fatalf("Inconsistent result for command %s at (%d, %d)\nExpected: \"%s\"\nActual: \"%s\"\n", run.command, i, j, run.result, actualContent)
			}
			printerContent := buf.String()
			if printerContent != run.print {
				t.Fatalf("Inconsistent print data for command %s at (%d, %d)\nExpected: \"%s\"\nActual: \"%s\"\n", run.command, i, j, run.print, printerContent)
			}
		}
	}
}
