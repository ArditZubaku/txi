package main

import "fmt"

// minGutterWidth mirrors VIM's numberwidth default: the gutter never shrinks
// below four columns, however few lines the buffer has.
const minGutterWidth = 4

// gutterWidth is how many columns the line numbers take, the widest number the
// buffer can show plus the column separating it from the text.
func gutterWidth(lineCount int) int {
	digits := len(fmt.Sprintf("%d", max(lineCount, 1)))
	return max(digits+1, minGutterWidth)
}

// lineNumberLabel renders row's entry in the gutter the way VIM does with both
// `number` and `relativenumber` set: the cursor's own line shows its absolute
// number, left-aligned, and every other line its distance from the cursor,
// right-aligned against the separating column.
func lineNumberLabel(row, cursorRow, width int) string {
	if row == cursorRow {
		return fmt.Sprintf("%-*d", width, row+1)
	}

	distance := row - cursorRow
	if distance < 0 {
		distance = -distance
	}

	return fmt.Sprintf("%*d ", width-1, distance)
}
