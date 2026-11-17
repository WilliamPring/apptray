package desktop

import (
	"reflect"
	"testing"
)

func TestParseFile(t *testing.T) {
	got, err := ParseFile("testdata/sample.desktop")
	if err != nil {
		t.Fatalf("ParseFile returned error: %v", err)
	}

	want := Entry{
		ID:      "sample",     // basename of the path, minus ".desktop"
		Name:    "Sample App", // not "Application Exemple", not "New Window"
		Exec:    "sample --flag %U",
		Comment: "Just a sample",
		Type:    "Application",

		Categories: []string{"Utility", "Network"},
	}
	// `got != want` no longer compiles: Entry now holds a slice, and slices
	// can't be compared with ==. reflect.DeepEqual compares contents instead.
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestParseFileMissing(t *testing.T) {
	_, err := ParseFile("testdata/does-not-exist.desktop")
	if err == nil {
		t.Fatal("expected an error for a missing file, got nil")
	}
}
