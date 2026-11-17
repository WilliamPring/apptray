package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"apptray/internal/desktop"
	"apptray/internal/search"
)

// searchEntry is a text entry that gives Esc back to the launcher instead of
// swallowing it like widget.Entry does.
type searchEntry struct {
	widget.Entry
	onEscape func()
}

func newSearchEntry(onEscape func()) *searchEntry {
	e := &searchEntry{onEscape: onEscape}
	e.ExtendBaseWidget(e)
	return e
}

func (e *searchEntry) TypedKey(key *fyne.KeyEvent) {
	if key.Name == fyne.KeyEscape {
		e.onEscape()
		return
	}
	e.Entry.TypedKey(key)
}

// Run opens the launcher window and blocks until it is closed.
func Run(entries []desktop.Entry) {
	a := app.New()
	w := a.NewWindow("apptray") // i3 will match on this title

	// matches is what the list currently shows. The list reads it through closures,
	// so when we reassign it and call Refresh(), the list redraws with the new data.
	matches := entries

	input := newSearchEntry(w.Close)
	input.SetPlaceHolder("Search apps...")

	list := widget.NewList(
		func() int { return len(matches) },
		func() fyne.CanvasObject { return widget.NewLabel("template") },
		func(i widget.ListItemID, o fyne.CanvasObject) {
			o.(*widget.Label).SetText(matches[i].Name)
		},
	)

	input.OnChanged = func(query string) {
		matches = search.Filter(entries, query)
		list.Refresh()
	}

	// Input on top, list fills the rest: this is the rofi layout.
	w.SetContent(container.NewBorder(input, nil, nil, nil, list))
	w.Resize(fyne.NewSize(600, 400))
	w.CenterOnScreen()
	w.Canvas().Focus(input) // start typing immediately

	// Esc also works when focus has moved off the entry (e.g. onto the list).
	w.Canvas().SetOnTypedKey(func(key *fyne.KeyEvent) {
		if key.Name == fyne.KeyEscape {
			w.Close()
		}
	})

	w.ShowAndRun()
}
