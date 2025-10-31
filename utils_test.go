package main

import (
	"strings"
	"testing"
)

func TestGenerateState(t *testing.T) {
	got, err := GenerateState()
	if err != nil {
		t.Errorf("GenerateState() error = %v", err)
	}
	if len(got) != 27 {
		t.Errorf("GenerateState() length = %v, want 27", len(got))
	}
}

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

func TestHashPassword(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty string", ""},
		{"normal string", "password"},
		{"complex string", "ComplexP@ssw0rd!@#"},
		{"very long string", "aVeryLongPasswordThatIsMuchLongerThanTypicalPasswords123456789"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := HashPassword(test.input)
			if err != nil {
				t.Errorf("HashPassword(%s) error = %v", test.input, err)
			}
			if got == "" {
				t.Errorf("HashPassword(%s) = empty string", test.input)
			}
		})
	}
}

func TestCheckPassword(t *testing.T) {
	hashed, _ := HashPassword("password")

	tests := []struct {
		name        string
		input       string
		shouldMatch bool
	}{
		{"matching password", "password", true},
		{"wrong password", "wrongpassword", false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := CheckPassword(test.input, hashed)
			if test.shouldMatch && err != nil {
				t.Errorf("CheckPassword(%s) = %v, want nil", test.input, err)
			}
			if !test.shouldMatch && err == nil {
				t.Errorf("CheckPassword(%s) = nil, want error", test.input)
			}
		})
	}
}
