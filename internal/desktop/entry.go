package desktop

import (
	"bufio"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// Entry is the small part of a .desktop file we care about.
type Entry struct {
	ID 		  string
	Name      string
	Exec      string
	Comment   string
	Type      string // "Application", "Link" or "Directory"
	NoDisplay bool   // true = app exists but must not appear in menus
	Hidden    bool   // true = entry was "deleted" by the user/distro

	// Categories comes from "Categories=Network;WebBrowser;" in the file.
	Categories []string
}

// ShouldShow reports whether this entry belongs in a launcher.
func (e Entry) ShouldShow() bool {
	if e.Type != "Application" || e.NoDisplay || e.Hidden {
		return false
	}
	// Settings panels (Xfce Appearance, Display, Accessibility...) are not "apps".
	return !slices.Contains(e.Categories, "Settings")
}

// ParseFile reads one .desktop file and returns its [Desktop Entry] fields.
func ParseFile(path string) (Entry, error) {
	f, err := os.Open(path)
	if err != nil {
		return Entry{}, err // missing/unreadable file: report it, don't panic
	}
	defer f.Close() // runs when ParseFile returns, however it returns

	e := Entry{ID: strings.TrimSuffix(filepath.Base(path), ".desktop")}
	inEntry := false // are we currently inside the [Desktop Entry] section?

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip blank lines and comments.
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// A section header such as [Desktop Entry] or [Desktop Action new-window].
		if strings.HasPrefix(line, "[") {
			inEntry = line == "[Desktop Entry]"
			continue
		}

		if !inEntry {
			continue
		}

		// Split on the FIRST "=" only: Exec values may contain more of them.
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}

		// "Name[fr]" is a different key than "Name", so translations are skipped.
		switch strings.TrimSpace(key) {
		case "Name":
			e.Name = strings.TrimSpace(value)
		case "Exec":
			e.Exec = strings.TrimSpace(value)
		case "Comment":
			e.Comment = strings.TrimSpace(value)
		case "Type":
			e.Type = strings.TrimSpace(value)
		case "NoDisplay":
			e.NoDisplay = strings.TrimSpace(value) == "true"
		case "Hidden":
			e.Hidden = strings.TrimSpace(value) == "true"
		case "Categories":
			// "A;B;" splits into ["A" "B" ""], so drop the empty piece.
			for _, c := range strings.Split(value, ";") {
				if c = strings.TrimSpace(c); c != "" {
					e.Categories = append(e.Categories, c)
				}
			}
		}
	}

	// Scan() returns false on both "end of file" and "read error".
	// scanner.Err() tells them apart (nil means a normal end).
	return e, scanner.Err()
}
