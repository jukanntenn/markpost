package filter

import (
	"errors"
	"testing"
)

type matchCase struct {
	title string
	want  bool
}

func runCases(t *testing.T, expr string, cs []matchCase) {
	t.Helper()
	m, err := Compile(expr)
	if err != nil {
		t.Fatalf("compile %q: unexpected error: %v", expr, err)
	}
	for _, e := range cs {
		if got := m.Match(e.title); got != e.want {
			t.Errorf("expr=%q title=%q: got %v want %v", expr, e.title, got, e.want)
		}
	}
}

func TestCompile_Semantics(t *testing.T) {
	t.Run("single match", func(t *testing.T) {
		runCases(t, "alpha", []matchCase{
			{"hello alpha world", true},
			{"ALPHA", true},
			{"alphabeta", true},
			{"beta gamma", false},
		})
	})
	t.Run("OR comma", func(t *testing.T) {
		runCases(t, "alpha, beta, gamma", []matchCase{
			{"alpha", true},
			{"has beta here", true},
			{"only GAMMA", true},
			{"nothing", false},
		})
	})
	t.Run("OR pipe equals comma", func(t *testing.T) {
		runCases(t, "alpha | beta | gamma", []matchCase{
			{"alpha", true}, {"beta", true}, {"gamma delta", true}, {"delta", false},
		})
	})
	t.Run("AND", func(t *testing.T) {
		runCases(t, "alpha & beta & gamma", []matchCase{
			{"alpha beta gamma", true},
			{"alpha beta", false},
			{"beta gamma", false},
		})
	})
	t.Run("NOT", func(t *testing.T) {
		runCases(t, "!alpha", []matchCase{
			{"beta gamma", true},
			{"alpha", false},
			{"alphabeta", false},
		})
	})
	t.Run("double negation is identity", func(t *testing.T) {
		runCases(t, "!!alpha", []matchCase{
			{"alpha", true}, {"beta", false},
		})
	})
	t.Run("quadruple negation", func(t *testing.T) {
		runCases(t, "!!!!alpha", []matchCase{
			{"alpha", true}, {"beta", false},
		})
	})
	t.Run("odd negation", func(t *testing.T) {
		runCases(t, "!!!alpha", []matchCase{
			{"alpha", false}, {"beta", true},
		})
	})
	t.Run("multi-word keyword (Model 2)", func(t *testing.T) {
		runCases(t, "key word 1, key word 2", []matchCase{
			{"the key word 1 here", true},
			{"key word 2 appears", true},
			{"keyword 1", false},
			{"key word 3", false},
		})
	})
	t.Run("NOT a multi-word keyword", func(t *testing.T) {
		runCases(t, "! key word", []matchCase{
			{"the key word here", false},
			{"the keyword here", true},
		})
	})
	t.Run("quoted equals unquoted", func(t *testing.T) {
		runCases(t, `"alpha" & "beta"`, []matchCase{
			{"alpha beta", true}, {"alpha", false},
		})
	})
}

func TestCompile_Precedence(t *testing.T) {
	t.Run("AND binds tighter than OR: a | b & c == a | (b & c)", func(t *testing.T) {
		runCases(t, "alpha | beta & gamma", []matchCase{
			{"alpha", true},
			{"beta gamma", true},
			{"beta", false},
			{"gamma", false},
		})
	})
	t.Run("NOT binds tighter than AND: !a & b == (!a) & b", func(t *testing.T) {
		runCases(t, "!alpha & beta", []matchCase{
			{"beta", true},
			{"alpha beta", false},
			{"alpha", false},
		})
	})
	t.Run("NOT binds tighter than OR: !a | b == (!a) | b", func(t *testing.T) {
		runCases(t, "!alpha | beta", []matchCase{
			{"gamma", true},
			{"alpha", false},
			{"alpha beta", true},
		})
	})
	t.Run("parentheses override: (a | b) & c", func(t *testing.T) {
		runCases(t, "(alpha | beta) & gamma", []matchCase{
			{"alpha gamma", true},
			{"beta gamma", true},
			{"alpha", false},
			{"gamma", false},
		})
	})
	t.Run("nested parentheses", func(t *testing.T) {
		runCases(t, "((alpha | beta) & !gamma) | delta", []matchCase{
			{"alpha", true},
			{"alpha gamma", false},
			{"beta", true},
			{"delta anything gamma", true},
		})
	})
	t.Run("realistic combined rule", func(t *testing.T) {
		runCases(t, "prod & (error, warning) & !debug", []matchCase{
			{"[prod] error occurred", true},
			{"[prod] warning: high cpu", true},
			{"[prod] error debug=true", false},
			{"[staging] error", false},
			{"[prod] info", false},
		})
	})
}

