package main

import (
	"log/slog"

	"github.com/nsf/termbox-go"
)

func insertRune(event termbox.Event) {
	ch := event.Ch
	switch event.Key {
	case termbox.KeySpace, termbox.KeyTab:
		ch = ' '
	}

	buf.InsertRune(currentRow, currentCol, ch)
	currentCol++
	modified = true
}

// deleteRune - drop the character under the cursor.
func deleteRune() {
	if currentCol >= buf.RuneLen(currentRow) {
		return
	}
	buf.DeleteRunes(currentRow, currentCol, currentCol+1)
	modified = true
}

// deleteWord is 'dw' and deleteToWordEnd is 'de'. Both stop at the end of the
// line even when the motion itself would carry on to the next one, which is
// what VIM does: an operator never eats the line break.
func deleteWord() {
	row, col := nextWordFrom(currentRow, currentCol)
	deleteTo(row, col)
}

func deleteToWordEnd() {
	row, col := endOfWordFrom(currentRow, currentCol)
	deleteTo(row, col+1)
}

// deleteToPrevWord is 'db': unlike dw/de it deletes behind the cursor, so the
// cursor follows the text back.
func deleteToPrevWord() {
	row, col := prevWordFrom(currentRow, currentCol)
	if row != currentRow {
		col = 0
	}
	if col >= currentCol {
		return
	}

	buf.DeleteRunes(currentRow, col, currentCol)
	currentCol = col
	modified = true
}

func deleteTo(row, col int) {
	if row != currentRow {
		col = buf.RuneLen(currentRow)
	}
	if col <= currentCol {
		return
	}

	buf.DeleteRunes(currentRow, currentCol, col)
	modified = true
}

// backspace deletes behind the cursor in Edit mode, joining onto the line
// above when there is nothing left to delete on this one. In Read mode it is
// VIM's plain leftwards move.
func backspace() {
	if mode != EditMode {
		left()
		return
	}

	switch {
	case currentCol > 0:
		currentCol--
		buf.DeleteRunes(currentRow, currentCol, currentCol+1)
	case currentRow > 0:
		currentRow--
		currentCol = buf.RuneLen(currentRow)
		buf.JoinLine(currentRow)
	default:
		return
	}

	modified = true
}

// saveFile is Ctrl-S, in either mode. It leaves the buffer marked modified if
// the write failed, so the status bar keeps saying so.
func saveFile() {
	if err := buf.Save(sourceFile); err != nil {
		slog.Error("Failed to save file", "path", sourceFile, "error", err)
		return
	}
	modified = false
}

func deleteLine() {
	buf.DeleteLine(currentRow)
	if currentRow >= buf.LineCount() {
		currentRow = buf.LineCount() - 1
	}
	modified = true
}
