package utils

import (
	"strings"
	"testing"
)

func TestGeneratePostKey(t *testing.T) {
	tests := []struct {
		name  string
		input int
	}{
		{"byteLength equals 8", 8},
		{"byteLength equals 0", 0},
		{"byteLength equals -8", -8},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := GeneratePostKey(test.input)
			if err != nil {
				t.Errorf("GeneratePostKey(%d) error = %v", test.input, err)
			}
			if got == "" {
				t.Errorf("GeneratePostKey(%d) = empty string", test.input)
			}
			if !strings.HasPrefix(got, "mpk-") {
				t.Errorf("GeneratePostKey(%d) missing prefix mpk-: %s", test.input, got)
			}
			baseLen := test.input
			if baseLen <= 0 {
				baseLen = 20
			}
			if len(got) != baseLen+4 {
				t.Errorf("GeneratePostKey(%d) length = %d, want %d", test.input, len(got), baseLen+4)
			}
			suffix := got[4:]
			for _, ch := range suffix {
				if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9')) {
					t.Errorf("GeneratePostKey(%d) contains non-alphanumeric in suffix: %q", test.input, ch)
				}
			}
		})
	}
}
