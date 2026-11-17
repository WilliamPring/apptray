# apptray — a tiny rofi-style app launcher in Go

Goal: type a few letters, see matching installed apps, in a small window on Arch Linux.
Scope for the POC: **lookup + display only** (launching an app comes later).

## How we work
Each step has a small goal and a list of Go concepts it teaches.
I (Claude) do step 0 and explain it. After that, **you write the code for each step**,
then paste it / tell me to look, and I review and explain what could be better.
Nothing moves on until the step works.

## Where the data comes from
Installed apps are described by `.desktop` files (INI-like text files):

- `/usr/share/applications/*.desktop`
- `~/.local/share/applications/*.desktop`

The part we care about is the `[Desktop Entry]` section: `Name=`, `Exec=`, `Comment=`,
`NoDisplay=`, `Type=`.

## Steps

| # | Goal | Go concepts you will meet |
|---|------|---------------------------|
| 0 | Empty module that builds and runs | `go mod`, `package main`, `go run` |
| 1 | Parse ONE `.desktop` file into a struct | structs, `os.Open`, `bufio.Scanner`, `strings`, errors, `defer` |
| 2 | Scan the application directories | `os.ReadDir`, `filepath.Join`, slices, `sort`, `os.UserHomeDir` |
| 3 | Filter entries by a query | functions, `strings.Contains`/`ToLower`, tests (`go test`, table-driven) |
| 4 | Parse `Categories=` and **group** entries by category | slices, `strings.Split`, `map[string][]Entry`, `reflect.DeepEqual` / `slices.Equal` |
| 5 | CLI POC: `go run . firefox` prints matches under group headings | `os.Args`, `flag`, splitting code into packages |
| 6 | Show it in a window, rofi-style (input on top, grouped list below) | a GUI toolkit (plan: Fyne), callbacks, goroutines basics |
| 7 | Polish: arrow keys, Esc to quit, fuzzy match, launch on Enter | `os/exec`, key events, better matching |
| 8 | Mode tabs: **Search · Categories · Power** | custom widgets, embedding, `map[string][]string`, XDG config, `flag` |

### Grouping design (step 4+)
- Group by the **first** entry in `Categories=` mapped to a friendly name
  (e.g. `WebBrowser`/`Network` -> "Internet", `Development` -> "Development").
  Unknown or missing -> "Other".
- Display: results sorted by group, with a non-selectable group header row, so it reads like
  a grouped rofi list. Typing still filters across all groups.
- Later idea: rofi-style "modes" (apps / run / windows) switched with Tab.

### "Like rofi" target look
Centered, borderless popup; text input on top; list below; Enter launches; Esc closes;
opened by an i3 keybinding.

Steps 1–4 are plain Go with **no GUI**, so you can build and test the logic
(even on the Mac) before touching windowing on Arch.

## GUI decision (for step 5)
Default plan: **Fyne** — pure-Go API, works on both X11 and Wayland, easy to learn.
Alternatives if we want something closer to rofi later: Gio, or raw X11 (`xgb`).
Confirmed environment: **X11 + i3** (main), possibly Wayland later.
- Fyne talks X11 natively, and on Wayland (e.g. sway) it runs through XWayland, so one codebase covers both.
- i3 will *tile* our window by default. Step 6 adds an i3 config snippet to make it a floating, centered, borderless popup bound to a key (like rofi):
  ```
  for_window [title="apptray"] floating enable, border none, move position center
  bindsym $mod+d exec --no-startup-id apptray
  ```
- Arch packages needed at step 5: `gcc libxcursor libxrandr libxinerama libxi mesa` (plus `libxxf86vm` if the build complains).

## Target layout (we grow into this, not up front)
```
apptray/
  main.go               # wiring only
  internal/desktop/     # parse + scan .desktop files   (steps 1-2)
  internal/search/      # filtering/matching            (step 3)
  internal/ui/          # window                        (step 5)
  internal/config/      # ~/.config/apptray/apptray.conf (step 8)
  internal/launcher/    # what the list shows, per tab   (step 8)
```

---

# Step 8 — Mode tabs

- Interactive mockup of the agreed design: `mockups/launcher.html` (open it in a browser).
- Step-by-step walkthrough of which file to edit and why: [GUIDE.md](GUIDE.md).

This section is the **what** — types, signatures, behaviour. GUIDE.md is the **how**.

## 8.0 The idea in one paragraph

Today the window is one thing: a search box over a list of apps. Step 8 keeps that as
the **Search** tab and adds two more: **Categories** (browse apps by tag, drill into one)
and **Power** (poweroff / reboot / lock). A tab strip sits at the bottom of the window.
The window still *opens* on Search, so nothing about today's behaviour is lost.

The extensibility trick is that all three tabs render the **same row type**. A tab is
just a different answer to "what rows should the list show right now?". Adding a fourth
mode later (SSH hosts, clipboard history, window switcher) means writing one function
that returns rows — no new UI code.

### Decisions already made (don't re-litigate while coding)
- Tabs are **modes**, not categories. Three of them, fixed, so the strip never overflows.
- Categories drill **in**: the tab shows a list of group names, Enter opens one.
- Typing while in Categories filters the **group names**; once drilled in, it filters the
  apps inside that group.
