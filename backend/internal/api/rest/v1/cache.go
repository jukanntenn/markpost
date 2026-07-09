package v1

import "strings"

// etagMatch reports whether the client's If-None-Match header matches the
// response ETag, per RFC 9110 §13.1.2 / RFC 9112 §8.8.3 semantics used for
// conditional GET revalidation:
//
//   - "*" matches any current representation (the origin always has one here);
//   - a comma-separated list of ETags matches if any member equals the current
//     ETag (strong comparison — we never emit weak validators);
//   - a leading "W/" weak validator prefix is accepted on the client side and
//     compared as a strong match, since our responses are byte-stable.
//
// The current etag is the bare hex value; both sides are wrapped in quotes for
// comparison.
func etagMatch(ifNoneMatch, etag string) bool {
	if ifNoneMatch == "" {
		return false
	}
	if strings.TrimSpace(ifNoneMatch) == "*" {
		return true
	}
	current := `"` + etag + `"`
	for _, raw := range strings.Split(ifNoneMatch, ",") {
		candidate := strings.TrimSpace(raw)
		if candidate == "" {
			continue
		}
		if strings.HasPrefix(candidate, "W/") {
			candidate = strings.TrimPrefix(candidate, "W/")
			candidate = strings.TrimSpace(candidate)
		}
		if candidate == "*" || candidate == current {
			return true
		}
	}
	return false
}
