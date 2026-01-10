package utils

import "testing"

func TestGenerateState(t *testing.T) {
	got, err := GenerateState()
	if err != nil {
		t.Errorf("GenerateState() error = %v", err)
	}
	if len(got) != 27 {
		t.Errorf("GenerateState() length = %v, want 27", len(got))
	}
}
