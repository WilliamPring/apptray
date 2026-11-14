package search

import (
	"strings"

	"apptray/internal/desktop"
)

// Filter returns the entries whose name contains query, ignoring case.
// An empty query returns everything.
func Filter(entries []desktop.Entry, query string) []desktop.Entry {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return entries
	}

	var out []desktop.Entry
	for _, e := range entries {
		if strings.Contains(strings.ToLower(e.Name), q) {
			out = append(out, e)
		}
	}
	return out
}
