package post

import "testing"

// FuzzSanitizeCacheTag exercises the purge-tag sanitizer with arbitrary QIDs to
// confirm it strips quote/backslash/newline characters that could break the
// Cloudflare JSON body or enable header injection, and never panics.
func FuzzSanitizeCacheTag(f *testing.F) {
	f.Add("p-abc")
	f.Add(`p-a"b`)
	f.Add("p-a\\b")
	f.Add("p-a\nb\rc")
	f.Add("")

	f.Fuzz(func(t *testing.T, qid string) {
		got := sanitizeCacheTag(qid)
		for _, bad := range []string{"\"", "\\", "\n", "\r"} {
			if contains(got, bad) {
				t.Errorf("sanitized tag %q still contains %q", got, bad)
			}
		}
	})
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
