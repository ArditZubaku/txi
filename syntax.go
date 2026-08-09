package main

import (
	"path/filepath"
	"strings"

	"github.com/nsf/termbox-go"
)

// VIM's default palette, which keeps every token readable over the cursor
// line's dark gray band as well as over the terminal's own background.
const (
	colorKeyword = termbox.ColorYellow
	colorType    = termbox.ColorGreen
	colorString  = termbox.ColorMagenta
	colorNumber  = termbox.ColorCyan
	colorComment = termbox.ColorLightBlue
	colorPlain   = termbox.ColorDefault
)

// Syntax is one language's lexical surface: enough to tell comments, strings,
// numbers and reserved words apart, and nothing more. Every delimiter is
// ASCII, so a rune of a line matches a byte of a delimiter one for one.
type Syntax struct {
	lineComment string
	blockStart  string
	blockEnd    string
	quotes      string
	keywords    map[string]bool
	types       map[string]bool
}

func words(list string) map[string]bool {
	set := make(map[string]bool)
	for _, w := range strings.Fields(list) {
		set[w] = true
	}

	return set
}

var goSyntax = &Syntax{
	lineComment: "//",
	blockStart:  "/*",
	blockEnd:    "*/",
	quotes:      "\"'`",
	keywords: words(`break case chan const continue default defer else fallthrough for
		func go goto if import interface map package range return select struct switch type var`),
	types: words(`any bool byte comparable complex64 complex128 error float32 float64
		int int8 int16 int32 int64 rune string uint uint8 uint16 uint32 uint64 uintptr
		append cap clear close complex copy delete imag len make max min new panic print println real recover
		true false nil iota`),
}

// One table covers the whole C-descended family: a keyword of a language the
// file isn't written in simply never appears in it.
var cSyntax = &Syntax{
	lineComment: "//",
	blockStart:  "/*",
	blockEnd:    "*/",
	quotes:      "\"'`",
	keywords: words(`abstract alignas alignof and as asm async await break case catch class
		const constexpr continue crate debugger default defer delete do dyn else enum explicit
		export extern final finally fn for friend function goto if impl implements import in
		instanceof interface let loop match mod move mut namespace new operator override package
		private protected pub public readonly ref register return sizeof static struct super
		switch synchronized template this throw throws trait try typedef typeof union unsafe
		use using var virtual volatile where while with yield`),
	types: words(`auto bool boolean byte char class double f32 f64 float i8 i16 i32 i64 i128
		int isize long never number object short signed size_t str string symbol tuple type u8
		u16 u32 u64 u128 uint unknown unsigned usize var void wchar_t
		true false null nil none undefined NULL nullptr self Self String Vec Option Result`),
}

// Anything whose comments start with '#'. Python and the shells share enough
// of their vocabulary that one table reads well for both.
var hashSyntax = &Syntax{
	lineComment: "#",
	quotes:      "\"'",
	keywords: words(`alias and as assert async await break case class continue declare def del
		do done elif else esac except exit export fi finally for from function global if import
		in is lambda local nonlocal not or pass raise readonly return select set shift source
		then trap try unset until while with yield`),
	types: words(`True False None self bool bytes dict float frozenset int list object set str tuple
		abs all any enumerate isinstance len open print range super type zip
		echo cd eval exec printf pwd read test true false`),
}

var extensionSyntax = map[string]*Syntax{
	".go": goSyntax,

	".c": cSyntax, ".h": cSyntax, ".cc": cSyntax, ".cpp": cSyntax, ".hpp": cSyntax,
	".cs": cSyntax, ".java": cSyntax, ".js": cSyntax, ".jsx": cSyntax, ".mjs": cSyntax,
	".cjs": cSyntax, ".ts": cSyntax, ".tsx": cSyntax, ".rs": cSyntax, ".kt": cSyntax,
	".swift": cSyntax, ".scala": cSyntax, ".dart": cSyntax, ".php": cSyntax, ".zig": cSyntax,

	".py": hashSyntax, ".sh": hashSyntax, ".bash": hashSyntax, ".zsh": hashSyntax,
	".rb": hashSyntax, ".pl": hashSyntax, ".yaml": hashSyntax, ".yml": hashSyntax,
	".toml": hashSyntax, ".tf": hashSyntax, ".conf": hashSyntax,
}

