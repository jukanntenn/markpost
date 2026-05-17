package delivery

import "testing"

func TestApplyIfNonEmpty(t *testing.T) {
	t.Run("non-empty value", func(t *testing.T) {
		s := "original"
		applyIfNonEmpty(&s, "  hello  ")
		if s != "hello" {
			t.Errorf("got %q, want %q", s, "hello")
		}
	})

	t.Run("empty value does not overwrite", func(t *testing.T) {
		s := "original"
		applyIfNonEmpty(&s, "")
		if s != "original" {
			t.Errorf("got %q, want %q", s, "original")
		}
	})

	t.Run("whitespace-only value overwrites with trimmed empty", func(t *testing.T) {
		s := "original"
		applyIfNonEmpty(&s, "   ")
		if s != "" {
			t.Errorf("got %q, want %q", s, "")
		}
	})
}
