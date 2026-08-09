package main

import "github.com/nsf/termbox-go"

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

func deleteLine() {
	buf.DeleteLine(currentRow)
	if currentRow >= buf.LineCount() {
		currentRow = buf.LineCount() - 1
	}
	modified = true
}
