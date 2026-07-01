// Package filter compiles and evaluates channel keyword filter expressions.
//
// Grammar (Model 2: spaces are keyword content, not separators):
//
//	expr   := or
//	or     := and ( ("," | "|") and )*   // OR, lowest precedence, left-assoc
//	and    := not ( "&" not )*           // AND, left-assoc
//	not    := "!" not | factor           // NOT, prefix, right-assoc
//	factor := KEYWORD | "(" expr ")"
//
// Operators are exactly seven ASCII characters: ,  |  &  !  (  )  "
// Any other character is literal keyword content. Multi-word phrases need no
// quotes ("key word" == key word). Quotes are required only to include operator
// characters in a keyword or to preserve leading/trailing spaces.
//
// Matching is case-insensitive substring, against the post title only. Both
// keywords and the title are normalized to Unicode NFC before comparison, so
// Korean/Vietnamese/diacritic-bearing scripts match across NFC/NFD forms.
// An empty expression matches everything (always deliver).
package filter

import "fmt"

// ParseError describes a malformed filter expression with a byte position.
type ParseError struct {
	Pos int
	Msg string
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("filter: parse error at pos %d: %s", e.Pos, e.Msg)
}

// Matcher evaluates a compiled expression against a post title.
type Matcher struct {
	root node
}

// Compile parses and compiles a filter expression. An empty or whitespace-only
// expression compiles to a matcher that matches everything.
func Compile(expr string) (*Matcher, error) {
	p := &parser{lex: &lexer{input: expr}}
	root, err := p.parse()
	if err != nil {
		return nil, err
	}
	return &Matcher{root: root}, nil
}

// MustCompile is like Compile but panics on error. Intended for tests.
func MustCompile(expr string) *Matcher {
	m, err := Compile(expr)
	if err != nil {
		panic(err)
	}
	return m
}

// Match reports whether the title satisfies the expression.
func (m *Matcher) Match(title string) bool {
	return m.root.eval(normalizeMatch(title))
}
