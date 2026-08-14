package infra

import (
	"context"
	"fmt"
	"time"

	"markpost/internal/domain/delivery"

	"gorm.io/gorm"
)

// AttemptRepository is the persistent best-effort delivery queue. It stores
// in-flight delivery attempts in delivery_attempts and archives terminal
// outcomes to delivery_history. All batch/claim operations use the
// dialect-safe subquery-LIMIT form (bare DELETE/UPDATE ... LIMIT is a Postgres
// syntax error).
type AttemptRepository struct {
	db *gorm.DB
}

// NewAttemptRepository creates an AttemptRepository backed by the given DB.
func NewAttemptRepository(db *gorm.DB) delivery.AttemptRepository {
	return &AttemptRepository{db: db}
}

// Create inserts one or more pending attempts.
func (r *AttemptRepository) Create(ctx context.Context, attempts []*delivery.Attempt) error {
	if len(attempts) == 0 {
		return nil
	}
	if err := r.db.WithContext(ctx).Create(&attempts).Error; err != nil {
		return fmt.Errorf("AttemptRepository.Create: %w", err)
	}
	return nil
}

// ClaimDue atomically claims up to limit due attempts (status=pending and
// next_at <= nowMs) and reserves each past the request timeout by advancing
// next_at to reserveUntilMs. This makes in-flight rows invisible to the next
// scheduler tick, preventing double-claim.
//
// The claim body (UPDATE ... WHERE id IN (SELECT ... LIMIT) RETURNING *)
// reserves rows by advancing next_at to reserveUntilMs, making in-flight rows
// invisible to the next scheduler tick (preventing double-claim). FOR UPDATE
// SKIP LOCKED lets concurrent claimers pick disjoint rows without blocking.
func (r *AttemptRepository) ClaimDue(ctx context.Context, nowMs, reserveUntilMs int64, limit int) ([]*delivery.Attempt, error) {
	if limit <= 0 {
		return nil, nil
	}

	// PostgreSQL supports FOR UPDATE SKIP LOCKED, letting concurrent claimers
	// (or scheduler ticks) claim disjoint rows without blocking.
	selectClause := "SELECT id FROM delivery_attempts WHERE status = ? AND next_at <= ? ORDER BY next_at LIMIT ? FOR UPDATE SKIP LOCKED"

	sql := fmt.Sprintf(
		"UPDATE delivery_attempts SET next_at = ? WHERE id IN (%s) RETURNING *",
		selectClause,
	)

	var claimed []*delivery.Attempt
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return tx.Raw(sql, reserveUntilMs, delivery.StatusPending, nowMs, limit).Scan(&claimed).Error
	})
	if err != nil {
		return nil, fmt.Errorf("AttemptRepository.ClaimDue: %w", err)
	}
	return claimed, nil
}

// MarkRetry records a failed (non-terminal) attempt: bumps the attempt count,
// stores the last error, and schedules the next attempt at nextAtMs.
func (r *AttemptRepository) MarkRetry(ctx context.Context, id int64, attempts int, lastError string, nextAtMs int64) error {
	result := r.db.WithContext(ctx).Model(&delivery.Attempt{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"attempts":   attempts,
			"last_error": lastError,
			"next_at":    nextAtMs,
		})
	if result.Error != nil {
		return fmt.Errorf("AttemptRepository.MarkRetry: %w", result.Error)
	}
	return nil
}

// MarkExpired transitions up to batchSize pending attempts whose created_at is
// past the wall (created_at < wallBefore) to expired, returning the claimed
// rows so the caller can archive them. It is called repeatedly by the
// scheduler until it returns an empty slice. Bounding the batch keeps each
// tick's lock scope and dead-tuple volume bounded even under a large pending
// backlog.
func (r *AttemptRepository) MarkExpired(ctx context.Context, wallBeforeMs int64, batchSize int) ([]*delivery.Attempt, error) {
	if batchSize <= 0 {
		return nil, nil
	}

	// created_at is a timestamp column. Compare against a time value in the same
	// form GORM stores (the driver's default location) rather than forcing UTC,
	// so the comparison is correct: PostgreSQL timestamps are timezone-aware
	// (any zone compares correctly).
	wallBefore := time.UnixMilli(wallBeforeMs)

	sql := `UPDATE delivery_attempts SET status = ?, updated_at = ?
	        WHERE id IN (
	            SELECT id FROM delivery_attempts
	            WHERE status = ? AND created_at < ?
	            ORDER BY created_at LIMIT ?
	        )
	        RETURNING *`

	var expired []*delivery.Attempt
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return tx.Raw(sql, delivery.StatusExpired, time.Now(), delivery.StatusPending, wallBefore, batchSize).Scan(&expired).Error
	})
	if err != nil {
		return nil, fmt.Errorf("AttemptRepository.MarkExpired: %w", err)
	}
	return expired, nil
}

