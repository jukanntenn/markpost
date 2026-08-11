package auth

import (
	"sync"
	"time"
)

// C2.1 层 B：账号级登录失败锁定（自建 map，非 tollbooth——需要"锁定到期"
// 语义；I.13 单实例假设，重启丢失可接受）。
// 连续 maxLoginAttempts 次失败 → 锁定 lockDuration；锁定期间任何登录
// （含正确密码）都返回 account_locked，且计数不继续加（K.7 C2-1，
// 防锁定期无限延长）。登录成功或锁定到期 → 清零。
const (
	maxLoginAttempts = 5
	lockDuration     = 15 * time.Minute
)

type attemptState struct {
	count       int
	lockedUntil time.Time
}

// LoginAttemptTracker tracks per-username failed login attempts in memory.
type LoginAttemptTracker struct {
	mu       sync.Mutex
	attempts map[string]*attemptState
}

// NewLoginAttemptTracker creates an empty tracker with a background sweeper
// that removes expired lock entries.
func NewLoginAttemptTracker() *LoginAttemptTracker {
	t := &LoginAttemptTracker{attempts: make(map[string]*attemptState)}
	go t.sweepLoop()
	return t
}

// CheckLocked reports whether the username is currently locked and the
// remaining lock duration. Entries that are merely accumulating failure counts
// (no lock yet) are left untouched.
func (t *LoginAttemptTracker) CheckLocked(username string) (bool, time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()
	st := t.attempts[username]
	if st == nil || st.lockedUntil.IsZero() {
		return false, 0
	}
	remaining := time.Until(st.lockedUntil)
	if remaining <= 0 {
		delete(t.attempts, username)
		return false, 0
	}
	return true, remaining
}

// RecordFailure registers one failed login. Returns whether the account just
// became (or already is) locked and the remaining lock duration. Count is only
// incremented while not locked (K.7 C2-1).
func (t *LoginAttemptTracker) RecordFailure(username string) (locked bool, remaining time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()
	st := t.attempts[username]
	now := time.Now()
	if st != nil {
		if !st.lockedUntil.IsZero() {
			if remaining = time.Until(st.lockedUntil); remaining > 0 {
				return true, remaining
			}
			// 锁定已到期：清除后按全新计数。
			delete(t.attempts, username)
			st = nil
		}
	}
	if st == nil {
		st = &attemptState{}
	}
	st.count++
	if st.count >= maxLoginAttempts {
		st.lockedUntil = now.Add(lockDuration)
		st.count = 0
		t.attempts[username] = st
		return true, lockDuration
	}
	t.attempts[username] = st
	return false, 0
}

// Reset clears the failure counter (successful login, or lock expiry handled
// lazily by the accessors above).
func (t *LoginAttemptTracker) Reset(username string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.attempts, username)
}

func (t *LoginAttemptTracker) sweepLoop() {
	ticker := time.NewTicker(lockDuration / 2)
	defer ticker.Stop()
	for range ticker.C {
		t.sweep()
	}
}

func (t *LoginAttemptTracker) sweep() {
	t.mu.Lock()
	defer t.mu.Unlock()
	now := time.Now()
	for name, st := range t.attempts {
		if now.After(st.lockedUntil) {
			delete(t.attempts, name)
		}
	}
}
