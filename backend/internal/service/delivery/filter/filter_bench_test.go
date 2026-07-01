package filter

import (
	"strings"
	"testing"
)

// Expression complexity tiers, exercising lexer/parser/AST size independently
// of the title being matched.
var benchExprs = []struct {
	name string
	expr string
}{
	{"Empty", ""},
	{"Single", "alert"},
	{"SimpleOr", "alpha, beta, gamma, delta, epsilon"},
	{"SimpleAnd", "alpha & beta & gamma & delta & epsilon"},
	{"Compound", "prod & (error, warning, fatal) & !debug & !(staging, local)"},
	{"DeepNest", "((((a | b) & (c | d)) | ((e | f) & (g | h))) & !(i | j)) | k"},
}

// Titles spanning realistic lengths, with a guaranteed substring so that a
// "hit" path actually finds something (vs. scanning the whole string on a miss).
func benchTitle(n int) string {
	var b strings.Builder
	b.WriteString("prod environment alert: ")
	word := " lorem ipsum dolor sit amet consectetur"
	for b.Len()+len(word) <= n {
		b.WriteString(word)
	}
	rest := n - b.Len()
	if rest > 0 {
		b.WriteString(word[:rest])
	}
	return b.String()
}

var (
	benchTitleShort  = benchTitle(32)
	benchTitleMedium = benchTitle(256)
	benchTitleLong   = benchTitle(4096)
)

func BenchmarkCompile(b *testing.B) {
	for _, bc := range benchExprs {
		b.Run(bc.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_, err := Compile(bc.expr)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkMatch(b *testing.B) {
	// A compound expression that hits on the title prefix (prod+alert) and is
	// not trivially short-circuited, so the AST is fully exercised.
	expr := "prod & (error, warning, alert) & !debug"
	m, err := Compile(expr)
	if err != nil {
		b.Fatal(err)
	}

	cases := []struct {
		name  string
		title string
	}{
		{"Short", benchTitleShort},
		{"Medium", benchTitleMedium},
		{"Long", benchTitleLong},
	}
	for _, c := range cases {
		b.Run(c.name+"_Hit", func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_ = m.Match(c.title)
			}
		})
		// A title guaranteed to miss every keyword drives the worst-case path:
		// the whole normalized title is scanned for each keyword before failing.
		miss := strings.Repeat("zzz ", len(c.title)/4)
		b.Run(c.name+"_Miss", func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_ = m.Match(miss)
			}
		})
	}
}

// BenchmarkCompileAndMatch mirrors the per-channel, per-post cost paid inside
// Deliver: the expression is recompiled each delivery (no caching by design).
func BenchmarkCompileAndMatch(b *testing.B) {
	expr := "prod & (error, warning, alert) & !debug"
	titles := []struct {
		name  string
		title string
	}{
		{"Short", benchTitleShort},
		{"Medium", benchTitleMedium},
		{"Long", benchTitleLong},
	}
	for _, t := range titles {
		b.Run(t.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				m, err := Compile(expr)
				if err != nil {
					b.Fatal(err)
				}
				_ = m.Match(t.title)
			}
		})
	}
}

// BenchmarkNormalize isolates the per-Match title cost (NFC + ToLower), the
// dominant factor for long titles, independent of AST size.
func BenchmarkNormalize(b *testing.B) {
	cases := []struct {
		name  string
		title string
	}{
		{"Short", benchTitleShort},
		{"Medium", benchTitleMedium},
		{"Long", benchTitleLong},
	}
	for _, c := range cases {
		b.Run(c.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_ = normalizeMatch(c.title)
			}
		})
	}
}