// ArchiveAndDelete writes a History row for the attempt's terminal state and
// deletes the attempt row in a single transaction, so the archive and the
// queue removal are atomic. errorCategory is the classified send-failure
// category (empty for delivered/expired).
func (r *AttemptRepository) ArchiveAndDelete(ctx context.Context, attempt *delivery.Attempt, status delivery.Status, lastError string, errorCategory string) error {
	history := &delivery.History{
		UserID:        &attempt.UserID,
		PostID:        &attempt.PostID,
		ChannelID:     &attempt.ChannelID,
		Status:        status,
		LastError:     lastError,
		ErrorCategory: errorCategory,
		CreatedAt:     attempt.CreatedAt,
	}

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(history).Error; err != nil {
			return fmt.Errorf("insert history: %w", err)
		}
		if err := tx.Where("id = ?", attempt.ID).Delete(&delivery.Attempt{}).Error; err != nil {
			return fmt.Errorf("delete attempt: %w", err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("AttemptRepository.ArchiveAndDelete: %w", err)
	}
	return nil
}

// CountByStatus returns the count of attempts in each status, for observability.
func (r *AttemptRepository) CountByStatus(ctx context.Context) (map[delivery.Status]int64, error) {
	type row struct {
		Status delivery.Status `gorm:"column:status"`
		Count  int64           `gorm:"column:count"`
	}
	var rows []row
	if err := r.db.WithContext(ctx).
		Model(&delivery.Attempt{}).
		Select("status, COUNT(*) AS count").
		Group("status").
		Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("AttemptRepository.CountByStatus: %w", err)
	}

	out := make(map[delivery.Status]int64, len(rows))
	for _, rw := range rows {
		out[rw.Status] = rw.Count
	}
	return out, nil
}

// PruneHistory deletes delivery_history rows older than the retention window
// in batches of batchSize, returning the total deleted. It uses the
// subquery-LIMIT form (bare DELETE ... LIMIT is a PostgreSQL syntax error).
func (r *AttemptRepository) PruneHistory(ctx context.Context, retention time.Duration, batchSize int) (int64, error) {
	if batchSize <= 0 {
		batchSize = 1000
	}
	cutoff := time.Now().Add(-retention)

	var total int64
	for {
		sql := `DELETE FROM delivery_history WHERE id IN (
		            SELECT id FROM delivery_history WHERE created_at < ? ORDER BY created_at LIMIT ?
		        )`
		result := r.db.WithContext(ctx).Exec(sql, cutoff, batchSize)
		if result.Error != nil {
			return total, fmt.Errorf("AttemptRepository.PruneHistory: %w", result.Error)
		}
		total += result.RowsAffected
		if result.RowsAffected < int64(batchSize) {
			break
		}
	}
	return total, nil
}

// ListHistory returns delivery history (newest first), paginated, with the post
// title/qid, channel name, and username JOINed at read time (the spec's
// normalization rule). LEFT JOIN preserves rows whose referenced post/channel/
// user was deleted — the corresponding pointer field is nil. filter scopes the
// result (see delivery.HistoryFilter).
func (r *AttemptRepository) ListHistory(ctx context.Context, filter delivery.HistoryFilter, offset, limit int) ([]*delivery.HistoryRow, error) {
	q := r.db.WithContext(ctx).Table("delivery_history AS h").
		Select(`h.id, h.status, h.last_error, h.error_category, h.created_at, h.channel_id,
		        p.title AS post_title, p.qid AS post_qid,
		        c.name AS channel_name,
		        u.username AS username`).
		Joins("LEFT JOIN posts p ON p.id = h.post_id").
		Joins("LEFT JOIN delivery_channels c ON c.id = h.channel_id").
		Joins("LEFT JOIN users u ON u.id = h.user_id").
		Order("h.created_at DESC")
	if filter.OwnerID > 0 {
		q = q.Where("h.user_id = ?", filter.OwnerID)
	}
	if filter.ChannelID > 0 {
		q = q.Where("h.channel_id = ?", filter.ChannelID)
	}
	if filter.Status > 0 {
		q = q.Where("h.status = ?", filter.Status)
	}
	if filter.ErrorCategory != "" {
		q = q.Where("h.error_category = ?", filter.ErrorCategory)
	}
	var rows []*delivery.HistoryRow
	if err := q.Offset(offset).Limit(limit).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("AttemptRepository.ListHistory: %w", err)
	}
	return rows, nil
}

