package ui

import (
	"fmt"
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"apptray/internal/config"
	"apptray/internal/desktop"
	"apptray/internal/launcher"
)

// searchEntry is a text entry that hands the navigation keys back to the
// launcher instead of swallowing them like widget.Entry does.
type searchEntry struct {
	widget.Entry
	onKey func(*fyne.KeyEvent) bool // true: handled, don't pass it on
}

func newSearchEntry(onKey func(*fyne.KeyEvent) bool) *searchEntry {
	e := &searchEntry{onKey: onKey}
	e.ExtendBaseWidget(e)
	return e
}

// AcceptsTab reports false so Tab reaches TypedKey. widget.Entry eats it
// otherwise, and the tab strip never sees the key.
func (e *searchEntry) AcceptsTab() bool { return false }

func (e *searchEntry) TypedKey(key *fyne.KeyEvent) {
	if e.onKey(key) {
		return
	}
	e.Entry.TypedKey(key)
}

// Run opens the launcher window on startTab and blocks until it is closed.
func Run(entries []desktop.Entry, cfg config.Config, startTab launcher.Tab) {
	a := app.New()
	w := a.NewWindow("apptray") // i3 will match on this title

	// The whole window is derived from these four. Every key handler changes
	// one of them and then calls refresh.
	tab := startTab
	open := ""    // category drilled into, "" when not
	query := ""   // the entry's text
	var rows []launcher.Item
	selected := 0

	var input *searchEntry
	var buttons []*widget.Button

	breadcrumb := widget.NewLabel("")

	list := widget.NewList(
		func() int { return len(rows) },
		func() fyne.CanvasObject {
			// Objects[0] is the title, Objects[2] the subtitle; the spacer
			// between them pushes the subtitle to the right edge.
			return container.NewHBox(
				widget.NewLabel("title"),
				layout.NewSpacer(),
				widget.NewLabel("subtitle"),
			)
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			objects := o.(*fyne.Container).Objects
			objects[0].(*widget.Label).SetText(rows[i].Title)
			objects[2].(*widget.Label).SetText(subtitle(rows[i]))
		},
	)
	list.OnSelected = func(i widget.ListItemID) { selected = i }

	// The one place the list is rebuilt. Every handler below ends here.
	refresh := func() {
		rows = launcher.Rows(entries, cfg, tab, open, query)
		list.UnselectAll()
		list.Refresh()

		selected = 0
		if len(rows) > 0 {
			list.Select(0) // also scrolls the row into view
		}

		breadcrumb.SetText(crumb(tab, open, len(rows)))
		for i, b := range buttons {
			b.Importance = widget.MediumImportance
			if launcher.Tab(i) == tab {
				b.Importance = widget.HighImportance
			}
			b.Refresh()
		}
	}

	// setTab wraps in both directions. The query survives the switch on
	// purpose: tabbing out of a category turns a scoped search into a global
	// one. The open category does not — you have left it.
	setTab := func(t launcher.Tab) {
		n := launcher.TabCount()
		tab = launcher.Tab(((int(t) % n) + n) % n)
		open = ""
		refresh()
		w.Canvas().Focus(input) // a button tap steals focus; take it back
	}

	move := func(delta int) {
		if len(rows) == 0 {
			return
		}
		i := min(max(selected+delta, 0), len(rows)-1)
		selected = i
		list.Select(i)
	}

	// Enter asks two questions, in order: do you have a Folder? then do you
	// have a Run? The row answers; the window never asks what kind it is.
	activate := func() {
		if selected < 0 || selected >= len(rows) {
			return
		}
		item := rows[selected]
		switch {
		case item.Folder != "":
			open = item.Folder
			input.SetText("") // the query picked the group; inside it, start fresh
			refresh()
		case item.Run != nil:
			if err := item.Run(); err != nil {
				// Stay open and keep the error on stderr: the user can pick
				// something else instead of the window vanishing silently.
				log.Printf("apptray: %v", err)
				return
			}
			w.Close() // the launcher's job is done; get out of the way
		}
	}

	// One key always goes backwards, and only closes the window when there is
	// nothing left to back out of.
	escape := func() {
		switch {
		case query != "":
			input.SetText("") // fires OnChanged, which refreshes
		case open != "":
			open = ""
			refresh()
		case tab != launcher.TabSearch:
			tab = launcher.TabSearch
			refresh()
		default:
			w.Close()
		}
	}

	input = newSearchEntry(func(key *fyne.KeyEvent) bool {
		switch key.Name {
		case fyne.KeyEscape:
			escape()
		case fyne.KeyReturn, fyne.KeyEnter:
			activate()
		case fyne.KeyDown:
			move(1)
		case fyne.KeyUp:
			move(-1)
		case fyne.KeyTab:
			// fyne.KeyEvent carries no modifiers, so Shift-Tab is not
			// distinguishable: cycle forward and wrap.
			setTab(tab + 1)
		case fyne.KeyLeft, fyne.KeyRight:
			// These move the text cursor, so they are only tab switchers
			// while there is no text to move through.
			if query != "" {
				return false
			}
			if key.Name == fyne.KeyLeft {
				setTab(tab - 1)
			} else {
				setTab(tab + 1)
			}
		default:
			return false
		}
		return true
	})
	input.SetPlaceHolder("Search apps...")
	input.OnChanged = func(s string) {
		query = s
		refresh()
	}

	counts := launcher.Counts(entries, cfg)
	cells := make([]fyne.CanvasObject, launcher.TabCount())
	buttons = make([]*widget.Button, launcher.TabCount())
	for i, name := range launcher.TabNames {
		b := widget.NewButton(fmt.Sprintf("%s (%d)", name, counts[i]), func() {
			setTab(launcher.Tab(i))
		})
		buttons[i] = b
		cells[i] = b
	}
	strip := container.NewGridWithColumns(launcher.TabCount(), cells...)

	// Input on top, list fills the rest: this is the rofi layout. cfg.TabsAt
	// only decides which edge the strip is glued to.
	header := container.NewVBox(input, breadcrumb)
	content := container.NewBorder(header, strip, nil, nil, list)
	if cfg.TabsAt == config.TabsTop {
		content = container.NewBorder(container.NewVBox(strip, header), nil, nil, nil, list)
	}

	w.SetContent(content)
	w.Resize(fyne.NewSize(600, 400))
	w.CenterOnScreen()
	w.Canvas().Focus(input) // start typing immediately

	// Esc also works when focus has moved off the entry (e.g. onto the list).
	w.Canvas().SetOnTypedKey(func(key *fyne.KeyEvent) {
		if key.Name == fyne.KeyEscape {
			escape()
		}
	})

	refresh()
	w.ShowAndRun()
}

// subtitle is what the right-hand label shows: how many apps a category holds,
// otherwise whatever the row carries (an app's groups, a power command).
func subtitle(item launcher.Item) string {
	if item.Folder != "" {
		return fmt.Sprintf("%d apps", item.Count)
	}
	return item.Subtitle
}

func crumb(tab launcher.Tab, open string, n int) string {
	path := launcher.TabNames[tab]
	if open != "" {
		path += " / " + open
	}
	return fmt.Sprintf("%s (%d)", path, n)
}
