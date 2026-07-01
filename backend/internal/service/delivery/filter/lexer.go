package filter

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

type tokenKind int

const (
	tokenEOF tokenKind = iota
	tokenKeyword
	tokenComma
	tokenPipe
	tokenAmp
	tokenNot
	tokenLParen
	tokenRParen
)

func (k tokenKind) String() string {
	switch k {
	case tokenEOF:
		return "EOF"
	case tokenKeyword:
		return "keyword"
	case tokenComma:
		return "','"
	case tokenPipe:
		return "'|'"
	case tokenAmp:
		return "'&'"
	case tokenNot:
		return "'!'"
	case tokenLParen:
		return "'('"
	case tokenRParen:
		return "')'"
	}
	return "unknown"
}

type token struct {
	kind  tokenKind
	value string
	pos   int
}

func isOperatorByte(c byte) bool {
	switch c {
	case ',', '|', '&', '!', '(', ')', '"':
		return true
	}
	return false
}

type lexer struct {
	input string
	pos   int
}

func (l *lexer) skipSpace() {
	for l.pos < len(l.input) {
		r, size := utf8.DecodeRuneInString(l.input[l.pos:])
		if !unicode.IsSpace(r) {
			break
		}
		l.pos += size
	}
}

func (l *lexer) next() (token, error) {
	l.skipSpace()
	if l.pos >= len(l.input) {
		return token{kind: tokenEOF, pos: l.pos}, nil
	}

	start := l.pos
	switch l.input[l.pos] {
	case ',':
		l.pos++
		return token{kind: tokenComma, pos: start}, nil
	case '|':
		l.pos++
		return token{kind: tokenPipe, pos: start}, nil
	case '&':
		l.pos++
		return token{kind: tokenAmp, pos: start}, nil
	case '!':
		l.pos++
		return token{kind: tokenNot, pos: start}, nil
	case '(':
		l.pos++
		return token{kind: tokenLParen, pos: start}, nil
	case ')':
		l.pos++
		return token{kind: tokenRParen, pos: start}, nil
	case '"':
		return l.readQuoted(start)
	}
	return l.readBare(start)
}

func (l *lexer) readBare(start int) (token, error) {
	var buf strings.Builder
	for l.pos < len(l.input) {
		c := l.input[l.pos]
		if isOperatorByte(c) {
			break
		}
		buf.WriteByte(c)
		l.pos++
	}
	value := strings.TrimSpace(buf.String())
	return token{kind: tokenKeyword, value: value, pos: start}, nil
}

func (l *lexer) readQuoted(start int) (token, error) {
	l.pos++

	var buf strings.Builder
	for {
		if l.pos >= len(l.input) {
			return token{}, &ParseError{Pos: start, Msg: "unterminated quoted string"}
		}
		c := l.input[l.pos]
		if c == '"' {
			if l.pos+1 < len(l.input) && l.input[l.pos+1] == '"' {
				buf.WriteByte('"')
				l.pos += 2
				continue
			}
			l.pos++
			return token{kind: tokenKeyword, value: buf.String(), pos: start}, nil
		}
		buf.WriteByte(c)
		l.pos++
	}
}
