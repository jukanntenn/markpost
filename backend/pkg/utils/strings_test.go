package utils

import "testing"

func TestNormalize(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"  Hello World  ", "hello world"},
		{"abc", "abc"},
		{"", ""},
		{"   ", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := Normalize(tt.input); got != tt.expected {
				t.Errorf("Normalize(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestApplyIfNonEmpty(t *testing.T) {
	t.Run("applies non-empty trimmed value", func(t *testing.T) {
		target := "old"
		ApplyIfNonEmpty(&target, "  new  ")
		if target != "new" {
			t.Errorf("expected %q, got %q", "new", target)
		}
	})

	t.Run("does nothing when value is only whitespace", func(t *testing.T) {
		target := "old"
		ApplyIfNonEmpty(&target, "   ")
		if target != "old" {
			t.Errorf("expected %q, got %q", "old", target)
		}
	})

	t.Run("does nothing when value is empty string", func(t *testing.T) {
		target := "old"
		ApplyIfNonEmpty(&target, "")
		if target != "old" {
			t.Errorf("expected %q, got %q", "old", target)
		}
	})
}
