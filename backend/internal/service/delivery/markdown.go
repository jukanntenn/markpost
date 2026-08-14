package delivery

import "regexp"

var (
	// inlineImgRe matches inline image markdown ![alt](url).
	inlineImgRe = regexp.MustCompile(`!\[([^\]]*)\]\([^)]*\)`)
	// refImgRe matches reference-style image markdown ![alt][ref].
	refImgRe = regexp.MustCompile(`!\[([^\]]*)\]\[[^\]]+\]`)
)

// stripImages removes markdown image syntax from s, keeping any alt text.
//
// The Feishu card markdown element only accepts Feishu-hosted image keys
// (img_v2_*) as image sources — never arbitrary URLs. A body preview that
// carries ![alt](http://...) makes Feishu reject the whole card with
// "card contains invalid image keys" (ErrCode 200570), so images are stripped
// before the preview is sent. Alt text is preserved so a meaningful label
// remains; an empty alt removes the image outright.
func stripImages(s string) string {
	s = inlineImgRe.ReplaceAllString(s, "$1")
	s = refImgRe.ReplaceAllString(s, "$1")
	return s
}
