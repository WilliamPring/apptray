package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"apptray/internal/desktop"
)

func TestLoadMissingFileIsNotAnError(t *testing.T) {
	got, err := Load("testdata/does-not-exist.conf")
	if err != nil {
		t.Fatalf("Load returned error for a missing file: %v", err)
	}
	if !reflect.DeepEqual(got, Default()) {
		t.Errorf("got %+v, want the defaults %+v", got, Default())
	}
}

func TestLoad(t *testing.T) {
	got, err := Load("testdata/sample.conf")
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if got.TabsAt != TabsTop {
		t.Errorf("TabsAt = %q, want %q", got.TabsAt, TabsTop)
	}

	want := map[string][]string{
		"Work":      {"slack", "zoom", "code"},
		"Dev Tools": {"code"},
		"Media":     {"vlc", "spotify"},
	}
	if !reflect.DeepEqual(got.Groups, want) {
		t.Errorf("Groups = %+v, want %+v", got.Groups, want)
	}
}

func TestLoadIgnoresBadSettingValue(t *testing.T) {
	// "tabs = sideways" is not a value we accept: keep the default rather
	// than refusing to start.
	path := filepath.Join(t.TempDir(), "apptray.conf")
	writeFile(t, path, "[settings]\ntabs = sideways\n")

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if got.TabsAt != TabsBottom {
		t.Errorf("TabsAt = %q, want the default %q", got.TabsAt, TabsBottom)
	}
}

func TestGroupsFor(t *testing.T) {
	c, err := Load("testdata/sample.conf")
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	tests := []struct {
		name  string
		entry desktop.Entry
		want  []string
	}{
		{
			"an app in two groups gets both, sorted",
			desktop.Entry{ID: "code"},
			[]string{"Dev Tools", "Work"},
		},
		{
			"a user tag beats the app's own categories",
			desktop.Entry{ID: "vlc", Categories: []string{"AudioVideo"}},
			[]string{"Media"},
		},
		{
			"no tag falls back to the first known category",
			desktop.Entry{ID: "firefox", Categories: []string{"Network", "WebBrowser"}},
			[]string{"Internet"},
		},
		{
			"categories we don't recognise are skipped, not fatal",
			desktop.Entry{ID: "nvim", Categories: []string{"ConsoleOnly", "TextEditor"}},
			[]string{"Development"},
		},
		{
			"no tag and no usable category falls back to Other",
			desktop.Entry{ID: "mystery"},
			[]string{FallbackGroup},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := c.GroupsFor(tt.entry)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GroupsFor(%q) = %v, want %v", tt.entry.ID, got, tt.want)
			}
		})
	}
}

func TestGroupsForNeverReturnsEmpty(t *testing.T) {
	// Callers rely on this, so pin it down with the emptiest input there is.
	if got := Default().GroupsFor(desktop.Entry{}); len(got) == 0 {
		t.Error("GroupsFor returned an empty slice")
	}
}

func TestDefaultPathPrefersXDG(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/tmp/xdg")

	want := filepath.Join("/tmp/xdg", "apptray", "apptray.conf")
	if got := DefaultPath(); got != want {
		t.Errorf("DefaultPath() = %q, want %q", got, want)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}
