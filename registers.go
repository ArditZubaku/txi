package main

import "slices"

// clipboard is VIM's unnamed register: whatever was last yanked or deleted,
// held either as whole lines (yy, dd) or as a run within one line (yw, x), the
// only two shapes the operators can produce.
type register struct {
	lines    [][]rune
	linewise bool
}

var clipboard register

func (r register) empty() bool {
	return len(r.lines) == 0
}

func yankLine() {
	n := min(count(), buf.LineCount()-currentRow)

	lines := make([][]rune, 0, n)
	for i := range n {
		lines = append(lines, slices.Clone(buf.Line(currentRow+i)))
	}
	clipboard = register{lines: lines, linewise: true}
}

func yankChars(row, from, to int) {
	clipboard = register{lines: [][]rune{slices.Clone(buf.Line(row)[from:to])}}
}

// yankWord is 'yw', yankToWordEnd is 'ye' and yankToPrevWord is 'yb'. Like the
// delete operators they stop at the end of the line, and like VIM they leave
// the cursor at the start of what was yanked.
func yankWord() {
	row, col := nextWordFrom(currentRow, currentCol)
	yankForwardTo(row, col)
}

func yankToWordEnd() {
	row, col := endOfWordFrom(currentRow, currentCol)
	yankForwardTo(row, col+1)
}

func yankForwardTo(row, col int) {
	if row != currentRow {
		col = buf.RuneLen(currentRow)
	}
	if col <= currentCol {
		return
	}
	yankChars(currentRow, currentCol, col)
}

func yankToPrevWord() {
	row, col := prevWordFrom(currentRow, currentCol)
	if row != currentRow {
		col = 0
	}
	if col >= currentCol {
		return
	}

	yankChars(currentRow, col, currentCol)
	currentCol = col
}

func pasteAfter()  { paste(true) }
func pasteBefore() { paste(false) }

func paste(after bool) {
	if clipboard.empty() {
		return
	}

	if clipboard.linewise {
		pasteLines(after)
		return
	}
	pasteChars(after)
}

func pasteLines(after bool) {
	row := currentRow
	if after {
		row++
	}

	at := row
	for range count() {
		for _, line := range clipboard.lines {
			touchInsertLine(at)
			buf.InsertLine(at)
			buf.SetLine(at, slices.Clone(line))
			at++
		}
	}

	currentRow, currentCol = row, 0
	modified = true
}

func pasteChars(after bool) {
	text := clipboard.lines[0]
	line := slices.Clone(buf.Line(currentRow))

	col := currentCol
	if after && len(line) > 0 {
		col++
	}

	pasted := make([]rune, 0, len(text)*count())
	for range count() {
		pasted = append(pasted, text...)
	}

	touchLine(currentRow)
	buf.SetLine(currentRow, slices.Insert(line, col, pasted...))
	currentCol = col + len(pasted) - 1
	modified = true
}
