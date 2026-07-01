package filter

import "fmt"

type parser struct {
	lex *lexer
	cur token
}

func (p *parser) advance() {
	t, err := p.lex.next()
	if err != nil {
		panic(err)
	}
	p.cur = t
}

func (p *parser) parse() (n node, err error) {
	defer func() {
		if r := recover(); r != nil {
			if pe, ok := r.(*ParseError); ok {
				err = pe
				return
			}
			panic(r)
		}
	}()

	p.advance()
	if p.cur.kind == tokenEOF {
		return alwaysTrueNode{}, nil
	}

	n = p.parseOr()
	if p.cur.kind != tokenEOF {
		return nil, &ParseError{Pos: p.cur.pos, Msg: fmt.Sprintf("unexpected %s", p.cur.kind)}
	}
	return n, nil
}

func (p *parser) parseOr() node {
	left := p.parseAnd()
	for p.cur.kind == tokenComma || p.cur.kind == tokenPipe {
		p.advance()
		right := p.parseAnd()
		left = orNode{left: left, right: right}
	}
	return left
}

func (p *parser) parseAnd() node {
	left := p.parseNot()
	for p.cur.kind == tokenAmp {
		p.advance()
		right := p.parseNot()
		left = andNode{left: left, right: right}
	}
	return left
}

func (p *parser) parseNot() node {
	if p.cur.kind == tokenNot {
		p.advance()
		return notNode{operand: p.parseNot()}
	}
	return p.parseFactor()
}

func (p *parser) parseFactor() node {
	switch p.cur.kind {
	case tokenLParen:
		p.advance()
		inner := p.parseOr()
		if p.cur.kind != tokenRParen {
			panic(&ParseError{Pos: p.cur.pos, Msg: fmt.Sprintf("expected ')', got %s", p.cur.kind)})
		}
		p.advance()
		return inner
	case tokenKeyword:
		if p.cur.value == "" {
			panic(&ParseError{Pos: p.cur.pos, Msg: "empty keyword"})
		}
		kw := newKeywordNode(p.cur.value)
		p.advance()
		return kw
	}
	panic(&ParseError{Pos: p.cur.pos, Msg: fmt.Sprintf("unexpected %s", p.cur.kind)})
}

func newKeywordNode(raw string) node {
	return keywordNode{lower: normalizeMatch(raw)}
}
