package filter

import (
	"math/rand"
	"testing"
)

func FuzzCompile_NeverPanics(f *testing.F) {
	seeds := []string{
		"", "a", "a, b, c", "a & b & !c", "(a|b)&c",
		`"a,b"`, `"say ""hi"""`, `""""`, "key word 1, key word 2",
		"C++/a\\b", "🚀go", "错误", "!a & (b | !c)",
		`"unterminated`, "a &&& b", "(((", "! ! ! a",
		"中文 关键词 & !英文", "a\tb\nc",
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Add(string([]byte{0xff, 0xfe, 0x00, 0x2c}))

	f.Fuzz(func(t *testing.T, in string) {
		m, err := Compile(in)
		if err != nil {
			return
		}
		for _, title := range []string{"", "x", "ALPHA", "alpha beta", "🚀", "a,b", "a & b"} {
			_ = m.Match(title)
		}
	})
}

func generateTitles(rnd *rand.Rand, words []string, n int) []string {
	seps := []string{" ", ", ", " & ", "/", "", "  "}
	out := make([]string, 0, n)
	for i := 0; i < n; i++ {
		k := 1 + rnd.Intn(4)
		var b []byte
		for j := 0; j < k; j++ {
			if j > 0 {
				b = append(b, seps[rnd.Intn(len(seps))]...)
			}
			w := words[rnd.Intn(len(words))]
			if rnd.Intn(2) == 0 {
				w = upperRandom(rnd, w)
			}
			b = append(b, w...)
		}
		out = append(out, string(b))
	}
	return out
}

func upperRandom(rnd *rand.Rand, w string) string {
	out := []byte(w)
	for i := range out {
		if rnd.Intn(2) == 0 && out[i] >= 'a' && out[i] <= 'z' {
			out[i] -= 32
		}
	}
	return string(out)
}

func assertEquivalent(t *testing.T, titles []string, lhs, rhs string) {
	t.Helper()
	ml, errl := Compile(lhs)
	if errl != nil {
		t.Fatalf("compile lhs=%q err=%v", lhs, errl)
	}
	mr, errr := Compile(rhs)
	if errr != nil {
		t.Fatalf("compile rhs=%q err=%v", rhs, errr)
	}
	for _, ti := range titles {
		if ml.Match(ti) != mr.Match(ti) {
			t.Errorf("inequivalence: lhs=%q rhs=%q title=%q", lhs, rhs, ti)
		}
	}
}

func TestProperty_DeMorgan(t *testing.T) {
	rnd := rand.New(rand.NewSource(1))
	words := []string{"alpha", "beta", "gamma", "delta", "epsilon"}
	titles := generateTitles(rnd, words, 2000)

	for i := 0; i < 200; i++ {
		a, b := words[rnd.Intn(len(words))], words[rnd.Intn(len(words))]
		lhs := "!(" + a + " & " + b + ")"
		rhs := "!" + a + " | !" + b
		assertEquivalent(t, titles, lhs, rhs)
	}
}

func TestProperty_DoubleNegation(t *testing.T) {
	rnd := rand.New(rand.NewSource(2))
	words := []string{"alpha", "beta", "gamma", "C++", "错误"}
	titles := generateTitles(rnd, words, 2000)
	for _, a := range words {
		assertEquivalent(t, titles, "!!"+a, a)
		assertEquivalent(t, titles, "!!!!"+a, a)
	}
}

func TestProperty_Commutativity(t *testing.T) {
	rnd := rand.New(rand.NewSource(3))
	words := []string{"alpha", "beta", "gamma", "delta"}
	titles := generateTitles(rnd, words, 2000)
	for _, a := range words {
		for _, b := range words {
			assertEquivalent(t, titles, a+" & "+b, b+" & "+a)
			assertEquivalent(t, titles, a+" | "+b, b+" | "+a)
		}
	}
}

func TestProperty_Distributivity(t *testing.T) {
	rnd := rand.New(rand.NewSource(4))
	words := []string{"alpha", "beta", "gamma"}
	titles := generateTitles(rnd, words, 3000)
	for _, a := range words {
		for _, b := range words {
			for _, c := range words {
				lhs := a + " & (" + b + " | " + c + ")"
				rhs := a + " & " + b + " | " + a + " & " + c
				assertEquivalent(t, titles, lhs, rhs)
			}
		}
	}
}

func TestProperty_TautologyAndContradiction(t *testing.T) {
	rnd := rand.New(rand.NewSource(5))
	words := []string{"alpha", "beta", "gamma"}
	titles := generateTitles(rnd, words, 2000)
	for _, a := range words {
		mTrue, err := Compile(a + " | !" + a)
		if err != nil {
			t.Fatalf("compile tautology: %v", err)
		}
		mFalse, err := Compile(a + " & !" + a)
		if err != nil {
			t.Fatalf("compile contradiction: %v", err)
		}
		for _, ti := range titles {
			if !mTrue.Match(ti) {
				t.Errorf("tautology a|!a should always match; a=%q title=%q", a, ti)
			}
			if mFalse.Match(ti) {
				t.Errorf("contradiction a&!a should never match; a=%q title=%q", a, ti)
			}
		}
	}
}