- A query **survives** switching tabs, so `Tab` out of Categories into Search turns a
  scoped search into a global one.
- Tagging keys on a **stable ID**, never on `Name` (see 8.1).
- Power actions are **stubbed** in this step — they log what they *would* run. We wire up
  real commands only once navigation is proven, so a stray Enter can't reboot your box.
- Launching apps is still step 7's job, not this step's.

### Prerequisite (do this first, on the Arch box)
`go.mod` has no `fyne.io/fyne/v2` line and there is no `go.sum`, so the tree does not
build right now. Run `go mod tidy` before anything else.

---

## 8.1 Give `Entry` a stable ID

**Why:** the config file has to name apps. `Name=` is a terrible key — it is translated,
it changes between versions, and two entries can share one. The freedesktop spec already
defines a **desktop file ID**: the basename minus `.desktop`, e.g.
`/usr/share/applications/firefox.desktop` → `firefox`.

**Change:** add one field to `desktop.Entry`:
```go
ID string // "firefox", from the filename — stable, unlike Name
```
and have `ParseFile` fill it from `path` before the scan loop.

**Watch out:** `TestParseFile` compares the whole struct with `reflect.DeepEqual`, so it
will fail until you add `ID: "sample"` to its `want`. That is the test doing its job.

**Bonus you get for free:** `scan.go` currently appends an app twice if it exists in both
`/usr/share/applications` and `~/.local/share/applications`. With an ID you can skip the
second one — the user's copy should win, so the per-user dir must be scanned first.

---

## 8.2 `internal/config` — user tags

### File format
One file, `$XDG_CONFIG_HOME/apptray/apptray.conf` (fall back to `~/.config/...`).
Same INI shape as a `.desktop` file, so you can reuse the parsing approach from step 1.

```ini
[settings]
tabs = bottom          # or "top"

[groups]
Work = slack; zoom; code
Media = vlc; spotify
```

Right-hand side is a `;`-separated list of **IDs** from 8.1. An app may appear under
several groups. A missing file is **not an error** — it means "defaults".

### Signatures
```go
package config

type Config struct {
    TabsAt string              // "bottom" (default) or "top"
    Groups map[string][]string // group name -> app IDs
}

// Default returns the config used when there is no file.
func Default() Config

// DefaultPath returns $XDG_CONFIG_HOME/apptray/apptray.conf,
// falling back to ~/.config/apptray/apptray.conf.
func DefaultPath() string

// Load reads path. A file that does not exist returns Default() and a nil error;
// a file that exists but cannot be read or parsed returns the error.
func Load(path string) (Config, error)

// GroupsFor returns every group e belongs to, in display order.
// Resolution order:
//   1. user groups from [groups] that list e.ID   (can be several)
//   2. otherwise the friendly name of the FIRST category we recognise
//   3. otherwise "Other"
// Never returns an empty slice.
func (c Config) GroupsFor(e desktop.Entry) []string
```

> Tier 2 originally said "the friendly name of `Categories[0]`". That is wrong for
> real files: VS Code ships `Categories=TextEditor;Development;IDE;`, and `TextEditor`
> is not one of the spec's main categories, so index 0 alone would file it under
> "Other". Walking the list and taking the first category we know fixes that and
> behaves identically for well-formed files like Firefox's `Network;WebBrowser;`.

The friendly-name map (`WebBrowser`/`Network` → "Internet", `Development` →
"Development", …) is the one already sketched under "Grouping design" above. Keep it
unexported in this package.

### Tests worth writing
- missing file → `Default()`, nil error
- an ID in two groups → `GroupsFor` returns both
- an app with no tag and no `Categories` → `["Other"]`
- unknown keys in `[settings]` are ignored rather than fatal

---

## 8.3 `internal/launcher` — rows, independent of Fyne

This package knows nothing about GUI. That is the point: it is pure data, so it is
testable without a display, and the UI becomes thin.

```go
package launcher

// Item is one row in the list, whatever tab produced it.
type Item struct {
    Title    string       // "Firefox", "Internet", "Reboot"
    Subtitle string       // app: its groups. category: "". power: the command.
    Count    int          // categories only: how many apps inside
    Folder   string       // non-empty => Enter drills into this category
    Run      func() error // non-nil => Enter performs this
}

type Tab int

const (
    TabSearch Tab = iota
    TabCategories
    TabPower
)

// TabNames is indexed by Tab.
var TabNames = []string{"Search", "Categories", "Power"}

func TabCount() int

// ParseTab maps a -mode flag value ("search", "categories", "power")
// to a Tab, case-insensitively. ok is false for anything else.
func ParseTab(s string) (tab Tab, ok bool)

// Rows is the whole model. open is the category currently drilled into ("" = none).
//
//   TabSearch                      -> apps matching query (all apps when query is "")
//   TabCategories, open == ""      -> one Item per group, Folder set, Count filled,
//                                     filtered by query matching the GROUP NAME
//   TabCategories, open == "Work"  -> apps in "Work" matching query
//   TabPower                       -> Power(), filtered by query
//
// Rows never returns nil; "no matches" is an empty slice.
func Rows(entries []desktop.Entry, c config.Config, tab Tab, open, query string) []Item

// Counts returns the number of rows each tab would show for an empty query,
// indexed by Tab — this is what the tab strip displays.
func Counts(entries []desktop.Entry, c config.Config) []int

// Power returns the power actions. For step 8 each Run only logs what it would do.
func Power() []Item
```

