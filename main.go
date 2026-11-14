package main

import (
	"fmt"
	"log"

	"apptray/internal/desktop"
)

func main() {
	entries, err := desktop.Scan(desktop.DefaultDirs())
	if err != nil {
		log.Fatal(err)
	}

	for _, e := range entries {
		fmt.Println(e.Name)
	}
}
