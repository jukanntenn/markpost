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
// The claim body (UPDATE ... WHERE id IN (SELECT ... LIMIT) RETURNING *) is
// identical across dialects; only the locking clause differs:
//   - Postgres/MySQL: FOR UPDATE SKIP LOCKED lets concurrent claimers (or
//     scheduler ticks) claim disjoint rows without blocking.
//   - SQLite: the clause is omitted because SQLite has no row-level locking
//     (it is a parse-time syntax error). Production SQLite pins
//     SetMaxOpenConns(1), so writes serialize through one connection and there
//     is no concurrent claimer to exclude.
func (r *AttemptRepository) ClaimDue(ctx context.Context, nowMs, reserveUntilMs int64, limit int) ([]*delivery.Attempt, error) {
	if limit <= 0 {
		return nil, nil
	}

	selectClause := "SELECT id FROM delivery_attempts WHERE status = ? AND next_at <= ? ORDER BY next_at LIMIT ?"
	if r.rowLockingDialect() {
		selectClause += " FOR UPDATE SKIP LOCKED"
	}

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
	// so the comparison is correct across dialects: Postgres/MySQL timestamps
	// are timezone-aware (any zone compares correctly), while SQLite stores a
	// formatted string and only compares correctly when the bound value uses the
	// same offset the stored rows use.
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
// queue removal are atomic.
func (r *AttemptRepository) ArchiveAndDelete(ctx context.Context, attempt *delivery.Attempt, status delivery.Status, lastError string) error {
	history := &delivery.History{
		UserID:    &attempt.UserID,
		PostID:    &attempt.PostID,
		ChannelID: &attempt.ChannelID,
		Status:    status,
		LastError: lastError,
		CreatedAt: attempt.CreatedAt,
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
// in batches of batchSize, returning the total deleted. It uses the portable
// subquery-LIMIT form (bare DELETE ... LIMIT is a Postgres syntax error;
// SQLite supports it only when the driver is compiled with the right flag).
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
// user was deleted — the corresponding pointer field is nil. ownerID > 0 scopes
// to one user; ownerID == 0 lists all rows (admin view, including anonymized).
func (r *AttemptRepository) ListHistory(ctx context.Context, ownerID int, offset, limit int) ([]*delivery.HistoryRow, error) {
	q := r.db.WithContext(ctx).Table("delivery_history AS h").
		Select(`h.id, h.status, h.last_error, h.created_at,
		        p.title AS post_title, p.qid AS post_qid,
		        c.name AS channel_name,
		        u.username AS username`).
		Joins("LEFT JOIN posts p ON p.id = h.post_id").
		Joins("LEFT JOIN delivery_channels c ON c.id = h.channel_id").
		Joins("LEFT JOIN users u ON u.id = h.user_id").
		Order("h.created_at DESC")
	if ownerID > 0 {
		q = q.Where("h.user_id = ?", ownerID)
	}
	var rows []*delivery.HistoryRow
	if err := q.Offset(offset).Limit(limit).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("AttemptRepository.ListHistory: %w", err)
	}
	return rows, nil
}

// CountHistory returns the total row count matching the same ownerID filter as
// ListHistory, for pagination.
func (r *AttemptRepository) CountHistory(ctx context.Context, ownerID int) (int64, error) {
	q := r.db.WithContext(ctx).Model(&delivery.History{})
	if ownerID > 0 {
		q = q.Where("user_id = ?", ownerID)
	}
	var count int64
	if err := q.Count(&count).Error; err != nil {
		return 0, fmt.Errorf("AttemptRepository.CountHistory: %w", err)
	}
	return count, nil
}

// rowLockingDialect reports whether the active DB dialect supports
// FOR UPDATE SKIP LOCKED. Postgres and MySQL 8.0+ do; SQLite does not (it is a
// parse-time syntax error), and its production pool pins MaxOpenConns(1) so
// the clause is unnecessary anyway.
func (r *AttemptRepository) rowLockingDialect() bool {
	name := r.db.Name()
	return name == "postgres" || name == "mysql"
}
