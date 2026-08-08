package main

import (
	"fmt"
	"os"

	"github.com/nsf/termbox-go"
)

func processKeyPress() {
	keyEvent := getKey()
	lineLen := len(textBuf[currentRow])

	if keyEvent.Key == termbox.KeyEsc {
		termbox.Close()
		os.Exit(0)
	} else if keyEvent.Ch != 0 {
		// handle characters
		fmt.Printf("here %+v", keyEvent)

		// NOTE: VIM motions for scrolling
		switch keyEvent.Ch {
		case 104: // h <left>
			left()
		case 106: // j <down>
			down()
		case 107: // k <up>
			up()
		case 108: // l <right>
			right()
		case 117: // u <up a page>
			pageUp()
		case 100: // d <down a page>
			pageDown()
		}

		if currentCol > lineLen {
			currentCol = lineLen
		}
	} else {
		switch keyEvent.Key {
		case termbox.KeyArrowUp:
			up()
		case termbox.KeyArrowDown:
			down()
		case termbox.KeyArrowLeft:
			left()
		case termbox.KeyArrowRight:
			right()
		case termbox.KeyHome:
			currentCol = 0
		case termbox.KeyEnd:
			currentCol = lineLen
		case termbox.KeyPgup:
			pageUp()
		case termbox.KeyPgdn:
			pageDown()
		}

		if currentCol > lineLen {
			currentCol = lineLen
		}
	}
}

func up() {
	if currentRow != 0 {
		currentRow--
	}
}

func down() {
	if currentRow < len(textBuf)-1 {
		currentRow++
	}
}

func left() {
	if currentCol != 0 {
		currentCol--
	} else if currentRow > 0 {
		// if we are not on the first line
		// move to the end of the previous line
		currentRow--
		currentCol = len(textBuf[currentRow])
	}
}

func right() {
	lineLen := len(textBuf[currentRow])
	if currentCol < lineLen {
		currentCol++
	} else if currentRow < len(textBuf)-1 {
		// if we are not on the last line
		// move to the first column of a the new line
		currentRow++
		currentCol = 0
	}
}

func pageUp() {
	if (currentRow - ROWS/4) > 0 {
		currentRow -= ROWS / 4
	}
}

func pageDown() {
	if (currentRow + ROWS/4) < len(textBuf)-1 {
		currentRow += ROWS / 4
	}
}
