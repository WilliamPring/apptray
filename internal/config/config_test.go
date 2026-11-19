package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"apptray/internal/desktop"
)

// writeConf drops a config file in a directory that the test framework deletes
// for us afterwards. Each case gets its own file, so the input sits right next
// to the assertion instead of in a shared testdata/ fixture.
func writeConf(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "apptray.conf")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("writing the test config: %v", err)
	}
	return path
}

func TestLoadMissingFile(t *testing.T) {
	// A fresh install has no config. That is the normal case, not an error.
	got, err := Load(filepath.Join(t.TempDir(), "does-not-exist.conf"))
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if !reflect.DeepEqual(got, Default()) {
		t.Errorf("got %+v, want %+v", got, Default())
	}
}

func TestLoad(t *testing.T) {
	path := writeConf(t, `
# a comment, and the blank line above it

[Settings]
tabs = top

[groups]
Work = slack; zoom; code
Media = vlc;
`)

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	want := Config{
		TabsAt: TabsTop, // "[Settings]" is matched case-insensitively
		Groups: map[string][]string{
			"Work":  {"slack", "zoom", "code"},
			"Media": {"vlc"}, // the trailing ";" must not produce an empty ID
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestLoadIgnoresBadSettings(t *testing.T) {
	// A typo in the config should never stop apptray from opening.
	path := writeConf(t, `
[settings]
theme = dracula
tabs = sideways
no-equals-sign-here
`)

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if got.TabsAt != TabsBottom {
		t.Errorf("TabsAt = %q, want the default %q", got.TabsAt, TabsBottom)
	}
}

func TestLoadRepeatedGroupKeyAppends(t *testing.T) {
	path := writeConf(t, "[groups]\nWork = slack; zoom\nWork = code\n")

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	want := []string{"slack", "zoom", "code"}
	if !reflect.DeepEqual(got.Groups["Work"], want) {
		t.Errorf("Groups[\"Work\"] = %v, want %v", got.Groups["Work"], want)
	}
}

func TestGroupsFor(t *testing.T) {
	c := Config{Groups: map[string][]string{
		"Work":  {"slack", "code"},
		"Admin": {"slack"},
	}}

	tests := []struct {
		name  string
		entry desktop.Entry
		want  []string
	}{
		{
			"an ID in two groups is in both, sorted",
			desktop.Entry{ID: "slack"},
			[]string{"Admin", "Work"},
		},
		{
			"a user tag wins over the entry's categories",
			desktop.Entry{ID: "code", Categories: []string{"Development"}},
			[]string{"Work"},
		},
		{
			"untagged falls back to the first category we recognise",
			desktop.Entry{ID: "firefox", Categories: []string{"Network", "WebBrowser"}},
			[]string{"Internet"},
		},
		{
			"unknown categories are skipped, not guessed at",
			desktop.Entry{ID: "nvim", Categories: []string{"Qt", "X-Vendor", "IDE"}},
			[]string{"Development"},
		},
		{
			"no tag and no categories",
			desktop.Entry{ID: "mystery"},
			[]string{FallbackGroup},
		},
		{
			"no tag and nothing recognisable",
			desktop.Entry{ID: "mystery", Categories: []string{"X-Vendor"}},
			[]string{FallbackGroup},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note the second case: "code" is tagged Work, so it must NOT also
			// pick up "Development". The tiers stop at the first match.
			if got := c.GroupsFor(tt.entry); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GroupsFor(%+v) = %v, want %v", tt.entry, got, tt.want)
			}
		})
	}
}

func TestDefaultPath(t *testing.T) {
	// t.Setenv restores the old value when the test ends, so these two cases
	// cannot leak into each other.
	t.Run("honours XDG_CONFIG_HOME", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", "/xdg")

		want := filepath.Join("/xdg", "apptray", "apptray.conf")
		if got := DefaultPath(); got != want {
			t.Errorf("DefaultPath() = %q, want %q", got, want)
		}
	})

	t.Run("falls back to ~/.config", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", "")
		t.Setenv("HOME", "/home/tester")

		want := filepath.Join("/home/tester", ".config", "apptray", "apptray.conf")
		if got := DefaultPath(); got != want {
			t.Errorf("DefaultPath() = %q, want %q", got, want)
		}
	})
}
