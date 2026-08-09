package main

import (
	"os"
	"time"

	"github.com/nsf/termbox-go"
)

func processKeyPress() {
	keyEvent := getKey()
	lineLen := buf.RuneLen(currentRow)

	switch {
	case keyEvent.Key == termbox.KeyEsc:
		esc()
	case keyEvent.Ch != 0:
		handleCharKey(keyEvent, lineLen)
	default:
		handleSpecialKey(keyEvent, lineLen)
	}

	lastKeyPressed = keyEvent.Ch
}

func handleCharKey(keyEvent termbox.Event, lineLen int) {
	switch mode {
	case EditMode:
		insertRune(keyEvent)
		modified = true
	case ReadMode:
		handleReadModeChar(keyEvent)
		if currentCol > lineLen {
			currentCol = lineLen
		}
	}
}

// readModeActions dispatches the single-key vim motions; 'g' is handled
// separately below since it can start a "gg" chord.
var readModeActions = map[rune]func(){
	'G': goToBottom,
	'I': goToStartOfLine,
	'A': goToEndOfLine,
	'a': editAfterWord,
	'h': left,
	'j': down,
	'k': up,
	'l': right,
	'w': nextWord,
	'b': prevWord,
	'e': endOfWord,
	'q': closeEditor,
	'i': editBeforeWord,
}

func handleReadModeChar(keyEvent termbox.Event) {
	// NOTE: VIM motions for scrolling
	if keyEvent.Ch == 'g' {
		if lastCh == 'g' && time.Since(lastChTime) < chordTimeout {
			// second 'g' of "gg" arrived in time - jump to the top
			goToTop()
			lastCh = 0
			return
		}
		lastChTime = time.Now()
	} else if action, ok := readModeActions[keyEvent.Ch]; ok {
		action()
	}
	lastCh = keyEvent.Ch
}

var specialKeyActions = map[termbox.Key]func(){
	termbox.KeyArrowUp:    up,
	termbox.KeyCtrlU:      pageUp,
	termbox.KeyArrowDown:  down,
	termbox.KeyCtrlD:      pageDown, // this is a vim motion actually, but it far easier to handle it like this
	termbox.KeyArrowLeft:  left,
	termbox.KeyArrowRight: right,
	termbox.KeyPgup:       pageUp,
	termbox.KeyPgdn:       pageDown,
}

func handleSpecialKey(keyEvent termbox.Event, lineLen int) {
	lastCh = 0 // any special key cancels a pending "g"

	switch keyEvent.Key {
	case termbox.KeyTab:
		insertRuneNTimes(keyEvent, 4)
	case termbox.KeySpace:
		insertRuneNTimes(keyEvent, 1)
	case termbox.KeyHome:
		currentCol = 0
	case termbox.KeyEnd:
		currentCol = lineLen
	default:
		if action, ok := specialKeyActions[keyEvent.Key]; ok {
			action()
		}
	}

	if currentCol > lineLen {
		currentCol = lineLen
	}
}

func insertRuneNTimes(keyEvent termbox.Event, n int) {
	if mode != EditMode {
		return
	}
	for range n {
		insertRune(keyEvent)
	}
	modified = true
}

func esc() {
	mode = ReadMode
	if currentCol > 0 && (lastKeyPressed == 'i' || lastKeyPressed == 'a') {
		currentCol--
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
	currentCol++
	mode = EditMode
	setCursorShape(CursorBlinkingBar)
}

func editBeforeWord() {
	mode = EditMode
	setCursorShape(CursorBlinkingBar)
}

func closeEditor() {
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
