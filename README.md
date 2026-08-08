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
- **Large-file-friendly loading** — `textBuf` is pre-sized from the file's byte length to avoid repeated reallocation, and the line scanner accepts lines beyond the default 64KB.

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