**Ordering:** groups alphabetical, apps alphabetical within a group, power actions in a
fixed hand-written order (Lock, Reboot, Poweroff) — least destructive first.

### Tests worth writing
These are the cheap, high-value ones, and they need no window:
- each `(tab, open, query)` combination returns the row count you expect
- `Rows(TabCategories, "", "")` has `Folder` set on every item and `Count > 0`
- drilling into a group that does not exist returns an empty slice, not a panic
- `Counts` has `len == TabCount()`
- `ParseTab("POWER")` → `TabPower, true`; `ParseTab("nope")` → `_, false`

---

## 8.4 `internal/ui` — the window

Reads `launcher.Rows` on every change and redraws. Hold this state in `Run`:

```go
tab   launcher.Tab // which mode
open  string       // category drilled into, "" when not
query string       // the entry's text
rows  []launcher.Item
```
and one unexported helper that re-runs `launcher.Rows` into `rows`, refreshes the list,
selects row 0 and updates the breadcrumb. Every key handler ends by calling it.

### Key map
| Key | Behaviour |
|-----|-----------|
| `Tab` / `Shift-Tab` | next / previous mode. Query is kept. |
| `←` `→` | same, **but only when the query is empty** (otherwise they move the text cursor) |
| `↑` `↓` | move the selection |
| `Enter` | `Folder != ""` → drill in; else `Run != nil` → run it |
| `Esc` | unwind one level: clear query → leave category → back to Search → quit |

That `Esc` ladder is why the launcher never feels like a trap — one key always goes
backwards, and only closes the window when there is nothing left to back out of.

### Fyne gotchas that will cost you an hour each
- Key events go to the **focused widget**. A `Canvas().SetOnTypedKey` handler never fires
  while the entry has focus. That is why `searchEntry` in `ui.go` embeds `widget.Entry`
  and overrides `TypedKey` — extend that type rather than adding a second mechanism.
- `widget.Entry` swallows `Tab` unless you override `AcceptsTab() bool` to return `false`.
- `fyne.KeyEvent` carries **no modifier info**, so Shift-Tab is not simply "Tab with
  shift". With three tabs, forward-only cycling that wraps is an acceptable first cut.
- `container.NewBorder(top, bottom, left, right, objects...)` stores the **centre
  objects first**, then the edges. If you build a two-label row for the list template and
  then index `Objects[0]`, know which one you are getting.
- `List.Select(0)` also scrolls the row into view — use it, don't hand-roll scrolling.
- Respect `cfg.TabsAt`: it decides whether the strip is the `top` or `bottom` argument to
  `NewBorder`. Nothing else about the layout changes.

Build the strip with `container.NewGridWithColumns(launcher.TabCount(), ...)` and
`widget.Button`; mark the active one with `Importance = widget.HighImportance`.

---

## 8.5 `main.go`

```go
mode := flag.String("mode", "search", "start in: search, categories or power")
```
Load the config, `ParseTab` the flag, and pass both plus the entries into `ui.Run`.
An unrecognised `-mode` should be a clear `log.Fatalf`, not a silent fall back to Search.

This is what makes a dedicated i3 binding possible:
```
bindsym $mod+shift+e exec --no-startup-id apptray -mode power
```

---

## 8.6 Order to build it in

Each line leaves the tree building and testable, so you can stop anywhere.

1. `go mod tidy` — get the tree compiling at all.
2. 8.1 `Entry.ID` + fix `entry_test.go`. Smallest possible change, proves the loop works.
3. 8.2 `internal/config` **with its tests, no UI**. Run `go test ./...`.
4. 8.3 `internal/launcher` **with its tests, still no UI**. At this point the whole
   feature is correct and proven, with not a single pixel drawn.
5. 8.4 the UI, in two passes: first just the tab strip switching between `Rows` results,
   then the drill-in and the `Esc` ladder.
6. 8.5 the flag.

## 8.7 Go concepts this step teaches

- **Struct embedding** (`searchEntry`) — Go's answer to inheritance: you get every method
  of `widget.Entry` for free and override only `TypedKey`.
- **Function values in a struct** (`Item.Run func() error`) — behaviour stored as data.
  This is what lets one list render apps, categories and power actions.
- **A named integer type with `iota`** (`Tab`) — cheap enums, and `TabNames` indexed by
  the type keeps display strings next to the values.
- **`map[string][]string`** — the group→IDs shape, and why map iteration order is random
  (hence the explicit sort before display).
- **Separating pure logic from I/O** — `internal/launcher` has no Fyne import, which is
  exactly why it is testable on the Mac with no display.
- **`flag`** — the standard library's argument parser, versus the raw `os.Args` of step 5.
