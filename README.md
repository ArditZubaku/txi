# txi

A small terminal text editor written in Go, built on [termbox-go](https://github.com/nsf/termbox-go). It's modal like VIM (separate Normal/Read and Insert/Edit modes) and currently implements a subset of VIM's motions and editing keys.

## Usage

```sh
go build -o txi .
./txi path/to/file      # opens (or creates) a file
./txi                   # starts a new, unnamed buffer (out.txt)
```

## Features

- **Modal editing** — a Read (Normal) mode for navigation and an Edit (Insert) mode for typing, with the terminal cursor changing shape (block vs. blinking bar) depending on mode.
- **VIM-style navigation** — `hjkl`, word motions (`w`/`b`/`e`), line jumps (`I`/`A`), buffer jumps (`gg`/`G`); see the full list below.
- **Saving** — `Ctrl-S`, in either mode. The buffer is streamed to a temporary file in the same directory and renamed over the target, so a failed write cannot truncate the original; untouched lines are copied as raw bytes, so a save costs no more memory than scrolling does. File permissions, CRLF line endings on untouched lines, and a missing trailing newline are all preserved.
- **Deleting** — `x`, `dw`, `de`, `db` and `dd` in Normal mode, `Backspace` in Insert mode, all in place: a line delete compacts the line index rather than rebuilding it, and a character delete reuses the edited line's own backing array, so deleting never costs more memory than the text it removes.
- **Line structure** — `o`/`O` open a line, `Enter` splits one at the cursor and `Backspace` at column 0 joins it back. Each shifts the line index in place rather than rebuilding it, and a split copies only the tail: the half before the cursor keeps the array it already had.
- **Word-class-aware word motions** — `w`/`b`/`e` classify runs of characters into whitespace / word (`[A-Za-z0-9_]`) / punctuation, so e.g. `"foo` is treated as two words (`"` then `foo`), matching VIM's default word boundaries.
- **Viewport scrolling** — the visible window follows the cursor both vertically and horizontally as the buffer grows past the terminal size.
- **Status bar** — current mode, file name, line count, modified/saved state, and cursor row/column.
- **Constant-memory file loading** — the file is never held in memory. Opening it builds an index of where each line starts (8 bytes per line) and nothing else; lines are read through one fixed 64KB window and decoded to runes only when they're on screen or under the cursor. Opening a 23MB file of 202,000 lines and jumping to the end costs **8.8MB of RSS**, and that figure doesn't move however far you scroll — 5.2MB of it is the Go runtime floor a one-line file also pays, so the file itself accounts for 2.6MB. See [Memory model](#memory-model).

## VIM motions implemented

| Key | Mode | Action |
| ----- | ------ | -------- |
| `h` `j` `k` `l` | Normal | move left / down / up / right |
| `w` | Normal | jump to the start of the next word |
| `b` | Normal | jump to the start of the previous word |
| `e` | Normal | jump to the end of the (next) word |
| `x` | Normal | delete the character under the cursor |
| `dw` | Normal | delete to the start of the next word (stops at end of line) |
| `de` | Normal | delete to the end of the current word (stops at end of line) |
| `db` | Normal | delete back to the start of the previous word (stops at the start of the line) |
| `dd` | Normal | delete the current line |
| `Backspace` | Insert | delete the character before the cursor, joining onto the line above at column 0 |
| `Enter` | Insert | split the line at the cursor (in Normal mode it moves down a line) |
| `gg` | Normal | jump to the top of the buffer |
| `G` | Normal | jump to the bottom of the buffer |
| `I` | Normal | jump to start of line and enter Insert mode |
| `A` | Normal | jump to end of line and enter Insert mode |
| `o` | Normal | open an empty line below and enter Insert mode |
| `O` | Normal | open an empty line above and enter Insert mode |
| `i` | Normal | enter Insert mode before the cursor |
| `a` | Normal | enter Insert mode after the cursor |
| `Ctrl-S` | Either | save to the file that was opened |
| `Ctrl-U` | Either | scroll up half a screen |
| `Ctrl-D` | Either | scroll down half a screen |
| `Esc` | Insert | return to Normal mode (cursor steps back a column, VIM-style) |

Arrow keys, `Home`, `End`, `PgUp`, and `PgDn` also work in either mode.

## Memory model

The file is never loaded. It stays on disk with its handle open, and `buffer.go`
holds three things, none of which scale with how much of it you have visited:

| | held | 23MB / 202k-line file |
| --- | --- | --- |
| Line index (`starts`) | one byte offset per line | ~1.6MB |
| Read window (`win`) | raw bytes around the cursor | 64KB |
| Decoded line cache | the one line under the cursor | a few hundred bytes |

Opening a file makes a single indexing pass with `ReadAt` through one reusable 1MB
buffer, recording where each line begins. Reading through a small buffer rather than
walking an `mmap` is deliberate: faulting a mapping in to find its newlines makes
every page of the file resident, and macOS will not hand those pages back
(`MADV_FREE_REUSABLE` is `EPERM` on file mappings). A read leaves the data in the OS
page cache without charging it to the process.

Everything after that pass is random access through the index: `G` and `gg` are
O(1), the status bar's line count is exact, and displaying a line is one `pread`
into the window when the line falls outside it. Lines edited in Insert mode live in
an overlay map that shadows the file, so an edit is never lost when the window moves.
Lifting a line into that overlay copies it once; every keystroke after that grows or
shrinks it in place, so typing a word costs no allocation per character.

Opening a line with `o`/`O` adds an index entry that borrows the offset of the line
below it, which leaves every neighbouring line's extent exactly as it was; the new
line itself is only ever read from the overlay that shadows it.

Deleting a line drops its entry from the index and renumbers the overlay; the bytes
themselves stay on disk, unreferenced. Because a line ends where the next one starts,
the line *above* a deleted one would otherwise inherit its bytes, so that one line is
copied into the overlay — the only allocation a `dd` makes, and it is bounded by the
length of that single line rather than by the size of the file.

A line wider than the window grows it for as long as that line is on screen, then it
shrinks back. Only the index scales with file size, at 8 bytes per line.

### Goals not yet implemented

- `:` command mode (`:w`, `:q`, `:wq`, ...) — saving is `Ctrl-S`, and `q` quits directly, with no unsaved-changes check
- Saving under a different name (`:w other.txt`), and any error reporting beyond a log line if the save fails
- Visual mode
- Search (`/`, `?`, `n`, `N`)
- The rest of the operators and text objects (`c`, `cw`, `dj`, `di(`, ..., and counts like `3dd`) — only `x`, `dw`, `de`, `db` and `dd` exist so far, and they stop at the line boundary instead of running onto the next line
- Yank/paste (`y`, `p`) and registers
- Undo/redo (`u`, `Ctrl-R`) — `undoBuf` is scaffolded in `globals.go` and shown in the status bar as `[Undo]`, but nothing populates it yet. Same story for `copyBuf`/`[Copy]`.
- Marks and macros
