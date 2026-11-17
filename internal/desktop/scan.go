package desktop

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// DefaultDirs returns the folders where Linux keeps .desktop files.
//
// The per-user folder comes first on purpose: Scan keeps the first entry it sees
// for a given ID, and a user's copy of an app is meant to override the system one.
func DefaultDirs() []string {
	var dirs []string

	// Per-user apps. If we can't find the home dir, just skip this one.
	if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, filepath.Join(home, ".local", "share", "applications"))
	}
	return append(dirs, "/usr/share/applications")
}

// Scan parses every .desktop file in dirs and returns the entries sorted by name.
// When the same ID appears in several dirs, the first one wins.
func Scan(dirs []string) ([]Entry, error) {
	var entries []Entry
	seen := make(map[string]bool)

	for _, dir := range dirs {
		files, err := os.ReadDir(dir)
		if err != nil {
			// A folder that doesn't exist (e.g. no ~/.local/share/applications) is fine.
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return nil, err // anything else (permissions...) is a real problem
		}

		for _, file := range files {
			if file.IsDir() || !strings.HasSuffix(file.Name(), ".desktop") {
				continue
			}

			e, err := ParseFile(filepath.Join(dir, file.Name()))
			if err != nil || e.Name == "" || !e.ShouldShow() {
				continue // broken, nameless or not meant for menus: skip it
			}
			if seen[e.ID] {
				continue // already have this app from an earlier (higher priority) dir
			}
			seen[e.ID] = true
			entries = append(entries, e)
		}
	}

	sort.Slice(entries, func(i, j int) bool {
		return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
	})
	return entries, nil
}