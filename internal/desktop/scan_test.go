package desktop

import "testing"

func TestScan(t *testing.T) {
	// "testdata/nope" doesn't exist: Scan must skip it, not fail.
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
