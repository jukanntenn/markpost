package delivery

import "testing"

func TestStripImages(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		{"no images", "hello world", "hello world"},
		{"inline empty alt", "![](http://img.x/a.jpg)", ""},
		{"inline with alt", "![截图](http://img.x/a.jpg)", "截图"},
		{"inline with alt and surrounding text", "before ![pic](http://x/a.png) after", "before pic after"},
		{"multiple images", "![](http://x/1.jpg)![](http://x/2.jpg)", ""},
		{"image plus text keeps text", "![](http://x/1.jpg)\n正文内容", "\n正文内容"},
		{"reference image", "![logo][logo-ref]", "logo"},
		{"link is not an image", "[a link](http://x)", "[a link](http://x)"},
		{"url with title", `![alt](http://x/a.jpg "title")`, "alt"},
		{"cjk alt", "![图片说明](https://pic.y/b.jpg)", "图片说明"},
		{"mixed image and link", "see [link](http://x) and ![](http://y/z.png) end", "see [link](http://x) and  end"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := stripImages(tt.in); got != tt.want {
				t.Errorf("stripImages(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestBuildBodyPreviewStripsImages(t *testing.T) {
	// The regression: a body with an inline image must not reach the Feishu card.
	body := "##### 标题\n\n![](http://img.zuanke8.cn/forum/202608/12/165503abc.jpg)\n\n正文"
	got := buildBodyPreview(body, 200)
	if containsImg := func(s string) bool {
		for i := 0; i+1 < len(s); i++ {
			if s[i] == '!' && s[i+1] == '[' {
				return true
			}
		}
		return false
	}; containsImg(got) {
		t.Errorf("buildBodyPreview kept image markdown: %q", got)
	}
}
