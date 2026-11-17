import (
	"log"
	"slices"
	"strings"

	"apptray/internal/config"
	"apptray/internal/desktop"
	"apptray/internal/search"
)

// Item is one row in the list, whatever mode produced it.
//
// Folder and Run are what make a single row type enough for apps, categories
// and power actions: a row carries its own behaviour, so the UI never has to
// ask what kind of row it is holding.
type Item struct {
	Title    string       // "Firefox", "Internet", "Reboot"
	Subtitle string       // app: its groups. category: "". power: the command.
	Count    int          // categories only: how many apps are inside
	Folder   string       // non-empty: Enter drills into this category
	Run      func() error // non-nil: Enter performs this
}

// Tab is a mode: one of the entries in the tab strip.
type Tab int

const (
	TabSearch Tab = iota
	TabCategories
	TabPower
)

// TabNames is indexed by Tab.
var TabNames = []string{"Search", "Categories", "Power"}

// TabCount returns how many modes there are.
func TabCount() int { return len(TabNames) }

// ParseTab maps a -mode flag value to a Tab, case-insensitively.
// ok is false for anything that is not a mode name.
func ParseTab(s string) (Tab, bool) {
	for i, name := range TabNames {
		if strings.EqualFold(strings.TrimSpace(s), name) {
			return Tab(i), true
		}
	}
	return TabSearch, false
}

// Rows is the whole model. open is the category currently drilled into
// ("" when we are not inside one).
//
//	TabSearch                     -> apps matching query (all apps when query is "")
//	TabCategories, open == ""     -> one row per group, filtered by GROUP NAME
//	TabCategories, open == "Work" -> the apps in "Work", filtered by app name
//	TabPower                      -> the power actions, filtered by title
//
// Rows never returns nil: "no matches" is an empty slice.
func Rows(entries []desktop.Entry, c config.Config, tab Tab, open, query string) []Item {
	items := []Item{}

	switch {
	case tab == TabSearch:
		for _, e := range search.Filter(entries, query) {
			items = append(items, appItem(e, c))
		}

	case tab == TabCategories && open == "":
		byGroup := groupIndex(entries, c)
		for _, name := range sortedKeys(byGroup) {
			if !matches(name, query) {
				continue
			}
			items = append(items, Item{
				Title:  name,
				Count:  byGroup[name],
				Folder: name, // non-empty, so Enter drills in
			})
		}

	case tab == TabCategories:
		// Inside a category: entries arrive sorted by name from desktop.Scan,
		// so filtering preserves that order.
		for _, e := range entries {
			if slices.Contains(c.GroupsFor(e), open) && matches(e.Name, query) {
				items = append(items, appItem(e, c))
			}
		}

	case tab == TabPower:
		for _, item := range Power() {
			if matches(item.Title, query) {
				items = append(items, item)
			}
		}
	}

	return items
}

// Counts returns how many rows each tab holds for an empty query, indexed by
// Tab. This is what the tab strip displays.
func Counts(entries []desktop.Entry, c config.Config) []int {
	counts := make([]int, TabCount())
	for tab := range counts {
		counts[tab] = len(Rows(entries, c, Tab(tab), "", ""))
	}
	return counts
}

// Power returns the power actions, least destructive first.
//
// For now each Run only logs what it would do. Wiring up the real commands
// waits until navigation is proven, so a stray Enter cannot reboot the machine.
func Power() []Item {
	return []Item{
		stub("Lock", "loginctl lock-session"),
		stub("Reboot", "systemctl reboot"),
		stub("Poweroff", "systemctl poweroff"),
	}
}

func stub(title, command string) Item {
	return Item{
		Title:    title,
		Subtitle: command,
		Run: func() error {
			log.Printf("apptray: would run %q", command)
			return nil
		},
	}
}

// appItem renders one app as a row. Launching is still step 7's job, so Run
// stays nil and the UI treats the row as inert.
func appItem(e desktop.Entry, c config.Config) Item {
	return Item{
		Title:    e.Name,
		Subtitle: strings.Join(c.GroupsFor(e), ", "),
	}
}

// groupIndex counts how many of entries fall into each group. Groups named in
// the config but with nothing installed never appear, because the count is
// built from the entries rather than from the config.
func groupIndex(entries []desktop.Entry, c config.Config) map[string]int {
	byGroup := map[string]int{}
	for _, e := range entries {
		for _, name := range c.GroupsFor(e) {
			byGroup[name]++
		}
	}
	return byGroup
}

func sortedKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys) // map order is random; the list must not reshuffle
	return keys
}

// matches is the same case-insensitive contains that search.Filter applies to
// app names, for the places where we filter plain strings instead of entries.
func matches(s, query string) bool {
	q := strings.ToLower(strings.TrimSpace(query))
	return q == "" || strings.Contains(strings.ToLower(s), q)
}