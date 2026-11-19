// Package search decides which apps a query matches, and in what order.
package search

import (
	"slices"
	"strings"

	"apptray/internal/desktop"
)

// How a match is scored. The numbers only matter relative to each other:
// they exist to push the name you meant to the top of the list.
const (
	bonusWordStart   = 10 // "fox" hitting the F of "Firefox" or the W of "GNU Wget"
	bonusConsecutive = 5  // the letter follows the previous match directly
	penaltyGap       = 1  // charged per character skipped to find this letter
)

// Score reports whether query's characters appear in s in order — a fuzzy
// match, so "lof" finds "LibreOffice Writer" — and how good the match is.
// Higher is better. An empty query matches everything with a score of 0.
//
// The search is greedy: each letter takes the first position it can. That is
// not always the highest-scoring arrangement (a full fuzzy finder solves this
// with dynamic programming), but it is one pass, easy to reason about, and
// right for the handful of apps a menu holds.
func Score(s, query string) (int, bool) {
	q := []rune(strings.ToLower(strings.TrimSpace(query)))
	if len(q) == 0 {
		return 0, true
	}
	hay := []rune(strings.ToLower(s))

	score := 0
	at := 0    // where in hay the next letter may start looking
	prev := -2 // where the previous letter matched; -2 so it is never "adjacent"

	for _, want := range q {
		pos := -1
		for i := at; i < len(hay); i++ {
			if hay[i] == want {
				pos = i
				break
			}
		}
		if pos < 0 {
			return 0, false // a letter the name does not have: not a match
		}

		switch {
		case pos == prev+1:
			score += bonusConsecutive
		case pos == 0 || isBoundary(hay[pos-1]):
			score += bonusWordStart
		}
		score -= (pos - at) * penaltyGap

		prev, at = pos, pos+1
	}
	return score, true
}

// Match is Score without the score, for the places that only need yes or no.
func Match(s, query string) bool {
	_, ok := Score(s, query)
	return ok
}

// isBoundary reports whether r ends a word, so the character after it counts
// as the start of the next one.
func isBoundary(r rune) bool {
	switch r {
	case ' ', '-', '_', '.', '/', ':':
		return true
	}
	return false
}

// Filter returns the entries matching query, best match first.
// An empty query returns everything, untouched and in its original order.
func Filter(entries []desktop.Entry, query string) []desktop.Entry {
	if strings.TrimSpace(query) == "" {
		return entries
	}

	type hit struct {
		entry desktop.Entry
		score int
	}

	var hits []hit
	for _, e := range entries {
		if score, ok := Score(e.Name, query); ok {
			hits = append(hits, hit{e, score})
		}
	}

	// Stable, so equal scores keep the alphabetical order desktop.Scan gave
	// us. An unstable sort would let the list reshuffle between keystrokes.
	slices.SortStableFunc(hits, func(a, b hit) int { return b.score - a.score })

	out := make([]desktop.Entry, len(hits))
	for i, h := range hits {
		out[i] = h.entry
	}
	return out
}
