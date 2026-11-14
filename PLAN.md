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
| 4 | CLI POC: `go run . firefox` prints matches | `os.Args`, `flag`, splitting code into packages |
| 5 | Show it in a window (text box + result list) | a GUI toolkit (plan: Fyne), callbacks, goroutines basics |
| 6 | Polish: arrow keys, Esc to quit, fuzzy match, launch on Enter | `os/exec`, key events, better matching |

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
```
