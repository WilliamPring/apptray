// Package config reads ~/.config/apptray/apptray.conf and answers the only
// question the rest of the program asks of it: what groups does this app
// belong to?
package config

import (
	"bufio"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"apptray/internal/desktop"
)

// Accepted values of the "tabs" setting: which edge the tab strip sits on.
const (
	TabsBottom = "bottom"
	TabsTop    = "top"
)

// FallbackGroup holds apps we have no better name for.
const FallbackGroup = "Other"

// friendly maps freedesktop category names to the group names we show.
// Keys are the spec's "main categories" plus the few extras that show up a lot.
var friendly = map[string]string{
	"Audio":            "Media",
	"AudioVideo":       "Media",
	"Video":            "Media",
	"Development":      "Development",
	"IDE":              "Development",
	"TextEditor":       "Development",
	"Education":        "Education",
	"Game":             "Games",
	"Graphics":         "Graphics",
	"Email":            "Internet",
	"Network":          "Internet",
	"WebBrowser":       "Internet",
	"Office":           "Office",
	"Science":          "Science",
	"System":           "System",
	"TerminalEmulator": "System",
	"Utility":          "Utilities",
}

// Config is the parsed contents of apptray.conf.
type Config struct {
	TabsAt string              // TabsBottom (default) or TabsTop
	Groups map[string][]string // group name -> desktop file IDs
}

// Default returns the config used when there is no file.
func Default() Config {
	return Config{TabsAt: TabsBottom, Groups: map[string][]string{}}
}

// DefaultPath returns $XDG_CONFIG_HOME/apptray/apptray.conf, falling back to
// ~/.config/apptray/apptray.conf. It returns "" if neither can be determined,
// which Load treats the same as a missing file.
func DefaultPath() string {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "apptray", "apptray.conf")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "apptray", "apptray.conf")
}

// Load reads path. A file that does not exist is not an error: a fresh install
// has no config, and that is the normal case, so we return Default() and nil.
// A file that exists but cannot be read is a real error and is reported.
func Load(path string) (Config, error) {
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Default(), nil
		}
		return Config{}, err // permissions, a directory in the way, ...
	}
	defer f.Close()

	c := Default()
	section := "" // which [section] we are inside

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") {
			section = strings.ToLower(strings.Trim(line, "[]"))
			continue
		}

		// Split on the FIRST "=": group names may not contain one, but let's
		// not make that a rule we enforce by accident.
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)

		switch section {
		case "settings":
			// Unknown keys, and unknown values for known keys, are ignored
			// rather than fatal: a typo should never stop apptray opening.
			if strings.EqualFold(key, "tabs") {
				if v := strings.ToLower(value); v == TabsTop || v == TabsBottom {
					c.TabsAt = v
				}
			}
		case "groups":
			if ids := splitIDs(value); key != "" && len(ids) > 0 {
				c.Groups[key] = append(c.Groups[key], ids...)
			}
		}
	}

	// Scan() stops on both end-of-file and read errors; Err() tells them apart.
	if err := scanner.Err(); err != nil {
		return Config{}, err
	}
	return c, nil
}

// splitIDs turns "slack; zoom; code" into ["slack" "zoom" "code"].
func splitIDs(value string) []string {
	var ids []string
	for _, id := range strings.Split(value, ";") {
		if id = strings.TrimSpace(id); id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}

// GroupsFor returns every group e belongs to, in display order.
//
// Three tiers, first match wins:
//  1. groups from [groups] that list e.ID  (an app may be in several)
//  2. the friendly name of the first category we recognise
//  3. FallbackGroup
//
// It never returns an empty slice, so callers never need a nil check.
func (c Config) GroupsFor(e desktop.Entry) []string {
	var groups []string
	for name, ids := range c.Groups {
		if slices.Contains(ids, e.ID) {
			groups = append(groups, name)
		}
	}
	if len(groups) > 0 {
		// Map iteration order is random in Go, so sort or the UI reshuffles.
		slices.Sort(groups)
		return groups
	}

	// Not every category is a "main" one: VS Code ships
	// "TextEditor;Development;IDE;", so take the first we actually know.
	for _, cat := range e.Categories {
		if name, ok := friendly[cat]; ok {
			return []string{name}
		}
	}
	return []string{FallbackGroup}
}
