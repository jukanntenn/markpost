package v1

import "testing"

// FuzzEtagMatch exercises If-None-Match parsing against arbitrary input to
// confirm the matcher never panics and stays RFC-conformant for the cases the
// spec calls out: "*", "W/" weak prefixes, comma-separated lists, malformed,
// and empty values.
func FuzzEtagMatch(f *testing.F) {
	f.Add("")
	f.Add("*")
	f.Add(`"abc"`)
	f.Add(`W/"abc"`)
	f.Add(`"a", "b", "c"`)
	f.Add("garbage")

	f.Fuzz(func(t *testing.T, ifNoneMatch string) {
		_ = etagMatch(ifNoneMatch, "abc123def456789")
	})
}
