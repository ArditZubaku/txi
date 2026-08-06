package main

import (
	"log/slog"
	"os"

	"github.com/mattn/go-runewidth"
	"github.com/nsf/termbox-go"
)

var (
	ROWS, COLS       int
	offsetX, offsetY int
	textBuf          [][]rune
)

func main() {
	textBuf = append(textBuf, []rune{'h', 'e', 'l', 'l', 'o'})
	textBuf = append(textBuf, []rune{'w', 'o', 'r', 'l', 'd'})

	runEditor()
}

func runEditor() {
	if err := termbox.Init(); err != nil {
		slog.Error("Could not init termbox", "error", err)
		os.Exit(1)
	}

	for {
		// Fetch current screen dimensions
		COLS, ROWS = termbox.Size()
		ROWS -= 1

		if COLS < 80 {
			COLS = 80
		}

		if err := termbox.Clear(termbox.ColorDefault, termbox.ColorDefault); err != nil {
			slog.Error("Could not clear terminal", "error", err)
			os.Exit(1) // TODO: Will think of something better in such a case
		}

		displayTextBuffer()

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

// nolint
func printMessage(col, row int, fg, bg termbox.Attribute, msg string) {
	for _, ch := range msg {
		termbox.SetCell(col, row, ch, fg, bg)
		col += runewidth.RuneWidth(ch)
	}
}

func displayTextBuffer() {
	bufLen := len(textBuf)

	for row := 0; row < ROWS; row++ {
		textBufRow := row + offsetY

		// Past end of buffer: draw line indicator once per row
		if textBufRow >= bufLen {
			termbox.SetCell(0, row, '*', termbox.ColorBlue, termbox.ColorDefault)
			continue
		}
		if textBufRow < 0 {
			continue
		}

		line := textBuf[textBufRow]
		lineLen := len(line)

		// Render visible characters in current row
		for col := 0; col < COLS; col++ {
			textBufCol := col + offsetX
			if textBufCol < 0 || textBufCol >= lineLen {
				continue
			}

			ch := line[textBufCol]
			if ch == '\t' {
				termbox.SetCell(col, row, ' ', termbox.ColorDefault, termbox.ColorGreen)
			} else {
				termbox.SetChar(col, row, ch)
			}
		}
	}
}
