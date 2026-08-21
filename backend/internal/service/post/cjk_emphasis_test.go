package post

import (
	"bytes"
	"testing"
)

func TestRenderPostHTML_CJKEmpasisAdjacentToFullwidthPunct(t *testing.T) {
	// Regression: "**" adjacent to a CJK fullwidth right parenthesis （U+FF09）
	// previously failed to close, causing mis-paired emphasis. The CJK-aware
	// emphasis parser treats fullwidth punctuation as neutral so ** closes.
	cases := []struct {
		name string
		body string
		want string // substring expected in rendered HTML
	}{
		{
			name: "fullwidth paren closes emphasis",
			body: "本项目对**有限无自环**）的定理",
			want: "<strong>有限无自环</strong>",
		},
		{
			name: "fullwidth comma opens emphasis",
			body: "说明，**强调内容**结束",
			want: "<strong>强调内容</strong>",
		},
		{
			name: "fullwidth period closes emphasis",
			body: "前文**强调内容**。后文",
			want: "<strong>强调内容</strong>",
		},
		{
			name: "ascii paren still works (control)",
			body: "text **bold** (after)",
			want: "<strong>bold</strong>",
		},
		{
			name: "pure han chars still work (control)",
			body: "中文**强调**中文",
			want: "<strong>强调</strong>",
		},
		{
			name: "multiple emphasis with fullwidth punct",
			body: "**a**，**b**，**c**",
			want: "<strong>a</strong>",
		},
		// The cases above also pass without the CJK parser (the punctuation
		// sits on the opener's left or the closer's right, which CommonMark
		// already handles). The cases below are the actual failure shape —
		// fullwidth punctuation immediately LEFT of a closing delimiter — and
		// discriminate the fix from default goldmark.
		{
			name: "fullwidth paren left of closer (production shape)",
			body: "**加粗（注）**后续文字",
			want: "<strong>加粗（注）</strong>",
		},
		{
			name: "mid-line closer preceded by fullwidth paren",
			body: "前文**（注）**后续",
			want: "<strong>（注）</strong>",
		},
		{
			name: "comma-separated item closing after fullwidth paren",
			body: "列表，**项目（一）**结束",
			want: "<strong>项目（一）</strong>",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			md := newGoldmark()
			var buf bytes.Buffer
			if err := md.Convert([]byte(tc.body), &buf); err != nil {
				t.Fatalf("Convert: %v", err)
			}
			if !bytes.Contains(buf.Bytes(), []byte(tc.want)) {
				t.Errorf("rendered HTML missing %q\n got: %s", tc.want, buf.String())
			}
		})
	}
}
