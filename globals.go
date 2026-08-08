package main

import "time"

var (
	ROWS, COLS             int
	offsetRow, offsetCol   int
	currentRow, currentCol int
	buf                    *Buffer
	undoBuf                [][]rune
	copyBuf                []rune
	sourceFile             string
	mode                   Mode
	modified               bool
)

// chordTimeout bounds how long a leading key of a two-key chord (e.g. "gg")
// stays pending before it's treated as a fresh, unrelated keypress.
const chordTimeout = 500 * time.Millisecond

var (
	lastCh     rune
	lastChTime time.Time
)

// CharClass mirrors VIM's word/punct/space split: a run of same-class
// characters is one "word" for w/b purposes, so e.g. `"foo` is two words
// (the quote, then foo) rather than one.
type CharClass int

const (
	ClassSpace CharClass = iota
	ClassWord
	ClassPunct
)

type Mode int

const (
	ReadMode Mode = iota
	EditMode
)

var lastKeyPressed rune
