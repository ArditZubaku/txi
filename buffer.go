package main

import (
	"bytes"
	"errors"
	"io"
	"log/slog"
	"os"
	"slices"
	"unicode/utf8"
)

// Buffer reads a file from disk on demand, holding only an index of where each
// line starts plus one fixed-size window of raw bytes around the cursor.
type Buffer struct {
	file  *os.File
	size  int64
	count int

	starts          []int64 // byte offset of each line's first byte
	endsWithNewline bool

	// win holds raw bytes for lines [winFrom, winTo), starting at byte winBase.
	// Slices into it are invalidated by the next fill.
	win            []byte
	winBase        int64
	winFrom, winTo int
	cacheRow       int
	cacheLine      []rune
	cacheOK        bool

	overlay map[int][]rune // lines edited in Insert mode, shadowing the file
}

const (
	windowBytes = 64 << 10
	windowBack  = 32 // lines kept behind the cursor, so scrolling back up rarely re-reads
	scanChunk   = 1 << 20
)

func newEmptyBuffer() *Buffer {
	return &Buffer{
		count:   1,
		overlay: map[int][]rune{0: {}},
		winFrom: -1,
		winTo:   -1,
	}
}

func openBuffer(name string) *Buffer {
	file, err := os.Open(name)
	if err != nil {
		slog.Error("Failed to open file", "error", err)
		return newEmptyBuffer()
	}

	info, err := file.Stat()
	if err != nil {
		slog.Error("Failed to stat file", "path", name, "error", err)
		closeFile(file)
		return newEmptyBuffer()
	}

	starts := buildIndex(file, info.Size())
	if len(starts) == 0 {
		closeFile(file)
		return newEmptyBuffer()
	}

	last := make([]byte, 1)
	if _, err := file.ReadAt(last, info.Size()-1); err != nil {
		slog.Error("Failed to read final byte", "path", name, "error", err)
		closeFile(file)
		return newEmptyBuffer()
	}

	return &Buffer{
		file:            file,
		size:            info.Size(),
		count:           len(starts),
		starts:          starts,
		endsWithNewline: last[0] == '\n',
		win:             make([]byte, 0, windowBytes),
		winFrom:         -1,
		winTo:           -1,
		overlay:         make(map[int][]rune),
	}
}

func closeFile(file *os.File) {
	if err := file.Close(); err != nil && !errors.Is(err, os.ErrClosed) {
		slog.Error("Failed to close file", "path", file.Name(), "error", err)
	}
}

// buildIndex records the offset of every line start, reading through one
// reusable buffer so the scan costs a fixed scanChunk rather than the file size.
// Splitting matches bufio.Scanner: '\n' terminates, a trailing '\n' adds no
// empty line, and a '\r' before it is stripped at read time.
func buildIndex(r io.ReaderAt, size int64) []int64 {
	if size == 0 {
		return nil
	}

	const guessedLineLen = 96

	starts := make([]int64, 1, size/guessedLineLen+1)
	scratch := make([]byte, scanChunk)

	for off := int64(0); off < size; {
		n, err := r.ReadAt(scratch, off)
		for i := 0; i < n; {
			j := bytes.IndexByte(scratch[i:n], '\n')
			if j < 0 {
				break
			}
			i += j + 1
			if off+int64(i) < size {
				starts = append(starts, off+int64(i))
			}
		}

		if n == 0 {
			break
		}
		off += int64(n)

		if err != nil {
			if !errors.Is(err, io.EOF) {
				slog.Error("Failed to index file", "offset", off, "error", err)
			}
			break
		}
	}

	return starts
}

func (b *Buffer) Close() {
	if b == nil || b.file == nil {
		return
	}
	closeFile(b.file)
	b.file = nil
}

func (b *Buffer) LineCount() int {
	return b.count
}

// lineEnd is exclusive and omits the terminator.
func (b *Buffer) lineEnd(i int) int64 {
	if i+1 < b.count {
		return b.starts[i+1] - 1
	}
	if b.endsWithNewline {
		return b.size - 1
	}
	return b.size
}

func (b *Buffer) fillWindow(i int) {
	from := max(0, i-windowBack)
	base := b.starts[from]

	// The window must hold the requested line whole, however long that line is.
	length := min(int64(windowBytes), b.size-base)
	if need := b.lineEnd(i) - base; need > length {
		length = need
	}

	oversized := int64(cap(b.win)) > 4*windowBytes && length <= windowBytes
	if int64(cap(b.win)) < length || oversized {
		b.win = make([]byte, length)
	}
	b.win = b.win[:length]

	n, err := b.file.ReadAt(b.win, base)
	if err != nil && !errors.Is(err, io.EOF) {
		slog.Error("Failed to read line window", "offset", base, "error", err)
	}
	b.win = b.win[:n]

	b.winBase, b.winFrom = base, from
	b.winTo = from
	for b.winTo < b.count && b.lineEnd(b.winTo) <= base+int64(n) {
		b.winTo++
	}
}

