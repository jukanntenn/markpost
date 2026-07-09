package v1

import "testing"

func TestEtagMatch(t *testing.T) {
	const current = "8957c7305ed33f00"
	currentQuoted := `"` + current + `"`

	tests := []struct {
		name        string
		ifNoneMatch string
		etag        string
		want        bool
	}{
		{"empty header", "", current, false},
		{"exact match", currentQuoted, current, true},
		{"wildcard", "*", current, true},
		{"whitespace wildcard", " * ", current, true},
		{"mismatch", `"deadbeef"`, current, false},
		{"weak prefix on client matches", `W/` + currentQuoted, current, true},
		{"one of multiple matches", `"deadbeef", ` + currentQuoted, current, true},
		{"none of multiple matches", `"aaa", "bbb"`, current, false},
		{"malformed empty member", `,, ` + currentQuoted + `,`, current, true},
		{"wildcard within list", `"deadbeef", *`, current, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := etagMatch(tc.ifNoneMatch, tc.etag); got != tc.want {
				t.Errorf("etagMatch(%q, %q) = %v, want %v", tc.ifNoneMatch, tc.etag, got, tc.want)
			}
		})
	}
}

func TestEtagMatch_NeverPanics(t *testing.T) {
	for _, inm := range []string{
		"", "*", "W/", "W/*", `W/"abc"`, `""`, "w/", ",,,",
		`"a", W/"b", *`, "garbage without quotes",
	} {
		_ = etagMatch(inm, "abc123")
	}
}
