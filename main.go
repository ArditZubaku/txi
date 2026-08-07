package main

import (
	"bufio"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/mattn/go-runewidth"
	"github.com/nsf/termbox-go"
)

var (
	ROWS, COLS             int
	offsetRow, offsetCol   int
	currentRow, currentCol int
	textBuf                [][]rune
	undoBuf                [][]rune
	copyBuf                []rune
	sourceFile             string
	mode                   int
	modified               bool
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

		scrollTextBuffer()
		displayTextBuffer()
		displayStatusBar()
		termbox.SetCursor(currentCol-offsetCol, currentRow-offsetRow)

		if err := termbox.Flush(); err != nil {
			slog.Error("Could not show message", "error", err)
			os.Exit(1) // TODO: Will think of something better in such a case
		}

		processKeyPress()
	}
}

func displayTextBuffer() {
	bufLen := len(textBuf)

	for row := 0; row < ROWS; row++ {
		textBufRow := row + offsetRow

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
			textBufCol := col + offsetCol
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

func displayStatusBar() {
	var modeStatus, copyStatus, undoStatus, fileStatus, cursorStatus string

	if mode > 0 {
		modeStatus = " EDIT: "
	} else {
		modeStatus = " VIEW: "
	}

	// truncate the file name
	fileNameLen := min(len(sourceFile), 8)

	status := "saved"
	if modified {
		status = "modified"
	}
	fileStatus = fmt.Sprintf("%s - %d lines %s", sourceFile[:fileNameLen], len(textBuf), status)

	cursorStatus = fmt.Sprintf("Row %s, Col %s ", strconv.Itoa(currentRow+1), strconv.Itoa(currentCol+1))

	if len(copyBuf) > 0 {
		copyStatus = " [Copy]"
	}

	if len(undoBuf) > 0 {
		undoStatus = " [Undo]"
	}

	usedSpace := len(modeStatus) + len(fileStatus) + len(cursorStatus) + len(copyStatus) + len(undoStatus)
	spaces := strings.Repeat(" ", COLS-usedSpace)
	txt := modeStatus + fileStatus + copyStatus + undoStatus + spaces + cursorStatus

	printMessage(0, ROWS, termbox.ColorBlack, termbox.ColorWhite, txt)
}

func printMessage(col, row int, fg, bg termbox.Attribute, msg string) {
	for _, ch := range msg {
		termbox.SetCell(col, row, ch, fg, bg)
		col += runewidth.RuneWidth(ch)
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

func scrollTextBuffer() {
	if currentRow < offsetRow {
		offsetRow = currentRow
	}

	if currentCol < offsetCol {
		offsetCol = currentCol
	}

	if currentRow >= offsetRow+ROWS {
		offsetRow = currentRow - ROWS + 1
	}

	if currentCol >= offsetCol+COLS {
		offsetCol = currentCol - COLS + 1
	}
}

func getKey() termbox.Event {
	var keyEvent termbox.Event

	switch event := termbox.PollEvent(); event.Type {
	case termbox.EventKey:
		keyEvent = event
	case termbox.EventError:
		panic(event.Err) // TODO: Will think of something better in such a case
	}

	return keyEvent
}
