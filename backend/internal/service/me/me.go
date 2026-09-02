// Package me owns self-scoped reads for the authenticated caller. It opens
// the /me namespace: user-facing surfaces answering questions about the
// caller's own data gather here (MRFC 2026-09-02-user-facing-retention-visibility).
package me

import (
	"math"
	"time"

	"markpost/internal/domain/user"
)

// Service resolves the caller's effective retention policy. The auth
// middleware reloads the user row on every request, so the context user
// already carries the current policy; resolution stays pure.
type Service struct {
	globalPostRetentionDays int
	globalHistoryRetention  time.Duration
}

// NewService mirrors the global fallback windows from config: the values an
// inherit (nil) policy resolves to.
func NewService(globalPostRetentionDays int, globalHistoryRetention time.Duration) *Service {
	return &Service{
		globalPostRetentionDays: globalPostRetentionDays,
		globalHistoryRetention:  globalHistoryRetention,
	}
}

// RetentionResult is the caller's effective retention policy. 0 means keep
// forever, reusing [post] retention_days' zero encoding.
type RetentionResult struct {
	PostsDays   int `json:"posts_days"`
	HistoryDays int `json:"history_days"`
}

// EffectiveRetention mirrors the prune predicate (the per-row CASE in
// post_repo / delivery_attempt_repo): an explicit override drives both
// tables, an inherit reads each table's own global, so the two numbers
// diverge only when the globals drift apart.
func (s *Service) EffectiveRetention(u *user.User) RetentionResult {
	if u.RetentionDays != nil {
		return RetentionResult{PostsDays: *u.RetentionDays, HistoryDays: *u.RetentionDays}
	}
	return RetentionResult{
		PostsDays:   s.globalPostRetentionDays,
		HistoryDays: durationToDays(s.globalHistoryRetention),
	}
}

// durationToDays renders a retention duration in whole days, ceiling for a
// nonzero remainder — the display must never read "0 days" while meaning
// "not forever".
func durationToDays(d time.Duration) int {
	return int(math.Ceil(d.Hours() / 24))
}
