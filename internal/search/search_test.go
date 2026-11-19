package search

import (
	"testing"

	"apptray/internal/desktop"
)

func TestFilter(t *testing.T) {
	entries := []desktop.Entry{
		{Name: "Firefox"},
		{Name: "Brave"},
		{Name: "LibreOffice Writer"},
	}

	// Table-driven test: one row per case, same checking code for all.
	tests := []struct {
		name  string
		query string
		want  int
	}{
		{"empty query returns all", "", 3},
		{"substring in the middle", "office", 1},
		{"no match", "zzz", 0},
		{"surrounding spaces ignored", "  brave ", 1},

		// Fuzzy: the letters only have to appear in order, so a query can
		// skip whole words. This is the point of the mode, and also the cost
		// of it — see the two matches for "FIRE" below.
		{"initials across words", "lw", 1},
		{"case insensitive, and fuzzy enough to catch a second name", "FIRE", 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Filter(entries, tt.query)
			if len(got) != tt.want {
				t.Errorf("Filter(%q) returned %d entries, want %d", tt.query, len(got), tt.want)
			}
		})
	}
}

func TestFilterRanks(t *testing.T) {
	entries := []desktop.Entry{
		{Name: "Firefox"},
		{Name: "Brave"},
		{Name: "LibreOffice Writer"},
	}

	// Both names contain f-i-r-e in order, so matching alone is not enough:
	// the one that spells it out has to come first.
	got := Filter(entries, "fire")
	if len(got) == 0 {
		t.Fatal("Filter found nothing")
	}
	if got[0].Name != "Firefox" {
		t.Errorf("best match is %q, want %q", got[0].Name, "Firefox")
	}
}

func TestScore(t *testing.T) {
	tests := []struct {
		name   string
		in     string
		query  string
		wantOK bool
	}{
		{"exact prefix", "Firefox", "fire", true},
		{"letters in order with gaps", "LibreOffice Writer", "lof", true},
		{"start of each word", "GNU Image Manipulation Program", "gimp", true},
		{"wrong order does not match", "Firefox", "erif", false},
		{"a letter that is not there", "Brave", "bz", false},
		{"empty query matches anything", "Brave", "   ", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, ok := Score(tt.in, tt.query); ok != tt.wantOK {
				t.Errorf("Score(%q, %q) ok = %v, want %v", tt.in, tt.query, ok, tt.wantOK)
			}
		})
	}
}

func TestScorePrefersWordStarts(t *testing.T) {
	// "gimp" spelled out by the initials of four words should beat the same
	// letters buried inside one word, or the ranking is useless.
	acronym, ok := Score("GNU Image Manipulation Program", "gimp")
	if !ok {
		t.Fatal("the acronym did not match")
	}
	buried, ok := Score("Imaging Mop", "gimp")
	if !ok {
		t.Skip("second name did not match; nothing to compare")
	}
	if acronym <= buried {
		t.Errorf("acronym scored %d, buried scored %d: want the acronym higher", acronym, buried)
	}
}
