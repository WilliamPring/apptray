# Step 8 implementation guide

Three docs, three jobs:

| Doc | Answers |
|---|---|
| [README.md](README.md) | how do I get this building and running? |
| [PLAN.md](PLAN.md) | **what** am I writing? (roadmap, signatures, behaviour) |
| **GUIDE.md** (this one) | **which file, in what order, and why?** |

Read PLAN.md §8 first for the signatures. This walks you through typing them in.
Each round ends with something you can run, so you are never more than one step away
from a working tree.

---

## Round 0 — make the tree build

No file to edit. On the Arch box:

```bash
go get fyne.io/fyne/v2@latest && go mod tidy
```

Right now `go.mod` is 24 bytes — module name and Go version, nothing else. `ui.go`
imports Fyne, so nothing in this repo compiles today. Do this before you judge any
error message, or you will chase a phantom.

---

## Round 1 — `internal/desktop/entry.go`

**The change:** one field on the struct, one line in `ParseFile`.

```go
ID string // "firefox", from the filename
```

**Where in `ParseFile`:** line 40 currently reads `var e Entry`. That is your spot — the
ID comes from the *filename*, not the file's contents, so it is known before you read a
single byte. Set it as you declare `e`, then leave the scan loop alone.

**How to build it:** `filepath.Base(path)` gives you `firefox.desktop`;
`strings.TrimSuffix` takes the extension off. `strings` is already imported,
`path/filepath` is not — you will add it.

**Why not just use `Name`?** Because your config file has to refer to apps somehow, and
`Name=` is translated per locale, changes between versions, and is not unique. The
filename is the freedesktop spec's own identifier. Tag `code`, and it stays `code`
whether the menu says "Visual Studio Code" or "Code - OSS".

### `internal/desktop/entry_test.go`

This **will** fail, on purpose. Line 24 uses `reflect.DeepEqual` on the whole struct, so
a new field means `got` and `want` differ. Add `ID: "sample"` to the `want` literal.

Worth noticing *why* it is `"sample"`: the test parses `testdata/sample.desktop`, so the
ID is the basename of that path. The test is confirming you read the filename, not
something inside the file.

**Run:** `go test ./internal/desktop/` — passes, and you have proven the loop works end
to end.

### `internal/desktop/scan.go` — optional, but it is free now

There is a latent bug: an app installed in both `/usr/share/applications` and
`~/.local/share/applications` gets appended **twice** (line 46), and you would see it
twice in the launcher. With an ID you can keep a `map[string]bool` of what you have seen
and skip repeats.

**The catch:** `DefaultDirs` (line 14) lists the system dir **first**. "First one wins"
dedupe would therefore make the system copy win — backwards. A user's `~/.local`
override exists precisely to beat the system one, so you would reverse the order in
`DefaultDirs` as part of this.

Skip it if you would rather stay focused; it is not load-bearing for step 8.

---

## Round 2 — new file `internal/config/config.go`

Create the directory. This package reads `~/.config/apptray/apptray.conf` and answers one
question: *what groups does this app belong to?*

**What goes in it:** the `Config` struct, `Default`, `DefaultPath`, `Load`, `GroupsFor`
(signatures in PLAN.md §8.2), plus an **unexported** map from freedesktop category names
to friendly ones (`"WebBrowser"` → `"Internet"`). Unexported because nothing outside this
package should care how the mapping works.

**The parsing is step 1 again.** `[section]` headers, `key=value` lines, `#` comments,
`strings.Cut` on the first `=`. You already wrote this once in `ParseFile`. Do not try to
share the code between them — they are similar today but will drift, and the duplication
is about fifteen lines. Resist the abstraction.

**The one genuinely interesting decision:** a missing file returns `Default()` **and a
nil error**. Most people's instinct is to return an error and let the caller decide. Do
not — "you have not written a config yet" is the normal case for a new install, not a
failure. Use `errors.Is(err, fs.ErrNotExist)` to tell "no file" from "file exists but I
cannot read it", exactly like `scan.go` does at line 31. A permissions problem is still a
real error.

**`GroupsFor` is the heart of the package.** Three tiers, in order: your tags win; else
the friendly name of `Categories[0]`; else `"Other"`. It must never return an empty
slice — if it can, every caller needs a nil check, and one of them will forget.

### `internal/config/config_test.go` + `internal/config/testdata/`

