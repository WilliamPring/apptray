package launcher

import (
	"testing"

	"apptray/internal/config"
	"apptray/internal/desktop"
)

// fixture mirrors what desktop.Scan would hand us: sorted by name.
//
// Resulting groups:
//
//	code    -> Dev Tools, Work  (user tags, and it is in two of them)
//	firefox -> Internet         (no tag, falls back to Categories)
//	gimp    -> Graphics         (ditto)
//	vlc     -> Media            (user tag beats its AudioVideo category)
//
// "zoom" is tagged Work in the config but is not installed, so it must not
// inflate the Work count.
func fixture() ([]desktop.Entry, config.Config) {
	entries := []desktop.Entry{
		{ID: "code", Name: "Code", Categories: []string{"TextEditor", "Development"}},
		{ID: "firefox", Name: "Firefox", Categories: []string{"Network", "WebBrowser"}},
		{ID: "gimp", Name: "GIMP", Categories: []string{"Graphics"}},
		{ID: "vlc", Name: "VLC", Categories: []string{"AudioVideo"}},
	}
	c := config.Config{
		TabsAt: config.TabsBottom,
		Groups: map[string][]string{
			"Work":      {"code", "zoom"},
			"Dev Tools": {"code"},
			"Media":     {"vlc"},
		},
	}
	return entries, c
}

func TestRows(t *testing.T) {
	entries, c := fixture()

	tests := []struct {
		name  string
		tab   Tab
		open  string
		query string
		want  int
	}{
		{"search with no query shows every app", TabSearch, "", "", 4},
		{"search is case insensitive", TabSearch, "", "FIRE", 1},
		{"search that matches nothing", TabSearch, "", "zzz", 0},

		{"categories lists every group in use", TabCategories, "", "", 5},
		{"typing in categories filters group names", TabCategories, "", "dev", 1},
		{"a group name that matches nothing", TabCategories, "", "zzz", 0},

		{"drilling into a group shows its apps", TabCategories, "Work", "", 1},
		{"drilling in then filtering", TabCategories, "Dev Tools", "cod", 1},
		{"drilling into a group that does not exist", TabCategories, "Nope", "", 0},

		{"power lists every action", TabPower, "", "", 3},
		{"power can be filtered too", TabPower, "", "reboot", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Rows(entries, c, tt.tab, tt.open, tt.query)
			if len(got) != tt.want {
				t.Errorf("Rows(%v, %q, %q) returned %d rows, want %d: %+v",
					tt.tab, tt.open, tt.query, len(got), tt.want, got)
			}
		})
	}
}

func TestRowsNeverReturnsNil(t *testing.T) {
	// The UI indexes this straight into a list, so nil would be a landmine.
	entries, c := fixture()
	if got := Rows(entries, c, TabCategories, "Nope", ""); got == nil {
		t.Error("Rows returned nil, want an empty slice")
	}
}

func TestCategoryRowsAreDrillable(t *testing.T) {
	entries, c := fixture()

	for _, item := range Rows(entries, c, TabCategories, "", "") {
		if item.Folder == "" {
			t.Errorf("category %q has no Folder, so Enter would do nothing", item.Title)
		}
		if item.Count <= 0 {
			t.Errorf("category %q has count %d, want at least 1", item.Title, item.Count)
		}
	}
}

func TestCategoryCountsOnlyInstalledApps(t *testing.T) {
	entries, c := fixture()

	for _, item := range Rows(entries, c, TabCategories, "", "") {
		if item.Title == "Work" && item.Count != 1 {
			t.Errorf("Work count = %d, want 1 (zoom is tagged but not installed)", item.Count)
		}
	}
}

func TestAppRowSubtitleListsEveryGroup(t *testing.T) {
	entries, c := fixture()

	rows := Rows(entries, c, TabSearch, "", "code")
	if len(rows) != 1 {
		t.Fatalf("got %d rows, want 1", len(rows))
	}
	if want := "Dev Tools, Work"; rows[0].Subtitle != want {
		t.Errorf("subtitle = %q, want %q", rows[0].Subtitle, want)
	}
}

func TestPowerActionsAreStubs(t *testing.T) {
	// Every power row must be runnable, and must not do anything real yet.
	for _, item := range Power() {
		if item.Run == nil {
			t.Errorf("power action %q has no Run", item.Title)
			continue
		}
		if err := item.Run(); err != nil {
			t.Errorf("power action %q returned %v, want nil", item.Title, err)
		}
		if item.Subtitle == "" {
			t.Errorf("power action %q does not say what it would run", item.Title)
		}
	}
}

func TestCounts(t *testing.T) {
	entries, c := fixture()

	got := Counts(entries, c)
	if len(got) != TabCount() {
		t.Fatalf("Counts returned %d values, want one per tab (%d)", len(got), TabCount())
	}

	want := []int{4, 5, 3} // apps, groups, power actions
	for tab, n := range want {
		if got[tab] != n {
			t.Errorf("%s count = %d, want %d", TabNames[tab], got[tab], n)
		}
	}
}

func TestParseTab(t *testing.T) {
	tests := []struct {
		in     string
		want   Tab
		wantOK bool
	}{
		{"search", TabSearch, true},
		{"categories", TabCategories, true},
		{"POWER", TabPower, true},
		{" power ", TabPower, true},
		{"nope", TabSearch, false},
		{"", TabSearch, false},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, ok := ParseTab(tt.in)
			if ok != tt.wantOK || got != tt.want {
				t.Errorf("ParseTab(%q) = %v, %v; want %v, %v", tt.in, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

func TestTabNamesCoversEveryTab(t *testing.T) {
	// TabNames is indexed by Tab, so a new tab without a name is an out-of-range
	// panic waiting to happen.
	if TabNames[TabPower] != "Power" {
		t.Errorf("TabNames[TabPower] = %q, want %q", TabNames[TabPower], "Power")
	}
}