// raw aliases the window: the next fill invalidates the result.
func (b *Buffer) raw(i int) []byte {
	if i < b.winFrom || i >= b.winTo {
		b.fillWindow(i)
		if i < b.winFrom || i >= b.winTo {
			return nil
		}
	}

	line := b.win[b.starts[i]-b.winBase : b.lineEnd(i)-b.winBase]
	if len(line) > 0 && line[len(line)-1] == '\r' {
		line = line[:len(line)-1]
	}

	return line
}

// Line 's result is owned by the Buffer; copy before modifying (see SetLine).
func (b *Buffer) Line(i int) []rune {
	if i < 0 || i >= b.count {
		return nil
	}
	if line, ok := b.overlay[i]; ok {
		return line
	}
	if b.cacheOK && b.cacheRow == i {
		return b.cacheLine
	}

	line := bytes.Runes(b.raw(i))
	b.cacheRow, b.cacheLine, b.cacheOK = i, line, true

	return line
}

// RuneLen avoids decoding: for unedited lines it counts runes over the raw bytes.
func (b *Buffer) RuneLen(i int) int {
	if i < 0 || i >= b.count {
		return 0
	}
	if line, ok := b.overlay[i]; ok {
		return len(line)
	}
	if b.cacheOK && b.cacheRow == i {
		return len(b.cacheLine)
	}

	return utf8.RuneCount(b.raw(i))
}

func (b *Buffer) Rune(i, col int) (rune, bool) {
	line := b.Line(i)
	if col < 0 || col >= len(line) {
		return 0, false
	}

	return line[col], true
}

func (b *Buffer) SetLine(i int, line []rune) {
	if i < 0 || i >= b.count {
		return
	}

	b.overlay[i] = line
	if b.cacheOK && b.cacheRow == i {
		b.cacheOK = false
	}
}

// InsertRune inserts ch at col in line i. The first insert into an unedited
// line has to copy it out of the read window, but from then on the line grows
// amortized in the overlay, so typing a run of characters into it does not
// reallocate on every keystroke.
func (b *Buffer) InsertRune(i, col int, ch rune) {
	line := b.Line(i)
	col = min(max(col, 0), len(line))

	if edited, ok := b.overlay[i]; ok {
		b.overlay[i] = slices.Insert(edited, col, ch)
		return
	}

	updated := make([]rune, len(line)+1)
	copy(updated, line[:col])
	updated[col] = ch
	copy(updated[col+1:], line[col:])
	b.SetLine(i, updated)
}

// DeleteRunes removes runes [from, to) from line i. An already-edited line is
// compacted in place; an unedited one is copied out at its new, shorter length,
// so a delete never allocates more than the line it shrinks.
func (b *Buffer) DeleteRunes(i, from, to int) {
	line := b.Line(i)
	from, to = max(from, 0), min(to, len(line))
	if from >= to {
		return
	}

	if edited, ok := b.overlay[i]; ok {
		b.overlay[i] = append(edited[:from], edited[to:]...)
		return
	}

	updated := make([]rune, len(line)-(to-from))
	copy(updated, line[:from])
	copy(updated[from:], line[to:])
	b.SetLine(i, updated)
}

// DeleteLine drops line i by compacting the index in place and re-keying the
// overlay, so nothing proportional to the rest of the file is rebuilt. The
// deleted bytes stay on disk, simply unreferenced by the index.
func (b *Buffer) DeleteLine(i int) {
	if i < 0 || i >= b.count {
		return
	}
	b.cacheOK = false

	// VIM leaves an empty line behind rather than an empty buffer
	if b.count == 1 {
		b.overlay[i] = b.overlay[i][:0]
		return
	}

	// A line ends where the next one starts, so dropping an index entry would
	// hand the deleted bytes to the line above it. Pin that line into the
	// overlay instead, where its content no longer depends on the index.
	if i > 0 {
		if _, ok := b.overlay[i-1]; !ok {
			b.overlay[i-1] = slices.Clone(b.Line(i - 1))
		}
	}

	b.starts = append(b.starts[:i], b.starts[i+1:]...)
	b.count--

	delete(b.overlay, i)
	if len(b.overlay) > 0 {
		shifted := make([]int, 0, len(b.overlay))
		for row := range b.overlay {
			if row > i {
				shifted = append(shifted, row)
			}
		}
		// ascending, so each line moves into a slot already vacated
		slices.Sort(shifted)
		for _, row := range shifted {
			b.overlay[row-1] = b.overlay[row]
			delete(b.overlay, row)
		}
	}

	// window line numbers no longer match the file after the shift
	b.winFrom, b.winTo = -1, -1
}
