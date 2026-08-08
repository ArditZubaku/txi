package main

import (
	"os"
	"time"

	"github.com/nsf/termbox-go"
)

func processKeyPress() {
	keyEvent := getKey()
	lineLen := buf.RuneLen(currentRow)

	if keyEvent.Key == termbox.KeyEsc {
		esc()
	} else {
		if keyEvent.Ch != 0 {
			// handle characters
			switch mode {
			case EditMode:
				insertRune(keyEvent)
				modified = true
			case ReadMode:
				// NOTE: VIM motions for scrolling
				if keyEvent.Ch == 103 && lastCh == 103 && time.Since(lastChTime) < chordTimeout {
					// second 'g' of "gg" arrived in time - jump to the top
					goToTop()
					lastCh = 0
				} else {
					switch keyEvent.Ch {
					case 'g':
						lastChTime = time.Now()
					case 'G':
						goToBottom()
					case 'I':
						goToStartOfLine()
					case 'A':
						goToEndOfLine()
					case 'a':
						editAfterWord()
					case 'h':
						left()
					case 'j':
						down()
					case 'k':
						up()
					case 'l':
						right()
					case 'w':
						nextWord()
					case 'b':
						prevWord()
					case 'e':
						endOfWord()
					case 'q':
						close()
					case 'i':
						editBeforeWord()
					}
					lastCh = keyEvent.Ch
				}

				if currentCol > lineLen {
					currentCol = lineLen
				}
			}
		} else {
			lastCh = 0 // any special key cancels a pending "g"

			switch keyEvent.Key {
			case termbox.KeyTab:
				if mode == EditMode {
					for range 4 {
						insertRune(keyEvent)
					}
					modified = true
				}
			case termbox.KeySpace:
				if mode == EditMode {
					insertRune(keyEvent)
					modified = true
				}
			case termbox.KeyArrowUp:
				up()
			case termbox.KeyCtrlU:
				pageUp()
			case termbox.KeyArrowDown:
				down()
			case termbox.KeyCtrlD:
				pageDown() // this is a vim motion actually, but it far easier to handle it like this
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

	lastKeyPressed = keyEvent.Ch
}

func esc() {
	mode = ReadMode
	if currentCol > 0 && (lastKeyPressed == 'i' || lastKeyPressed == 'a') {
		currentCol = currentCol - 1
	}
	setCursorShape(CursorDefault)
}

func up() {
	if currentRow != 0 {
		currentRow--
	}
}

func down() {
	if currentRow < buf.LineCount()-1 {
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
		currentCol = buf.RuneLen(currentRow)
	}
}

func right() {
	lineLen := buf.RuneLen(currentRow)
	if currentCol < lineLen {
		currentCol++
	} else if currentRow < buf.LineCount()-1 {
		// if we are not on the last line
		// move to the first column of a the new line
		currentRow++
		currentCol = 0
	}
}

func pageUp() {
	if (currentRow - ROWS/2) > 0 {
		currentRow -= ROWS / 2
	} else {
		currentRow = 0
	}
}

func pageDown() {
	if (currentRow + ROWS/2) < buf.LineCount()-1 {
		currentRow += ROWS / 2
	} else {
		currentRow = buf.LineCount() - 1
	}
}

func goToTop() {
	currentRow = 0
	currentCol = 0
}

func goToBottom() {
	currentRow = buf.LineCount() - 1
	currentCol = 0
}

func goToEndOfLine() {
	currentCol = buf.RuneLen(currentRow)
	mode = EditMode
	setCursorShape(CursorBlinkingBar)
}

func goToStartOfLine() {
	currentCol = 0
	mode = EditMode
	setCursorShape(CursorBlinkingBar)
}

func editAfterWord() {
	currentCol = currentCol + 1
	mode = EditMode
	setCursorShape(CursorBlinkingBar)
}

func editBeforeWord() {
	mode = EditMode
	setCursorShape(CursorBlinkingBar)
}

func close() {
	buf.Close()
	termbox.Close()
	os.Exit(0)
}

func isSpace(ch rune) bool {
	return ch == ' ' || ch == '\t'
}

func isWordChar(ch rune) bool {
	return ch == '_' ||
		(ch >= 'a' && ch <= 'z') ||
		(ch >= 'A' && ch <= 'Z') ||
		(ch >= '0' && ch <= '9')
}

func classOf(ch rune) CharClass {
	switch {
	case isSpace(ch):
		return ClassSpace
	case isWordChar(ch):
		return ClassWord
	default:
		return ClassPunct
	}
}

// charAt treats past-end-of-line as whitespace, so word motions see line
// breaks as word boundaries without special-casing them separately.
func charAt(row, col int) rune {
	ch, ok := buf.Rune(row, col)
	if !ok {
		return ' '
	}
	return ch
}

func stepForward(row, col int) (int, int) {
	if col < buf.RuneLen(row) {
		return row, col + 1
	}
	if row < buf.LineCount()-1 {
		return row + 1, 0
	}
	return row, col
}

func nextWord() {
	row, col := currentRow, currentCol
	startClass := classOf(charAt(row, col))

	if startClass != ClassSpace {
		for classOf(charAt(row, col)) == startClass {
			nr, nc := stepForward(row, col)
			if nr == row && nc == col {
				break
			}
			row, col = nr, nc
		}
	}

	for classOf(charAt(row, col)) == ClassSpace {
		nr, nc := stepForward(row, col)
		if nr == row && nc == col {
			break
		}
		row, col = nr, nc
	}

	currentRow, currentCol = row, col
}

func endOfWord() {
	row, col := currentRow, currentCol

	// always advance at least one position, so pressing 'e' at the end of
	// a word moves to the end of the next one instead of staying put
	row, col = stepForward(row, col)

	for classOf(charAt(row, col)) == ClassSpace {
		nr, nc := stepForward(row, col)
		if nr == row && nc == col {
			break
		}
		row, col = nr, nc
	}

	wordClass := classOf(charAt(row, col))
	for {
		nr, nc := stepForward(row, col)
		if (nr == row && nc == col) || classOf(charAt(nr, nc)) != wordClass {
			break
		}
		row, col = nr, nc
	}

	currentRow, currentCol = row, col
}

func stepBackward(row, col int) (int, int) {
	if col > 0 {
		return row, col - 1
	}
	if row > 0 {
		return row - 1, buf.RuneLen(row - 1)
	}
	return row, col
}

func prevWord() {
	row, col := currentRow, currentCol

	// step off the current word first, so pressing 'b' from a word-start
	// lands on the previous word instead of itself
	row, col = stepBackward(row, col)

	for classOf(charAt(row, col)) == ClassSpace {
		nr, nc := stepBackward(row, col)
		if nr == row && nc == col {
			break
		}
		row, col = nr, nc
	}

	wordClass := classOf(charAt(row, col))
	for {
		pr, pc := stepBackward(row, col)
		if (pr == row && pc == col) || classOf(charAt(pr, pc)) != wordClass {
			break
		}
		row, col = pr, pc
	}

	currentRow, currentCol = row, col
}
