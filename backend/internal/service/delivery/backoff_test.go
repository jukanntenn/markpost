package delivery

import (
	"testing"
	"time"
)

func TestBackoffSequence(t *testing.T) {
	got := BackoffSequence()
	want := []time.Duration{1 * time.Minute, 5 * time.Minute, 10 * time.Minute, 20 * time.Minute}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("seq[%d] = %v, want %v", i, got[i], want[i])
		}
	}
}

func TestNextBackoff(t *testing.T) {
	cases := []struct {
		attempts int
		want     time.Duration
		wantOK   bool
	}{
		{0, 1 * time.Minute, true},
		{1, 5 * time.Minute, true},
		{2, 10 * time.Minute, true},
		{3, 20 * time.Minute, true},
		{4, 0, false},
		{5, 0, false},
		{-1, 1 * time.Minute, true},
	}
	for _, c := range cases {
		got, ok := NextBackoff(c.attempts)
		if got != c.want || ok != c.wantOK {
			t.Errorf("NextBackoff(%d) = (%v, %v), want (%v, %v)", c.attempts, got, ok, c.want, c.wantOK)
		}
	}
}

func TestComputeExpiryWall(t *testing.T) {
	cases := []struct {
		name string
		seq  []time.Duration
		want time.Duration
	}{
		{"default rounds 36m up to 40m", []time.Duration{1 * time.Minute, 5 * time.Minute, 10 * time.Minute, 20 * time.Minute}, 40 * time.Minute},
		{"exact multiple stays put", []time.Duration{10 * time.Minute, 20 * time.Minute}, 30 * time.Minute},
		{"single sub-round rounds up", []time.Duration{1 * time.Minute}, 10 * time.Minute},
		{"empty yields zero", []time.Duration{}, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := computeExpiryWall(c.seq); got != c.want {
				t.Errorf("computeExpiryWall = %v, want %v", got, c.want)
			}
		})
	}
}

func TestExpiryWallMatchesDefaultSequence(t *testing.T) {
	if got := ExpiryWall(); got != 40*time.Minute {
		t.Errorf("ExpiryWall = %v, want 40m", got)
	}
}
