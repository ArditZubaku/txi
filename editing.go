package main

import "github.com/nsf/termbox-go"

func insertRune(event termbox.Event) {
	currentLine := buf.Line(currentRow)

	// currentLine may alias the shared read window,
	// so build a fresh slice rather than inserting in place.
	updated := make([]rune, len(currentLine)+1)
	copy(updated[:currentCol], currentLine[:currentCol])

	switch event.Key {
	case termbox.KeySpace, termbox.KeyTab:
		updated[currentCol] = ' '
	default:
		updated[currentCol] = event.Ch
	}

	copy(updated[currentCol+1:], currentLine[currentCol:])
	buf.SetLine(currentRow, updated)
	currentCol++
}
