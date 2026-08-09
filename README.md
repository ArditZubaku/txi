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
| `gg` | Normal | jump to the top of the buffer |
| `G` | Normal | jump to the bottom of the buffer |
| `I` | Normal | jump to start of line and enter Insert mode |
| `A` | Normal | jump to end of line and enter Insert mode |
| `i` | Normal | enter Insert mode before the cursor |
| `a` | Normal | enter Insert mode after the cursor |
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

A line wider than the window grows it for as long as that line is on screen, then it
shrinks back. Only the index scales with file size, at 8 bytes per line.

### Goals not yet implemented

- Saving to disk (there is currently no write/`:w` path — the "modified"/"saved" status only reflects in-memory state)
- `:` command mode (`:w`, `:q`, `:wq`, ...) — `q` quits directly, with no unsaved-changes check
- Visual mode
- Search (`/`, `?`, `n`, `N`)
- Operators and text objects (`d`, `c`, `x`, `dd`, `cw`, ...)
- Yank/paste (`y`, `p`) and registers
- Undo/redo (`u`, `Ctrl-R`) — `undoBuf` is scaffolded in `globals.go` and shown in the status bar as `[Undo]`, but nothing populates it yet. Same story for `copyBuf`/`[Copy]`.
- Marks and macros
- `o`/`O` (open a new line below/above)
