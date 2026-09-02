package me

import (
	"testing"
	"time"

	"markpost/internal/domain/user"
)

func TestEffectiveRetention(t *testing.T) {
	forever, thirty := 0, 30

	cases := []struct {
		name string
		svc  *Service
		u    *user.User
		want RetentionResult
	}{
		{
			name: "explicit forever drives both tables",
			svc:  NewService(7, 168*time.Hour),
			u:    &user.User{RetentionDays: &forever},
			want: RetentionResult{PostsDays: 0, HistoryDays: 0},
		},
		{
			name: "explicit N drives both tables",
			svc:  NewService(7, 168*time.Hour),
			u:    &user.User{RetentionDays: &thirty},
			want: RetentionResult{PostsDays: 30, HistoryDays: 30},
		},
		{
			name: "inherit reads each table's own global",
			svc:  NewService(7, 168*time.Hour),
			u:    &user.User{},
			want: RetentionResult{PostsDays: 7, HistoryDays: 7},
		},
		{
			name: "inherit under global post retention 0 keeps posts forever",
			svc:  NewService(0, 168*time.Hour),
			u:    &user.User{},
			want: RetentionResult{PostsDays: 0, HistoryDays: 7},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.svc.EffectiveRetention(tc.u); got != tc.want {
				t.Errorf("EffectiveRetention() = %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestDurationToDays(t *testing.T) {
	cases := []struct {
		d    time.Duration
		want int
	}{
		{168 * time.Hour, 7}, // exact days
		{36 * time.Hour, 2},  // nonzero remainder rounds up
		{12 * time.Hour, 1},  // sub-day never displays as 0
		{90 * time.Hour, 4},  // 3.75 → 4
	}
	for _, tc := range cases {
		if got := durationToDays(tc.d); got != tc.want {
			t.Errorf("durationToDays(%v) = %d, want %d", tc.d, got, tc.want)
		}
	}
}