Same `testdata/` pattern as `desktop`. Write a small `.conf` fixture and test:

- missing file → defaults, nil error
- one ID in two groups → both returned
- untagged app with no `Categories` → `["Other"]`
- junk key in `[settings]` → ignored, not fatal

**Run:** `go test ./...` — and note this works **on the Mac**. No display needed.

---

## Round 3 — new file `internal/launcher/launcher.go`

The most important file in step 8, and it imports **no Fyne**. Check that when you are
done: if `fyne.io` appears in the import block, the design has leaked.

**What goes in it:** `Item`, `Tab` + the `iota` constants, `TabNames`, `TabCount`,
`ParseTab`, `Rows`, `Counts`, `Power` (PLAN.md §8.3).

**`Item` is where the extensibility lives.** One row type for apps, categories *and*
power actions. The two fields that make that work:

- `Folder string` — non-empty means "Enter drills into this"
- `Run func() error` — a **function stored in a struct field**. The row carries its own
  behaviour, so the UI never asks "what kind of row is this?". It asks "do you have a
  `Folder`? no? then do you have a `Run`?" That is the whole dispatch.

This is why adding an SSH-hosts mode later is one function, not a UI change.

**`Rows` is a switch on `(tab, open)`** with four arms — the table in PLAN.md §8.3 is
literally the spec. For `TabSearch` you delegate to `search.Filter`, which already exists
and already handles the empty query. For `TabCategories` with `open == ""` you are
filtering **group names**, not app names, which is the bit people get wrong on the first
pass.

**Map iteration order is random in Go.** Building the category list means iterating
`c.Groups`, so you must sort before returning or the list reshuffles on every keystroke.
This will bite you visibly.

**`Power` returns stubs.** Each `Run` does `log.Printf("would run: ...")` and nothing
else. Keep it that way until navigation is proven — otherwise a mistyped Enter during
development reboots your machine.

### `internal/launcher/launcher_test.go`

Table-driven, like `search_test.go`:

- cases over `(tab, open, query)` asserting row counts
- every category row has `Folder` set and `Count > 0`
- drilling into a nonexistent group returns empty, not a panic
- `len(Counts()) == TabCount()`
- `ParseTab("POWER")` → `TabPower, true`; `ParseTab("nope")` → `_, false`

**When this passes, step 8 is functionally done** — correct, tested, and not one pixel
drawn. The UI after this is presentation only. That is the payoff for keeping Fyne out of
this file.

---

## Round 4 — `internal/ui/ui.go`

Now the window. Four state variables in `Run` (`tab`, `open`, `query`, `rows`) and **one
unexported helper** that re-runs `launcher.Rows`, refreshes the list, selects row 0 and
updates the breadcrumb. Every key handler ends by calling that helper. If you find
yourself refreshing the list in two different places, you have drifted.

**Do it in two passes.** First just the tab strip switching between `Rows` results. Only
when that is solid, add drill-in and the Esc ladder.

**Extend `searchEntry`, do not add a second mechanism.** It is already there at
`ui.go:15` doing exactly this for Esc. Tab and the arrows go in the same `TypedKey`
override. You will also need `AcceptsTab() bool` returning `false`, or `widget.Entry`
eats the Tab key before you see it.

**The ←/→ conflict is real:** those keys move the text cursor. Only treat them as
tab-switchers when the query is empty.

The rest of the traps are listed in PLAN.md §8.4 — `NewBorder` argument order, no
modifiers in `KeyEvent`, `List.Select` scrolling for you. Read that block before you
start, not after.

---

## Round 5 — `main.go`, then `README.md`

`main.go` stays wiring only: `flag.String("mode", ...)`, load the config, `ParseTab`,
hand everything to `ui.Run`. An unknown mode is `log.Fatalf` — a typo'd `-mode powr`
silently opening Search is worse than an error.

Then update **`README.md`'s "Project layout"** — it still lists only three packages.
You will have five.

---

## If you get stuck

- **It does not compile at all** → Round 0. Nothing works before `go mod tidy`.
- **`entry_test.go` fails after Round 1** → expected. Add `ID: "sample"`.
- **The category list reorders itself as you type** → you are iterating a map without
  sorting (Round 3).
- **Tab does nothing in the window** → `AcceptsTab()` (Round 4).
- **Esc does nothing after you edit `TypedKey`** → you probably dropped the early
  `return` and are passing Esc through to `widget.Entry`, which swallows it.
