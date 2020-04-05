package editor

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/fmpwizard/go-quilljs-delta/delta"
)

func debugDeltaString(t *testing.T, d delta.Delta) string {
	data, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestNormalCommand(t *testing.T) {
	content := *delta.New(nil).Insert("Code Emacs Vim Sam ed", nil)
	e := NewDeltaEditor(content)
	err := Run("/Emacs/a/ is not so great/", e)
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
	e := NewDeltaEditor(content)
	err := Run("/Emacs/a/ is not so great/", e)
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
	e := NewDeltaEditor(content)
	err := Run(",x/m /d", e)
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
