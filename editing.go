package main

import "github.com/nsf/termbox-go"

func insertRune(event termbox.Event) {
	// TODO: improve this, unneccessary doubling of memory, we could just maneuver with the textBuf directly
	currentLine := textBuf[currentRow]
	lineLen := len(currentLine)

	runesToInsert := make([]rune, lineLen+1)
	copy(runesToInsert[:currentCol], currentLine[:currentCol])

	switch event.Key {
	case termbox.KeySpace:
		runesToInsert[currentCol] = ' '
	case termbox.KeyTab:
		runesToInsert[currentCol] = ' '
	default:
		runesToInsert[currentCol] = event.Ch
	}

	copy(runesToInsert[currentCol+1:], textBuf[currentRow][currentCol:])
	textBuf[currentRow] = runesToInsert
	currentCol++
}
