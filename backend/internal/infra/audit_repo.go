package infra

import (
	"context"
	"encoding/json"
	"fmt"

	"markpost/internal/domain/audit"

	"gorm.io/gorm"
)

// AuditRepository implements audit.Repository backed by GORM.
type AuditRepository struct {
	db *gorm.DB
}

func NewAuditRepository(db *gorm.DB) *AuditRepository {
	return &AuditRepository{db: db}
}

func (r *AuditRepository) Record(ctx context.Context, e audit.Entry) error {
	var metadata json.RawMessage
	if e.Metadata != nil {
		b, err := json.Marshal(e.Metadata)
		if err != nil {
			return err
		}
		metadata = b
	}
	log := audit.Log{
		ActorID:    e.ActorID,
		Action:     e.Action,
		TargetType: e.TargetType,
		TargetID:   e.TargetID,
		Metadata:   metadata,
		IP:         e.IP,
	}
	return r.db.WithContext(ctx).Create(&log).Error
}

// applyAuditFilter scopes the query by the filter's optional fields (D4.3).
// Every column is qualified with the "l." alias because List joins users
// (which also has created_at); an unqualified created_at trips SQLSTATE 42702
// (ambiguous column) as soon as a Since/Until filter is applied. Both callers
// build their query from Table("audit_logs AS l"), so the alias always exists.
func applyAuditFilter(q *gorm.DB, filter audit.AuditFilter) *gorm.DB {
	if filter.ActorID > 0 {
		q = q.Where("l.actor_id = ?", filter.ActorID)
	}
	if filter.Action != "" {
		q = q.Where("l.action = ?", filter.Action)
	}
	if filter.TargetType != "" {
		q = q.Where("l.target_type = ?", filter.TargetType)
	}
	if filter.TargetID != "" {
		q = q.Where("l.target_id = ?", filter.TargetID)
	}
	if filter.Since != nil {
		q = q.Where("l.created_at >= ?", filter.Since)
	}
	if filter.Until != nil {
		q = q.Where("l.created_at <= ?", filter.Until)
	}
	return q
}

// List returns audit logs (newest first) matching the filter, joined to the
// actor's username (D4.1), plus the total count for pagination.
func (r *AuditRepository) List(ctx context.Context, filter audit.AuditFilter, offset, limit int) ([]audit.LogRow, int64, error) {
	var total int64
	countQ := applyAuditFilter(r.db.WithContext(ctx).Table("audit_logs AS l"), filter)
	if err := countQ.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("AuditRepository.List count: %w", err)
	}

	rows, err := r.listRows(ctx, filter, offset, limit)
	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (r *AuditRepository) listRows(ctx context.Context, filter audit.AuditFilter, offset, limit int) ([]audit.LogRow, error) {
	// Join users twice: once for the actor, once for the target (only when the
	// target is a user — DEV-1 narratives use the username instead of the raw
	// id). target_id is varchar and may hold non-numeric ids for posts/channels,
	// so the cast is guarded by a regex CASE to avoid invalid-input errors on
	// rows whose target is not a user.
	q := r.db.WithContext(ctx).Table("audit_logs AS l").
		Select("l.*, u.username AS actor_username, tu.username AS target_username").
		Joins("LEFT JOIN users u ON u.id = l.actor_id").
		Joins("LEFT JOIN users tu ON l.target_type = 'user' AND tu.id = CASE WHEN l.target_id ~ '^[0-9]+$' THEN l.target_id::integer ELSE NULL END").
		Order("l.created_at DESC")
	q = applyAuditFilter(q, filter)
	var rows []audit.LogRow
	if err := q.Offset(offset).Limit(limit).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("AuditRepository.List: %w", err)
	}
	return rows, nil
}

// ActionCounts returns the number of log rows per action under the filter
// (D4.3 筛选计数 facets). Only the current filter's scope is applied; the
// action dimension itself is not filtered.
func (r *AuditRepository) ActionCounts(ctx context.Context, filter audit.AuditFilter) (map[string]int64, error) {
	filter.Action = ""
	// Same "l." alias as listRows so applyAuditFilter's qualified columns
	// resolve; ActionCounts needs no join.
	q := applyAuditFilter(r.db.WithContext(ctx).Table("audit_logs AS l"), filter)
	q = q.Select("l.action, COUNT(*) AS count").Group("l.action")
	type row struct {
		Action string `gorm:"column:action"`
		Count  int64  `gorm:"column:count"`
	}
	var rows []row
	if err := q.Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("AuditRepository.ActionCounts: %w", err)
	}
	out := make(map[string]int64, len(rows))
	for _, rw := range rows {
		out[rw.Action] = rw.Count
	}
	return out, nil
}
