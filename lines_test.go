package main

import "testing"

func TestInsertLine(t *testing.T) {
	cases := []struct {
		name string
		at   int
		want []string
	}{
		{"at the top", 0, []string{"", "a", "bb", "ccc"}},
		{"in the middle", 1, []string{"a", "", "bb", "ccc"}},
		{"at the end", 3, []string{"a", "bb", "ccc", ""}},
		{"past the end", 4, []string{"a", "bb", "ccc"}},
		{"before the start", -1, []string{"a", "bb", "ccc"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b := openBuffer(writeTemp(t, "a\nbb\nccc\n"))
			defer b.Close()

			b.InsertLine(tc.at)
			wantLines(t, b, tc.want...)
		})
	}
}

// The inserted line borrows the offset of the line below it, so the lines
// either side of it must still read back from the file untouched.
func TestInsertLineKeepsNeighboursReadable(t *testing.T) {
	b := openBuffer(writeTemp(t, "a\nbb\nccc\ndddd\n"))
	defer b.Close()

	b.InsertLine(2)
	b.SetLine(2, []rune("NEW"))
	wantLines(t, b, "a", "bb", "NEW", "ccc", "dddd")

	// again, now that the index already holds a duplicated offset
	b.InsertLine(4)
	wantLines(t, b, "a", "bb", "NEW", "ccc", "", "dddd")
}

// A file with no trailing newline ends its last line at size rather than at a
// terminator, so appending a line must not eat that line's final character.
func TestInsertLineAtTheEndOfAnUnterminatedFile(t *testing.T) {
	for _, tc := range []struct{ name, content string }{
		{"terminated", "a\nbb\n"},
		{"unterminated", "a\nbb"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := openBuffer(writeTemp(t, tc.content))
			defer b.Close()

			b.InsertLine(2)
			wantLines(t, b, "a", "bb", "")
		})
	}
}

func TestInsertLineShiftsOverlay(t *testing.T) {
	b := openBuffer(writeTemp(t, "a\nb\nc\nd\n"))
	defer b.Close()

	b.SetLine(1, []rune("B"))
	b.SetLine(3, []rune("D"))

	b.InsertLine(1)
	wantLines(t, b, "a", "", "B", "c", "D")
}

func TestInsertLineOnAnEmptyBuffer(t *testing.T) {
	b := newEmptyBuffer()

	b.InsertLine(1)
	b.InsertRune(1, 0, 'x')
	wantLines(t, b, "", "x")

	b.DeleteLine(0)
	wantLines(t, b, "x")
}

func TestInsertLineSaves(t *testing.T) {
	path := writeTemp(t, "a\nbb\n")
	b := openBuffer(path)
	defer b.Close()

	b.InsertLine(1)
	b.InsertRune(1, 0, 'X')
	if err := b.Save(path); err != nil {
		t.Fatal(err)
	}

	if got := readBack(t, path); got != "a\nX\nbb\n" {
		t.Fatalf("file = %q, want %q", got, "a\nX\nbb\n")
	}
	wantLines(t, b, "a", "X", "bb")
}

func TestInsertLineAcrossBigFile(t *testing.T) {
	path := bigFile(t, 5000)
	want := fullDecode(t, path)

	b := openBuffer(path)
	defer b.Close()

	// 41 is longer than the window, 97 is empty
	for _, row := range []int{4999, 2345, 98, 42, 41, 1, 0} {
		b.InsertLine(row)
		want = append(want[:row], append([][]rune{{}}, want[row:]...)...)
	}

	if b.LineCount() != len(want) {
		t.Fatalf("LineCount = %d, want %d", b.LineCount(), len(want))
	}
	for i := range want {
		if got := string(b.Line(i)); got != string(want[i]) {
			t.Fatalf("Line(%d) = %.40q, want %.40q", i, got, string(want[i]))
		}
	}
}

func TestOpenLineOperators(t *testing.T) {
	t.Run("o opens below", func(t *testing.T) {
		b := atCursor(t, "a\nbb\n", 0, 1)
		mode = ReadMode

		openLineBelow()
		wantLines(t, b, "a", "", "bb")
		if currentRow != 1 || currentCol != 0 || mode != EditMode || !modified {
			t.Errorf("cursor at %d,%d, mode %v, modified %v", currentRow, currentCol, mode, modified)
		}
	})

	t.Run("O opens above", func(t *testing.T) {
		b := atCursor(t, "a\nbb\n", 1, 2)
		mode = ReadMode

		openLineAbove()
		wantLines(t, b, "a", "", "bb")
		if currentRow != 1 || currentCol != 0 || mode != EditMode {
			t.Errorf("cursor at %d,%d, mode %v", currentRow, currentCol, mode)
		}
	})

	t.Run("o on the last line", func(t *testing.T) {
		b := atCursor(t, "a\nbb\n", 1, 0)
		mode = ReadMode

		openLineBelow()
		wantLines(t, b, "a", "bb", "")
		if currentRow != 2 {
			t.Errorf("currentRow = %d, want 2", currentRow)
		}
	})
}
