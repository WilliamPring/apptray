package main

import (
	"flag"
	"log"

	"apptray/internal/config"
	"apptray/internal/desktop"
	"apptray/internal/launcher"
	"apptray/internal/ui"
)

func main() {
	mode := flag.String("mode", "search", "start in: search, categories or power")
	flag.Parse()

	// A typo'd -mode silently opening Search is worse than an error.
	tab, ok := launcher.ParseTab(*mode)
	if !ok {
		log.Fatalf("unknown -mode %q: want search, categories or power", *mode)
	}

	// A missing config file is not an error: it means defaults.
	cfg, err := config.Load(config.DefaultPath())
	if err != nil {
		log.Fatal(err)
	}

	entries, err := desktop.Scan(desktop.DefaultDirs())
	if err != nil {
		log.Fatal(err)
	}

	ui.Run(entries, cfg, tab)
}
