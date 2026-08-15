package post

import (
	"regexp"
	"testing"
)

var stampedImgRe = regexp.MustCompile(`(?i)<img\b referrerpolicy="no-referrer"`)

func countMatches(re *regexp.Regexp, s string) int {
	return len(re.FindAllStringIndex(s, -1))
}

// FuzzAddNoReferrerToImages exercises the post-sanitize image stamper with
// arbitrary input to confirm it never panics and preserves its contract: the
// number of <img open tags is unchanged, and every one of them carries the
// injected attribute right after the tag name.
func FuzzAddNoReferrerToImages(f *testing.F) {
	f.Add(`<img src="https://forum.example/x.png">`)
	f.Add("<img>")
	f.Add(`<IMG SRC=x>`)
	f.Add("<image><imgfoo>")
	f.Add(`<p>a</p> <img src=a> <img src=b> text`)
	f.Add("")
	f.Add("<img\nsrc=x>")

	f.Fuzz(func(t *testing.T, htmlContent string) {
		out := addNoReferrerToImages(htmlContent)

		if in, got := countMatches(imgTagRe, htmlContent), countMatches(imgTagRe, out); in != got {
			t.Fatalf("img count changed: in=%d out=%d\nin:  %q\nout: %q", in, got, htmlContent, out)
		}
		if total, stamped := countMatches(imgTagRe, out), countMatches(stampedImgRe, out); total != stamped {
			t.Fatalf("not every img stamped: total=%d stamped=%d\nin:  %q\nout: %q", total, stamped, htmlContent, out)
		}
	})
}
