package infra

import (
	"context"
	"encoding/json"

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

func (r *AuditRepository) List(ctx context.Context, offset, limit int) ([]audit.Log, int64, error) {
	var logs []audit.Log
	var total int64
	if err := r.db.WithContext(ctx).Model(&audit.Log{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := r.db.WithContext(ctx).
		Order("created_at DESC").
		Offset(offset).Limit(limit).
		Find(&logs).Error; err != nil {
		return nil, 0, err
	}
	return logs, total, nil
}
