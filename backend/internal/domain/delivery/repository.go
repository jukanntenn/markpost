package delivery

import (
	"context"
	"time"
)

// Repository defines the interface for delivery channel data access.
type Repository interface {
	GetByUserID(ctx context.Context, userID int) ([]Channel, error)
	GetByIDAndUserID(ctx context.Context, id int, userID int) (*Channel, error)
	Create(ctx context.Context, channel *Channel) error
	Update(ctx context.Context, channel *Channel) error
	DeleteByIDAndUserID(ctx context.Context, id int, userID int) (int64, error)
	ListAll(ctx context.Context, offset, limit int) ([]Channel, error)
	CountAll(ctx context.Context) (int64, error)
}

// AttemptRepository defines persistence for the delivery best-effort retry
// queue. All batch/claim methods use the dialect-safe subquery-LIMIT form so
// they are valid across Postgres, MySQL, and SQLite (bare DELETE/UPDATE ...
// LIMIT is a Postgres syntax error).
type AttemptRepository interface {
	// Create inserts one or more pending attempts.
	Create(ctx context.Context, attempts []*Attempt) error
	// ClaimDue atomically claims up to limit due attempts (status=pending and
	// next_at <= now) and reserves each past the request timeout by advancing
	// next_at. FOR UPDATE SKIP LOCKED is used on Postgres/MySQL; on SQLite it
	// is omitted (single-connection serialization prevents double-claim).
	ClaimDue(ctx context.Context, now, reserveUntilMs int64, limit int) ([]*Attempt, error)
	// MarkRetry records a failed (non-terminal) attempt: bumps the attempt
	// count, sets last_error, and schedules the next attempt at nextAtMs.
	MarkRetry(ctx context.Context, id int64, attempts int, lastError string, nextAtMs int64) error
	// MarkExpired transitions up to batchSize pending attempts whose
	// created_at is past the wall to expired, returning the claimed rows for
	// archival. It is called repeatedly by the scheduler until it returns none.
	MarkExpired(ctx context.Context, wallBeforeMs int64, batchSize int) ([]*Attempt, error)
	// ArchiveAndDelete writes a History row for the attempt's terminal state
	// and deletes the attempt row in a single transaction.
	ArchiveAndDelete(ctx context.Context, attempt *Attempt, status Status, lastError string) error
	// CountByStatus returns the count of attempts in each status, for
	// observability.
	CountByStatus(ctx context.Context) (map[Status]int64, error)
	// PruneHistory deletes delivery_history rows older than the retention
	// window in batches of batchSize, returning the total deleted. It uses the
	// portable subquery-LIMIT form.
	PruneHistory(ctx context.Context, retention time.Duration, batchSize int) (int64, error)
	// ListHistory returns delivery history (newest first), paginated, with the
	// post title/qid, channel name, and username JOINed at read time. filter
	// scopes the result: OwnerID > 0 limits to one user (NULL user_id rows are
	// excluded from a user's own page); OwnerID == 0 lists all rows including
	// anonymized ones (admin view). ChannelID > 0 further limits to one channel.
	ListHistory(ctx context.Context, filter HistoryFilter, offset, limit int) ([]*HistoryRow, error)
	// CountHistory returns the total row count matching the same filter as
	// ListHistory, for pagination.
	CountHistory(ctx context.Context, filter HistoryFilter) (int64, error)
}

// HistoryFilter scopes a delivery_history read. A zero value selects every row
// (the admin all-rows view). OwnerID > 0 limits to one user; ChannelID > 0
// limits to one channel (always within the OwnerID scope when set).
type HistoryFilter struct {
	OwnerID   int
	ChannelID int
}