var baseNameSyntax = map[string]*Syntax{
	"makefile": hashSyntax, "dockerfile": hashSyntax, "gemfile": hashSyntax,
	".bashrc": hashSyntax, ".zshrc": hashSyntax, ".profile": hashSyntax,
	".gitignore": hashSyntax, ".gitconfig": hashSyntax, ".env": hashSyntax,
}

// detectSyntax returns nil for a file no rule matches, which leaves it drawn
// in the terminal's own colours.
func detectSyntax(name string) *Syntax {
	base := strings.ToLower(filepath.Base(name))
	if s, ok := extensionSyntax[filepath.Ext(base)]; ok {
		return s
	}

	return baseNameSyntax[base]
}

// colors is reused by every row of every redraw: highlighting a screenful
// allocates only when a line is longer than the longest one drawn so far.
var colors []termbox.Attribute

// lineColors returns one colour per rune of line, or nil when the file has no
// syntax, plus the block-comment state the line below starts in.
func lineColors(line []rune, inBlock bool) ([]termbox.Attribute, bool) {
	if syntax == nil {
		return nil, false
	}

	if cap(colors) < len(line) {
		colors = make([]termbox.Attribute, len(line))
	}

	out := colors[:len(line)]
	for i := range out {
		out[i] = colorPlain
	}

	return out, syntax.highlight(line, inBlock, out)
}

// A block comment opened above the window still colours the top of it, so the
// state has to be lexed rather than assumed. Walking back to the start of the
// file would make a redraw cost the whole buffer, so the search is bounded:
// past blockLookback lines the window is taken to start outside a comment.
const blockLookback = 64

func blockStateBefore(row int) bool {
	if syntax == nil || syntax.blockStart == "" {
		return false
	}

	inBlock := false
	for i := max(row-blockLookback, 0); i < row; i++ {
		inBlock = syntax.highlight(buf.Line(i), inBlock, nil)
	}

	return inBlock
}

func hasPrefixAt(line []rune, i int, prefix string) bool {
	if prefix == "" || i+len(prefix) > len(line) {
		return false
	}
	for _, ch := range prefix {
		if line[i] != ch {
			return false
		}
		i++
	}

	return true
}

func paint(out []termbox.Attribute, from, to int, color termbox.Attribute) {
	for i := from; i < to && i < len(out); i++ {
		out[i] = color
	}
}

// highlight colours one line into out (which may be nil to run the lexer for
// its state alone) and reports whether the line leaves a block comment open,
// which is the only state the next line needs.
func (s *Syntax) highlight(line []rune, inBlock bool, out []termbox.Attribute) bool {
	for i := 0; i < len(line); {
		switch {
		case inBlock:
			start := i
			i, inBlock = s.scanBlock(line, i)
			paint(out, start, i, colorComment)

		case hasPrefixAt(line, i, s.lineComment):
			paint(out, i, len(line), colorComment)

			return false

		case hasPrefixAt(line, i, s.blockStart):
			start := i
			i, inBlock = s.scanBlock(line, i+len(s.blockStart))
			paint(out, start, i, colorComment)

		case strings.ContainsRune(s.quotes, line[i]):
			start := i
			i = s.scanString(line, i)
			paint(out, start, i, colorString)

		case line[i] >= '0' && line[i] <= '9':
			start := i
			for i < len(line) && (isWordChar(line[i]) || line[i] == '.') {
				i++
			}
			paint(out, start, i, colorNumber)

		case isWordChar(line[i]):
			start := i
			for i < len(line) && isWordChar(line[i]) {
				i++
			}
			word := string(line[start:i])
			switch {
			case s.keywords[word]:
				paint(out, start, i, colorKeyword)
			case s.types[word]:
				paint(out, start, i, colorType)
			}

		default:
			i++
		}
	}

	return inBlock
}

func (s *Syntax) scanBlock(line []rune, i int) (int, bool) {
	for ; i < len(line); i++ {
		if hasPrefixAt(line, i, s.blockEnd) {
			return i + len(s.blockEnd), false
		}
	}

	return len(line), true
}

// An unterminated string runs to the end of the line rather than into the next
// one: half-typed quotes are normal in Insert mode, and colouring the rest of
// the file over one of them would be worse than getting the line wrong.
func (s *Syntax) scanString(line []rune, i int) int {
	quote := line[i]
	for i++; i < len(line); i++ {
		if line[i] == '\\' && quote != '`' {
			i++
			continue
		}
		if line[i] == quote {
			return i + 1
		}
	}

	return len(line)
}
