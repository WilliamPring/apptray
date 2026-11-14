package main

import (
	"log"

	"apptray/internal/desktop"
	"apptray/internal/ui"
)

func main() {
	entries, err := desktop.Scan(desktop.DefaultDirs())
	if err != nil {
		log.Fatal(err)
	}

	ui.Run(entries)
}
