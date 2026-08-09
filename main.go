package main

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"

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

	if len(os.Args) > 1 {
		sourceFile = os.Args[1]
		buf = openBuffer(sourceFile)
	} else {
		sourceFile = "out.txt"
		buf = newEmptyBuffer()
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
		termbox.SetCursor(currentCol-offsetCol+gutterWidth(buf.LineCount()), currentRow-offsetRow)

		if err := termbox.Flush(); err != nil {
			slog.Error("Could not show message", "error", err)
			os.Exit(1) // TODO: Will think of something better in such a case
		}

		processKeyPress()
	}
}

const cursorLineBg = termbox.ColorDarkGray

// Characters drawn over the band with SetChar keep the colours painted here.
func highlightRow(row int) {
	for col := 0; col < COLS; col++ {
		termbox.SetCell(col, row, ' ', termbox.ColorDefault, cursorLineBg)
	}
}

func displayTextBuffer() {
	bufLen := buf.LineCount()
	gutter := gutterWidth(bufLen)
	textCols := COLS - gutter

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

		numberColor, background := termbox.ColorBlue, termbox.ColorDefault
		if textBufRow == currentRow {
			numberColor, background = termbox.ColorYellow, cursorLineBg
			highlightRow(row)
		}
		printMessage(0, row, numberColor, background, lineNumberLabel(textBufRow, currentRow, gutter))

		line := buf.Line(textBufRow)
		lineLen := len(line)

		// Render visible characters in current row
		for col := 0; col < textCols; col++ {
			textBufCol := col + offsetCol
			if textBufCol < 0 || textBufCol >= lineLen {
				continue
			}

			ch := line[textBufCol]
			if ch == '\t' {
				ch = ' '
			}
			termbox.SetChar(gutter+col, row, ch)
		}
	}
}

func displayStatusBar() {
	var modeStatus, copyStatus, undoStatus, redoStatus, countStatus, fileStatus, cursorStatus string

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
	fileStatus = fmt.Sprintf("%s - %d lines %s", sourceFile[:fileNameLen], buf.LineCount(), status)

	cursorStatus = fmt.Sprintf("Row %s, Col %s ", strconv.Itoa(currentRow+1), strconv.Itoa(currentCol+1))

	if !clipboard.empty() {
		copyStatus = " [Copy]"
	}

	if len(undoStack) > 0 {
		undoStatus = " [Undo]"
	}

	if len(redoStack) > 0 {
		redoStatus = " [Redo]"
	}

	if pendingCount > 0 {
		countStatus = strconv.Itoa(pendingCount) + " "
	}

	leftStatus := modeStatus + fileStatus + copyStatus + undoStatus + redoStatus
	rightStatus := countStatus + cursorStatus
	spaces := strings.Repeat(" ", max(COLS-len(leftStatus)-len(rightStatus), 0))
	txt := leftStatus + spaces + rightStatus

	printMessage(0, ROWS, termbox.ColorBlack, termbox.ColorWhite, txt)
}

func printMessage(col, row int, fg, bg termbox.Attribute, msg string) {
	for _, ch := range msg {
		termbox.SetCell(col, row, ch, fg, bg)
		col += runewidth.RuneWidth(ch)
	}
}

func scrollTextBuffer() {
	textCols := COLS - gutterWidth(buf.LineCount())

	if currentRow < offsetRow {
		offsetRow = currentRow
	}

	if currentCol < offsetCol {
		offsetCol = currentCol
	}

	if currentRow >= offsetRow+ROWS {
		offsetRow = currentRow - ROWS + 1
	}

	if currentCol >= offsetCol+textCols {
		offsetCol = currentCol - textCols + 1
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
