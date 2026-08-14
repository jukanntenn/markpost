package delivery

import (
	"context"
	"time"
)

// Repository defines the interface for delivery channel data access.
type Repository interface {
	GetByUserID(ctx context.Context, userID int) ([]Channel, error)
	GetByIDAndUserID(ctx context.Context, id int, userID int) (*Channel, error)
	GetByID(ctx context.Context, id int) (*Channel, error)
	Create(ctx context.Context, channel *Channel) error
	Update(ctx context.Context, channel *Channel) error
	SetEnabled(ctx context.Context, id int, enabled bool) error
	DeleteByIDAndUserID(ctx context.Context, id int, userID int) (int64, error)
	DeleteByID(ctx context.Context, id int) (int64, error)
	ListAll(ctx context.Context, offset, limit int) ([]Channel, error)
	CountAll(ctx context.Context) (int64, error)
}

// AttemptRepository defines persistence for the delivery best-effort retry
// queue. All batch/claim methods use the subquery-LIMIT form (bare
// DELETE/UPDATE ... LIMIT is a PostgreSQL syntax error).
type AttemptRepository interface {
	// Create inserts one or more pending attempts.
	Create(ctx context.Context, attempts []*Attempt) error
	// ClaimDue atomically claims up to limit due attempts (status=pending and
	// next_at <= now) and reserves each past the request timeout by advancing
	// next_at. FOR UPDATE SKIP LOCKED lets concurrent claimers pick disjoint
	// rows without blocking.
	ClaimDue(ctx context.Context, now, reserveUntilMs int64, limit int) ([]*Attempt, error)
	// MarkRetry records a failed (non-terminal) attempt: bumps the attempt
	// count, sets last_error, and schedules the next attempt at nextAtMs.
	MarkRetry(ctx context.Context, id int64, attempts int, lastError string, nextAtMs int64) error
	// MarkExpired transitions up to batchSize pending attempts whose
	// created_at is past the wall to expired, returning the claimed rows for
	// archival. It is called repeatedly by the scheduler until it returns none.
	MarkExpired(ctx context.Context, wallBeforeMs int64, batchSize int) ([]*Attempt, error)
	// ArchiveAndDelete writes a History row for the attempt's terminal state
	// and deletes the attempt row in a single transaction. errorCategory is the
	// classified send-failure category (empty for delivered/expired).
	ArchiveAndDelete(ctx context.Context, attempt *Attempt, status Status, lastError string, errorCategory string) error
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
	// LatestPerChannel returns the most recent history row for each of the
	// user's channels (one row per channel_id), used to render the per-channel
	// delivery-health overview on the channel list. Channels with no history
	// are absent from the result.
	LatestPerChannel(ctx context.Context, userID int) ([]*HistoryRow, error)
	// ListPending returns the user's in-flight (status=Pending) attempts joined
	// to their post and channel, for the dashboard activity feed's "投递中"
	// state (K.2 / GET /delivery/pending).
	ListPending(ctx context.Context, userID int) ([]*PendingAttemptRow, error)
	// DailyStats aggregates delivery_history rows by UTC day and status for the
	// user over the last days days (trend chart, B2.7). Days with no rows are
	// absent from the result.
	DailyStats(ctx context.Context, userID, days int) ([]*DailyStat, error)
	// DailyStatsAll is the admin cross-user variant of DailyStats (D2.5).
	DailyStatsAll(ctx context.Context, days int) ([]*DailyStat, error)
	// TodayCounts returns the user's today counters for the pipeline status
	// bar (K.2): delivered/failed from today's history plus pending from
	// in-flight attempts.
	TodayCounts(ctx context.Context, userID int) (*TodayCounts, error)
	// CountSince counts delivery_history rows created at or after since (admin
	// stats week delta, D2.4).
	CountSince(ctx context.Context, since time.Time) (int64, error)
	// LockedChannels returns channels whose 24h history window shows all
	// failures or a >50% failure rate (K.7 D2-1 SQL), for the admin "需要关注"
	// card (D2.1).
	LockedChannels(ctx context.Context) ([]*LockedChannel, error)
}

// HistoryFilter scopes a delivery_history read. A zero value selects every row
// (the admin all-rows view). OwnerID > 0 limits to one user; ChannelID > 0
// limits to one channel (always within the OwnerID scope when set); Status sets
// a terminal status filter (0 = no filter); ErrorCategory limits to one
// classified failure category ("" = no filter).
type HistoryFilter struct {
	OwnerID       int
	ChannelID     int
	Status        Status
	ErrorCategory string
}

// DailyStat is one day's terminal delivery outcome counts.
type DailyStat struct {
	Day       string `json:"day" gorm:"column:day"` // YYYY-MM-DD (UTC)
	Delivered int64  `json:"delivered" gorm:"column:delivered"`
	Failed    int64  `json:"failed" gorm:"column:failed"`
	Expired   int64  `json:"expired" gorm:"column:expired"`
}

// TodayCounts holds the user's today delivery counters (K.2).
type TodayCounts struct {
	Delivered int64 `json:"delivered"`
	Failed    int64 `json:"failed"`
	Pending   int64 `json:"pending"`
}

// PendingAttemptRow is the read projection of an in-flight attempt joined to
// its post and channel (K.2 activity feed "投递中" state).
type PendingAttemptRow struct {
	ID          int64     `json:"id" gorm:"column:id"`
	PostID      int       `json:"post_id" gorm:"column:post_id"`
	ChannelID   int       `json:"channel_id" gorm:"column:channel_id"`
	PostTitle   string    `json:"post_title" gorm:"column:post_title"`
	PostQID     string    `json:"post_qid" gorm:"column:post_qid"`
	ChannelName string    `json:"channel_name" gorm:"column:channel_name"`
	CreatedAt   time.Time `json:"created_at" gorm:"column:created_at"`
}

// LockedChannel is a channel flagged by the admin "需要关注" failing-channel
// query (D2.1/K.7): the 24h window shows all failures or a >50% failure rate.
type LockedChannel struct {
	ChannelID   int        `json:"channel_id" gorm:"column:channel_id"`
	ChannelName string     `json:"channel_name" gorm:"column:channel_name"`
	Username    string     `json:"username" gorm:"column:username"`
	Fails       int64      `json:"fails" gorm:"column:fails"`
	Total       int64      `json:"total" gorm:"column:total"`
	FailureRate float64    `json:"failure_rate" gorm:"column:failure_rate"`
	LastError   string     `json:"last_error" gorm:"column:last_error"`
	LastAt      *time.Time `json:"last_at" gorm:"column:last_at"`
}
