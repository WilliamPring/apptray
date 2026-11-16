# apptray

A small rofi-style application launcher written in Go, for Arch Linux (X11 / i3).
Type to filter your installed apps. See [PLAN.md](PLAN.md) for the learning roadmap.

## Dependencies

### System packages (Arch Linux)

```bash
sudo pacman -S go gcc libxcursor libxrandr libxinerama libxi libxxf86vm mesa
```

| Package | Why |
|---|---|
| `go` | The Go toolchain (the version in `go.mod` or newer) |
| `gcc` | Fyne uses cgo, so it needs a C compiler |
| `libxcursor` `libxrandr` `libxinerama` `libxi` `libxxf86vm` | X11 libraries used by Fyne's window layer |
| `mesa` | OpenGL, which Fyne renders with |

### Go modules

- [`fyne.io/fyne/v2`](https://fyne.io) for the GUI

Fetch them once from the project root:

```bash
go get fyne.io/fyne/v2@latest
go mod tidy
```

`go mod tidy` records the dependency in `go.mod` and `go.sum`. After that, a plain
`go build` downloads anything missing.

## Build and run

```bash
go run .                  # build and run
go build -o apptray .     # build a binary
go test ./...             # run all tests
```

The first build is slow (a few minutes) because Fyne compiles C code. Later builds are fast.

The packages without a GUI (`internal/desktop`, `internal/search`) have no system
dependencies beyond Go, so their tests also run on macOS.

## Use with i3

Add to `~/.config/i3/config`, then reload with `$mod+Shift+c`:

```
for_window [title="apptray"] floating enable, border none, move position center
bindsym $mod+d exec --no-startup-id /path/to/apptray
```

Change `$mod+d` if that key is already bound (for example to rofi or dmenu).

## Project layout

```
main.go                 wiring only
internal/desktop/       parse and scan .desktop files
internal/search/        filter entries by query
internal/ui/            Fyne window
```
