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
		{"case insensitive", "FIRE", 1},
		{"substring in the middle", "office", 1},
		{"no match", "zzz", 0},
		{"surrounding spaces ignored", "  brave ", 1},
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
