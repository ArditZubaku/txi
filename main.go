package main

import (
	"bufio"
	"errors"
	"log/slog"
	"os"

	"github.com/nsf/termbox-go"
)

var (
	ROWS, COLS       int
	offsetX, offsetY int
	textBuf          [][]rune
	sourceFile       string
)

func main() {
	runEditor()
}

func runEditor() {
	if err := termbox.Init(); err != nil {
		slog.Error("Could not init termbox", "error", err)
		os.Exit(1)
	}

	if len(os.Args) > 1 {
		sourceFile = os.Args[1]
		readFile(sourceFile)
	} else {
		sourceFile = "out.txt"
		// So we have a new line at the end of the file
		textBuf = append(textBuf, []rune{})
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
		if e.Type == termbox.EventKey && (e.Key == termbox.KeyEsc || e.Key == termbox.KeyCtrlQ) {
			termbox.Close()
			break
		}
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

func readFile(name string) {
	file, err := os.Open(name)
	if err != nil {
		sourceFile = name
		textBuf = append(textBuf, []rune{})
		return
	}

	defer func() {
		if err := file.Close(); err != nil && !errors.Is(err, os.ErrClosed) {
			slog.Error("Failed to close file", "path", file.Name(), "error", err)
		}
	}()

	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		line := scanner.Text()
		textBuf = append(textBuf, []rune{})

		for i := 0; i < len(line); i++ {
			textBuf[lineNum] = append(textBuf[lineNum], rune(line[i]))
		}

		lineNum++
	}

	if err := scanner.Err(); err != nil {
		slog.Error("Failed to scan the file", "file", file.Name(), "error", err)
		// TODO: Show the user something about this
		return
	}

	if lineNum == 0 {
		textBuf = append(textBuf, []rune{})
	}
}