func TestCompile_SpecialCharsAndQuoting(t *testing.T) {
	t.Run("operators-free specials need no quotes", func(t *testing.T) {
		runCases(t, "C++, a/b, a\\b", []matchCase{
			{"learn C++ today", true},
			{"path a/b/c", true},
			{"win a\\b shell", true},
			{"plain text", false},
		})
	})
	t.Run("quoted keyword containing comma", func(t *testing.T) {
		runCases(t, `"a,b"`, []matchCase{
			{"x,a,b,y", true},
			{"a, b", false},
		})
	})
	t.Run("quoted keyword containing ampersand", func(t *testing.T) {
		runCases(t, `"a & b"`, []matchCase{
			{"see a & b now", true},
			{"a &  b", false},
		})
	})
	t.Run(`double-quote doubling -> literal quote`, func(t *testing.T) {
		runCases(t, `"say ""hi"""`, []matchCase{
			{`she say "hi" please`, true},
			{`she say hi please`, false},
		})
	})
	t.Run(`four quotes -> keyword is a single "`, func(t *testing.T) {
		runCases(t, `""""`, []matchCase{
			{`has " in it`, true},
			{`no quote here`, false},
		})
	})
	t.Run("backslash is always literal", func(t *testing.T) {
		runCases(t, `a\b & c\d`, []matchCase{
			{`a\b and c\d`, true},
			{`a\b only`, false},
		})
	})
	t.Run("quote preserves leading space", func(t *testing.T) {
		runCases(t, `" err"`, []matchCase{
			{"foo err", true},
			{"fooerr", false},
		})
	})
}

func TestCompile_EmptyMatchesAll(t *testing.T) {
	for _, expr := range []string{"", "   ", "\t\n  "} {
		m, err := Compile(expr)
		if err != nil {
			t.Errorf("compile %q: unexpected error %v", expr, err)
			continue
		}
		for _, title := range []string{"anything", "", "ALPHA", "🚀 错误 C++"} {
			if !m.Match(title) {
				t.Errorf("empty expr %q should match title %q", expr, title)
			}
		}
	}
}

func TestCompile_ValidEdgeExpressions(t *testing.T) {
	valid := []string{
		"a", "(a)", "((a))", "!a", "!(a)", "a & b", "a, b", "a | b",
		`"a,b"`, `"a & b"`, `""""`, `!!a`, `! !a`,
		"(a | b) & !c", "prod & (error, warning) & !debug",
		`a\b`, `C++`, `a/b`,
	}
	for _, e := range valid {
		if _, err := Compile(e); err != nil {
			t.Errorf("expected %q to compile, got %v", e, err)
		}
	}
}

func TestCompile_RejectsInvalid(t *testing.T) {
	invalid := []string{
		"a,,b", "a && b", "a &", "& a", "&", "|", ",", ",a", "a,",
		"a | | b", "a & & b",
		"!", "! &", "(,)",
		"(a", "a)", "()", "(a,)", ")(a",
		`"abc`, `"""`,
		"a (b)", "(a)(b)", `a"b"`, "alpha (beta)",
		`""`, `a & ""`, "(), a",
		"& | ,", "! &", "(!)",
	}
	for _, e := range invalid {
		m, err := Compile(e)
		if err == nil {
			t.Errorf("expected %q to be REJECTED, but compiled to %v", e, m)
			continue
		}
		var pe *ParseError
		if !errors.As(err, &pe) {
			t.Errorf("expected *ParseError for %q, got %T: %v", e, err, err)
		}
	}
}