// CountHistory returns the total row count matching the same filter as
// ListHistory, for pagination.
func (r *AttemptRepository) CountHistory(ctx context.Context, filter delivery.HistoryFilter) (int64, error) {
	q := r.db.WithContext(ctx).Model(&delivery.History{})
	if filter.OwnerID > 0 {
		q = q.Where("user_id = ?", filter.OwnerID)
	}
	if filter.ChannelID > 0 {
		q = q.Where("channel_id = ?", filter.ChannelID)
	}
	if filter.Status > 0 {
		q = q.Where("status = ?", filter.Status)
	}
	if filter.ErrorCategory != "" {
		q = q.Where("error_category = ?", filter.ErrorCategory)
	}
	var count int64
	if err := q.Count(&count).Error; err != nil {
		return 0, fmt.Errorf("AttemptRepository.CountHistory: %w", err)
	}
	return count, nil
}

// LatestPerChannel returns the most recent delivery_history row per channel for
// the user (one row per channel_id). It uses a correlated subquery so the
// statement is simple and portable. The data volume is small - a user's 7-day
// retention window, typically well under a thousand rows - so the correlated
// MAX lookup is effectively index-backed by the channel_id index and resolves
// in milliseconds.
func (r *AttemptRepository) LatestPerChannel(ctx context.Context, userID int) ([]*delivery.HistoryRow, error) {
	const sql = `SELECT h.id, h.status, h.last_error, h.created_at, h.channel_id,
	                    p.title AS post_title, p.qid AS post_qid,
	                    c.name AS channel_name,
	                    u.username AS username
	               FROM delivery_history AS h
	               LEFT JOIN posts p             ON p.id = h.post_id
	               LEFT JOIN delivery_channels c ON c.id = h.channel_id
	               LEFT JOIN users u             ON u.id = h.user_id
	              WHERE h.user_id = ?
	                AND h.created_at = (
	                    SELECT MAX(h2.created_at)
	                      FROM delivery_history h2
	                     WHERE h2.channel_id = h.channel_id
	                       AND h2.user_id = h.user_id
	                )
	              ORDER BY h.channel_id`
	var rows []*delivery.HistoryRow
	if err := r.db.WithContext(ctx).Raw(sql, userID).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("AttemptRepository.LatestPerChannel: %w", err)
	}
	return rows, nil
}

// ListPending returns the user's in-flight (status=Pending) attempts joined to
// their post (title/qid) and channel (name). Used by the dashboard activity
// feed's "投递中" state (K.2).
func (r *AttemptRepository) ListPending(ctx context.Context, userID int) ([]*delivery.PendingAttemptRow, error) {
	const sql = `SELECT a.id, a.post_id, a.channel_id,
	                    p.title AS post_title, p.qid AS post_qid,
	                    c.name AS channel_name,
	                    a.created_at
	               FROM delivery_attempts AS a
	               LEFT JOIN posts p             ON p.id = a.post_id
	               LEFT JOIN delivery_channels c ON c.id = a.channel_id
	              WHERE a.user_id = ? AND a.status = ?
	              ORDER BY a.created_at DESC`
	var rows []*delivery.PendingAttemptRow
	if err := r.db.WithContext(ctx).Raw(sql, userID, delivery.StatusPending).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("AttemptRepository.ListPending: %w", err)
	}
	return rows, nil
}

