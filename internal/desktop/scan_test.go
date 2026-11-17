package desktop

import "testing"

func TestScan(t *testing.T) {
	// "testdata/nope" doesn't exist: Scan must skip it, not fail.
	// testdata/hidden.desktop has NoDisplay=true, and testdata/settings.desktop
	// is in the Settings category: both must be filtered out.
	got, err := Scan([]string{"testdata", "testdata/nope"})
	if err != nil {
		t.Fatalf("Scan returned error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d entries, want 1: %+v", len(got), got)
	}
	if got[0].Name != "Sample App" {
		t.Errorf("got name %q, want %q", got[0].Name, "Sample App")
	}
}

func TestScanDeduplicatesByID(t *testing.T) {
	// Both dirs hold a "sample.desktop", so both produce the ID "sample".
	// Scan must return it once, keeping the copy from the dir listed first.
	got, err := Scan([]string{"testdata/override", "testdata"})
	if err != nil {
		t.Fatalf("Scan returned error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d entries, want 1: %+v", len(got), got)
	}
	if got[0].Name != "Sample App (user copy)" {
		t.Errorf("got name %q, want the override to win", got[0].Name)
	}
}
