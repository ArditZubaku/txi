package main

import (
	"log/slog"
	"os"

	"github.com/mattn/go-runewidth"
	"github.com/nsf/termbox-go"
)

func main() {
	runEditor()
}

func runEditor() {
	if err := termbox.Init(); err != nil {
		slog.Error("Could not init termbox", "error", err)
		os.Exit(1)
	}

	msg := "TXI - A bare bones text editor"

	// Fetch current screen dimensions
	width, height := termbox.Size()

	// Calculate center coordinates
	x := (width - len(msg)) / 2
	y := height / 2

	for {
		printMessage(
			x,
			y,
			termbox.ColorDefault,
			termbox.ColorDefault,
			msg,
		)
		if err := termbox.Flush(); err != nil {
			slog.Error("Could not show message", "error", err)
			os.Exit(1) // TODO: Will think of something better in such a case
		}

		e := termbox.PollEvent()
		if e.Type == termbox.EventKey && e.Key == termbox.KeyEsc {
			termbox.Close()
			break
		}
	}
}

func printMessage(col, row int, fg, bg termbox.Attribute, msg string) {
	for _, ch := range msg {
		termbox.SetCell(col, row, ch, fg, bg)
		col += runewidth.RuneWidth(ch)
	}
}
