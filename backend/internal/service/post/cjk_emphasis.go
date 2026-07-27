package post

import (
	"unicode"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// This file works around a known CommonMark deficiency for CJK text.
// CommonMark's emphasis flanking rules (spec 6.2) treat Unicode punctuation
// specially: a ** right after a punctuation rune (e.g. the fullwidth right
// parenthesis ) U+FF09) cannot close emphasis. goldmark's util.IsPunctRune
// (unicode.IsSymbol || unicode.IsPunct) classifies CJK fullwidth punctuation
// as punctuation, so 中文**强调**) fails to close and the ** sequences pair
// incorrectly. The goldmark author acknowledged this in issue #61:
// "CommonMark spec are created by westerners, so they did not take account
// into our languages."
//
// The fix: register a custom emphasis InlineParser that scans delimiters with
// a CJK-friendly punctuation predicate, treating CJK fullwidth punctuation as
// neutral (neither punctuation nor whitespace) so ** closes normally. This is
// the only mechanism Go's ecosystem offers - goldmark has no CJK emphasis
// extension, and forking is avoided by using exported APIs only.

// cjkEmphasisParser replaces the default emphasis parser to apply a CJK-aware
// punctuation check when scanning delimiters.
type cjkEmphasisParser struct{}

// Trigger returns the bytes that start emphasis parsing (same as default).
func (s *cjkEmphasisParser) Trigger() []byte {
	return []byte{'*', '_'}
}

// Parse mirrors parser.emphasisParser.Parse but calls scanCJKDelimiter
// instead of parser.ScanDelimiter.
func (s *cjkEmphasisParser) Parse(parent ast.Node, block text.Reader, pc parser.Context) ast.Node {
	before := block.PrecendingCharacter()
	line, segment := block.PeekLine()
	node := scanCJKDelimiter(line, before, 1, defaultEmphasisDelimiterProcessor)
	if node == nil {
		return nil
	}
	node.Segment = segment.WithStop(segment.Start + node.OriginalLength)
	block.Advance(node.OriginalLength)
	pc.PushDelimiter(node)
	return node
}

// defaultEmphasisDelimiterProcessor mirrors goldmark's unexported one. It is
// safe to keep our own copy because the processor only depends on the
// delimiter byte and match length, not on any internal state.
type cjkEmphasisDelimiterProcessor struct{}

func (p *cjkEmphasisDelimiterProcessor) IsDelimiter(b byte) bool {
	return b == '*' || b == '_'
}

func (p *cjkEmphasisDelimiterProcessor) CanOpenCloser(opener, closer *parser.Delimiter) bool {
	return opener.Char == closer.Char
}

func (p *cjkEmphasisDelimiterProcessor) OnMatch(consumes int) ast.Node {
	return ast.NewEmphasis(consumes)
}

var defaultEmphasisDelimiterProcessor = &cjkEmphasisDelimiterProcessor{}

// scanCJKDelimiter is a copy of parser.ScanDelimiter with one change:
// isPunctRune replaces util.IsPunctRune so that CJK fullwidth punctuation
// (e.g. )(,。、) is treated as a neutral rune instead of punctuation.
// Everything else is byte-for-byte identical to goldmark@v1.7.13
// parser/delimiter.go:114 ScanDelimiter.
func scanCJKDelimiter(line []byte, before rune, minimum int, processor parser.DelimiterProcessor) *Delimiter {
	i := 0
	c := line[i]
	j := i
	if !processor.IsDelimiter(c) {
		return nil
	}
	for ; j < len(line) && c == line[j]; j++ {
	}
	if (j - i) >= minimum {
		after := rune(' ')
		if j != len(line) {
			after = util.ToRune(line, j)
		}

		var canOpen, canClose bool
		beforeIsPunctuation := isPunctRune(before)
		beforeIsWhitespace := util.IsSpaceRune(before)
		afterIsPunctuation := isPunctRune(after)
		afterIsWhitespace := util.IsSpaceRune(after)

		isLeft := !afterIsWhitespace &&
			(!afterIsPunctuation || beforeIsWhitespace || beforeIsPunctuation)
		isRight := !beforeIsWhitespace &&
			(!beforeIsPunctuation || afterIsWhitespace || afterIsPunctuation)

		if line[i] == '_' {
			canOpen = isLeft && (!isRight || beforeIsPunctuation)
			canClose = isRight && (!isLeft || afterIsPunctuation)
		} else {
			canOpen = isLeft
			canClose = isRight
		}
		return parser.NewDelimiter(canOpen, canClose, j-i, c, processor)
	}
	return nil
}

// Delimiter is an alias for parser.Delimiter so the copy above reads cleanly.
type Delimiter = parser.Delimiter

// isPunctRune is util.IsPunctRune with CJK fullwidth punctuation excluded.
// CJK fullwidth punctuation (Pe/Ps/Pi/Po categories in the CJK ranges) is the
// root cause of the flanking failure; treating it as neutral lets ** close
// normally. Han ideographs are already non-punctuation in unicode.IsPunct, so
// they are unaffected either way.
func isPunctRune(r rune) bool {
	if isCJKFullwidthPunct(r) {
		return false
	}
	return unicode.IsSymbol(r) || unicode.IsPunct(r)
}

// isCJKFullwidthPunct reports whether r is a CJK fullwidth/halfwidth
// punctuation rune. These are the ASCII punctuation counterparts remapped into
// the U+FF00 block plus the CJK-specific punctuation in U+3000-303F and the
// traditional/fullwidth forms in U+FE10-FE6F. By excluding exactly these from
// the punctuation predicate, emphasis delimiters adjacent to them behave as if
// they were adjacent to a letter, which matches CJK user expectations.
func isCJKFullwidthPunct(r rune) bool {
	switch {
	// Fullwidth ASCII variants: ！＂＃＄％＆＇（）＊＋，－．／：；＜＝＞？＠［＼］＾＿｀｛｜｝～
	case r >= 0xFF01 && r <= 0xFF0F,
		r >= 0xFF1A && r <= 0xFF20,
		r >= 0xFF3B && r <= 0xFF40,
		r >= 0xFF5B && r <= 0xFF65:
		return true
	// CJK Symbols and Punctuation: 、。〃〄々〇〒〓〈〉《》「」『』【】 etc.
	case r >= 0x3001 && r <= 0x303F:
		return true
	// Vertical / presentation forms punctuation
	case r >= 0xFE10 && r <= 0xFE1F,
		r >= 0xFE30 && r <= 0xFE4F,
		r >= 0xFE50 && r <= 0xFE6F:
		return true
	}
	return false
}