// dailyStatsQuery is the shared GROUP BY day/status aggregation (B2.7/D2.5).
// Scope is appended by the caller.
func dailyStatsQuery(db *gorm.DB, ctx context.Context, userID int, days int, scope string, args ...any) ([]*delivery.DailyStat, error) {
	sql := `SELECT TO_CHAR(DATE(created_at), 'YYYY-MM-DD') AS day,
	                    COUNT(*) FILTER (WHERE status = ?) AS delivered,
	                    COUNT(*) FILTER (WHERE status = ?) AS failed,
	                    COUNT(*) FILTER (WHERE status = ?) AS expired
	               FROM delivery_history
	              WHERE created_at >= ? ` + scope + `
	              GROUP BY DATE(created_at)
	              ORDER BY day ASC`
	params := append([]any{delivery.StatusDelivered, delivery.StatusFailed, delivery.StatusExpired, time.Now().AddDate(0, 0, -days+1)}, args...)
	var rows []*delivery.DailyStat
	if err := db.WithContext(ctx).Raw(sql, params...).Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// DailyStats aggregates delivery_history rows by UTC day and status for the
// user over the last days days (trend chart, B2.7).
func (r *AttemptRepository) DailyStats(ctx context.Context, userID, days int) ([]*delivery.DailyStat, error) {
	rows, err := dailyStatsQuery(r.db, ctx, userID, days, "AND user_id = ?", userID)
	if err != nil {
		return nil, fmt.Errorf("AttemptRepository.DailyStats: %w", err)
	}
	return rows, nil
}

// DailyStatsAll is the admin cross-user variant of DailyStats (D2.5).
func (r *AttemptRepository) DailyStatsAll(ctx context.Context, days int) ([]*delivery.DailyStat, error) {
	rows, err := dailyStatsQuery(r.db, ctx, 0, days, "")
	if err != nil {
		return nil, fmt.Errorf("AttemptRepository.DailyStatsAll: %w", err)
	}
	return rows, nil
}

// TodayCounts returns the user's today counters (K.2): delivered/failed from
// today's history plus pending from in-flight attempts.
func (r *AttemptRepository) TodayCounts(ctx context.Context, userID int) (*delivery.TodayCounts, error) {
	const sql = `SELECT
	                (SELECT COUNT(*) FROM delivery_history
	                  WHERE user_id = ? AND status = ? AND created_at >= date_trunc('day', now())) AS delivered,
	                (SELECT COUNT(*) FROM delivery_history
	                  WHERE user_id = ? AND status IN (?, ?) AND created_at >= date_trunc('day', now())) AS failed,
	                (SELECT COUNT(*) FROM delivery_attempts
	                  WHERE user_id = ? AND status = ?) AS pending`
	var out delivery.TodayCounts
	if err := r.db.WithContext(ctx).Raw(sql,
		userID, delivery.StatusDelivered,
		userID, delivery.StatusFailed, delivery.StatusExpired,
		userID, delivery.StatusPending,
	).Scan(&out).Error; err != nil {
		return nil, fmt.Errorf("AttemptRepository.TodayCounts: %w", err)
	}
	return &out, nil
}

// CountSince counts delivery_history rows created at or after since (admin
// stats week delta, D2.4).
func (r *AttemptRepository) CountSince(ctx context.Context, since time.Time) (int64, error) {
	return countQuery(ctx, r.db.Model(&delivery.History{}).Where("created_at >= ?", since), "CountSince")
}

// LockedChannels returns channels whose 24h history window shows all failures
// or a >50% failure rate (K.7 D2-1 SQL), for the admin "需要关注" card (D2.1).
func (r *AttemptRepository) LockedChannels(ctx context.Context) ([]*delivery.LockedChannel, error) {
	const sql = `SELECT h.channel_id,
	                    c.name AS channel_name,
	                    u.username AS username,
	                    COUNT(*) FILTER (WHERE h.status IN (?, ?)) AS fails,
	                    COUNT(*) AS total,
	                    (COUNT(*) FILTER (WHERE h.status IN (?, ?)))::float / COUNT(*) AS failure_rate,
	                    (SELECT h2.last_error FROM delivery_history h2
	                      WHERE h2.channel_id = h.channel_id
	                      ORDER BY h2.created_at DESC LIMIT 1) AS last_error,
	                    MAX(h.created_at) AS last_at
	               FROM delivery_history AS h
	               LEFT JOIN delivery_channels c ON c.id = h.channel_id
	               LEFT JOIN users u             ON u.id = h.user_id
	              WHERE h.created_at > now() - INTERVAL '24 hours'
	              GROUP BY h.channel_id, c.name, u.username
	             HAVING (COUNT(*) FILTER (WHERE h.status IN (?, ?)) = COUNT(*))
	                 OR (COUNT(*) FILTER (WHERE h.status IN (?, ?))) * 100 / COUNT(*) > 50
	              ORDER BY total DESC`
	var rows []*delivery.LockedChannel
	if err := r.db.WithContext(ctx).Raw(sql,
		delivery.StatusFailed, delivery.StatusExpired,
		delivery.StatusFailed, delivery.StatusExpired,
		delivery.StatusFailed, delivery.StatusExpired,
		delivery.StatusFailed, delivery.StatusExpired,
	).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("AttemptRepository.LockedChannels: %w", err)
	}
	return rows, nil
}
