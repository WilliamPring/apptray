package desktop

import "testing"

func TestParseFile(t *testing.T) {
	got, err := ParseFile("testdata/sample.desktop")
	if err != nil {
		t.Fatalf("ParseFile returned error: %v", err)
	}

	want := Entry{
		Name:    "Sample App", // not "Application Exemple", not "New Window"
		Exec:    "sample --flag %U",
		Comment: "Just a sample",
	}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestParseFileMissing(t *testing.T) {
	_, err := ParseFile("testdata/does-not-exist.desktop")
	if err == nil {
		t.Fatal("expected an error for a missing file, got nil")
	}
}
