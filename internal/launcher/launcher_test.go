package launcher

import (
	"reflect"
	"testing"

	"apptray/internal/config"
	"apptray/internal/desktop"
)

// fixture returns the same small world for every test: five apps, already in
// the name order that desktop.Scan would hand us, and a config that tags two
// of them. The resulting groups are:
//
//	Graphics (GIMP), Internet (Firefox), Other (Mystery), Work (Code, Slack)
//
// Note that Code and Slack are tagged, so they leave Development and Internet
// entirely — a tag replaces the category, it does not add to it.
func fixture() ([]desktop.Entry, config.Config) {
	entries := []desktop.Entry{
		{ID: "code", Name: "Code", Categories: []string{"TextEditor", "Development"}},
		{ID: "firefox", Name: "Firefox", Categories: []string{"Network", "WebBrowser"}},
		{ID: "gimp", Name: "GIMP", Categories: []string{"Graphics"}},
		{ID: "mystery", Name: "Mystery"},
		{ID: "slack", Name: "Slack", Categories: []string{"Network"}},
	}
	cfg := config.Config{
		TabsAt: config.TabsBottom,
		Groups: map[string][]string{"Work": {"slack", "code"}},
	}
	return entries, cfg
}

func TestRows(t *testing.T) {
	entries, cfg := fixture()

	tests := []struct {
		name  string
		tab   Tab
		open  string
		query string
		want  int
	}{
		{"search, no query, every app", TabSearch, "", "", 5},
		{"search filters by name", TabSearch, "", "fire", 1},
		{"search ignores case", TabSearch, "", "GIMP", 1},
		{"search with no match", TabSearch, "", "zzz", 0},

		{"categories lists the groups", TabCategories, "", "", 4},
		{"categories filters the GROUP name, not the apps", TabCategories, "", "work", 1},
		{"a group name that matches nothing", TabCategories, "", "zzz", 0},

		{"drilled in, the apps of that group", TabCategories, "Work", "", 2},
		{"drilled in, the query filters apps", TabCategories, "Work", "sla", 1},
		{"drilled into a group that does not exist", TabCategories, "Nope", "", 0},

		{"power lists every action", TabPower, "", "", 3},
		{"power filters by title", TabPower, "", "boot", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Rows(entries, cfg, tt.tab, tt.open, tt.query)

			// "No matches" must be an empty slice, never nil: the UI ranges
			// over this and asks for len() without a nil check.
			if got == nil {
				t.Fatal("Rows returned nil")
			}
			if len(got) != tt.want {
				t.Errorf("Rows(%v, %q, %q) returned %d rows, want %d",
					tt.tab, tt.open, tt.query, len(got), tt.want)
			}
		})
	}
}

func TestRowsCategoriesAreDrillable(t *testing.T) {
	entries, cfg := fixture()

	got := Rows(entries, cfg, TabCategories, "", "")

	// Map iteration order is random, so this also pins down the sort.
	wantTitles := []string{"Graphics", "Internet", "Other", "Work"}
	titles := make([]string, len(got))
	for i, item := range got {
		titles[i] = item.Title
	}
	if !reflect.DeepEqual(titles, wantTitles) {
		t.Errorf("group titles = %v, want %v", titles, wantTitles)
	}

	for _, item := range got {
		// Folder is what tells the UI that Enter means "drill in".
		if item.Folder != item.Title {
			t.Errorf("%q: Folder = %q, want %q", item.Title, item.Folder, item.Title)
		}
		if item.Count <= 0 {
			t.Errorf("%q: Count = %d, want more than 0", item.Title, item.Count)
		}
	}

	if work := got[3]; work.Count != 2 {
		t.Errorf("Work holds %d apps, want 2 (Code and Slack)", work.Count)
	}
}

func TestRowsInsideAGroup(t *testing.T) {
	entries, cfg := fixture()

	got := Rows(entries, cfg, TabCategories, "Work", "")

	wantTitles := []string{"Code", "Slack"} // the scan order, preserved
	titles := make([]string, len(got))
	for i, item := range got {
		titles[i] = item.Title
		// An app is a leaf: Enter must not try to drill into it.
		if item.Folder != "" {
			t.Errorf("%q: Folder = %q, want empty", item.Title, item.Folder)
		}
	}
	if !reflect.DeepEqual(titles, wantTitles) {
		t.Errorf("titles = %v, want %v", titles, wantTitles)
	}
}

func TestCounts(t *testing.T) {
	entries, cfg := fixture()

	got := Counts(entries, cfg)

	if len(got) != TabCount() {
		t.Fatalf("Counts returned %d numbers, want one per tab (%d)", len(got), TabCount())
	}

	want := []int{5, 4, 3} // apps, groups, power actions
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Counts = %v, want %v", got, want)
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
		{"  Search  ", TabSearch, true},
		{"nope", TabSearch, false},
		{"", TabSearch, false},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, ok := ParseTab(tt.in)
			if ok != tt.wantOK {
				t.Fatalf("ParseTab(%q) ok = %v, want %v", tt.in, ok, tt.wantOK)
			}
			if ok && got != tt.want {
				t.Errorf("ParseTab(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestPower(t *testing.T) {
	got := Power()

	// Hand-written order, least destructive first: a mis-aimed Enter should
	// lock the screen, not power the machine off.
	wantTitles := []string{"Lock", "Reboot", "Poweroff"}
	titles := make([]string, len(got))
	for i, item := range got {
		titles[i] = item.Title
	}
	if !reflect.DeepEqual(titles, wantTitles) {
		t.Errorf("power titles = %v, want %v", titles, wantTitles)
	}

	for _, item := range got {
		if item.Subtitle == "" {
			t.Errorf("%q: Subtitle is empty, the row should show its command", item.Title)
		}
		if item.Run == nil {
			t.Fatalf("%q: Run is nil, Enter would do nothing", item.Title)
		}
		// Still a stub in step 8: it logs rather than rebooting the test host.
		if err := item.Run(); err != nil {
			t.Errorf("%q: Run returned %v", item.Title, err)
		}
	}
}

func TestTabNamesCoverEveryTab(t *testing.T) {
	// TabNames is indexed by Tab, so a new tab added without a name would
	// panic at render time rather than here.
	if TabCount() != len(TabNames) {
		t.Fatalf("TabCount() = %d, len(TabNames) = %d", TabCount(), len(TabNames))
	}
	if TabNames[TabPower] != "Power" {
		t.Errorf("TabNames[TabPower] = %q, want %q", TabNames[TabPower], "Power")
	}
}
