package delivery

import "time"

// backoffSequence is the hardcoded list of delays applied after each failed
// delivery attempt. The first attempt happens immediately on claim (no leading
// wait); each subsequent attempt waits backoffSequence[attempts-1] before
// retrying. With this sequence a delivery exhausts at t=36m after up to 5
// attempts (1 immediate + 4 retries).
//
// Hardcoded, not configurable: there is exactly one delivery channel kind
// today (Feishu), so per-channel retry tuning has no consumer. Changing the
// sequence is a code change + release, which already restarts the process and
// clears in-flight state. See specs/backend/delivery.md Decision 4.
var backoffSequence = [...]time.Duration{
	1 * time.Minute,
	5 * time.Minute,
	10 * time.Minute,
	20 * time.Minute,
}

// BackoffSequence returns a copy of the hardcoded retry delay sequence. It is
// read-only at the call site.
func BackoffSequence() []time.Duration {
	out := make([]time.Duration, len(backoffSequence))
	copy(out, backoffSequence[:])
	return out
}

// NextBackoff returns the delay before the (attempts+1)-th delivery attempt,
// or ok=false if the sequence is exhausted (attempts already covers the whole
// sequence). attempts is the count of attempts already performed.
func NextBackoff(attempts int) (time.Duration, bool) {
	if attempts < 0 {
		attempts = 0
	}
	if attempts >= len(backoffSequence) {
		return 0, false
	}
	return backoffSequence[attempts], true
}

// computeExpiryWall derives the hard time wall from a backoff sequence:
//
//	wall = round_up_to_10min( sum(sequence) )
//
// The round-up guarantees a non-zero margin so the last retry does not collide
// with the wall. For the default [1m,5m,10m,20m]: sum=36m → 40m. An empty
// sequence (no-retry mode) yields 0, meaning the wall does not participate.
func computeExpiryWall(seq []time.Duration) time.Duration {
	const round = 10 * time.Minute

	var sum time.Duration
	for _, d := range seq {
		sum += d
	}
	if sum <= 0 {
		return 0
	}
	if r := sum % round; r != 0 {
		sum += round - r
	}
	return sum
}

// ExpiryWall returns the auto-computed expiry wall for the hardcoded backoff
// sequence.
func ExpiryWall() time.Duration {
	return computeExpiryWall(backoffSequence[:])
}
