// Package audit defines the audit log domain model.
package audit

import (
	"context"
	"encoding/json"
	"time"
)

// Log is an immutable record of an admin write operation.
type Log struct {
	ID         int64           `json:"id" gorm:"primaryKey;autoIncrement"`
	ActorID    int             `json:"actor_id" gorm:"not null;index"`
	Action     string          `json:"action" gorm:"not null;size:64"`
	TargetType string          `json:"target_type" gorm:"not null;size:32"`
	TargetID   string          `json:"target_id" gorm:"size:64"`
	Metadata   json.RawMessage `json:"metadata" gorm:"not null;type:jsonb;default:'{}'::jsonb"`
	IP         string          `json:"ip" gorm:"size:45"`
	CreatedAt  time.Time       `json:"created_at" gorm:"autoCreateTime"`
}

func (Log) TableName() string { return "audit_logs" }

// LogRow is the read projection of an audit log row joined to the actor's
// username at read time (D4.1 — AdminAuditLogItem 补 actor_username).
// TargetUsername is non-nil only when TargetType == "user" and the target user
// still exists (DEV-1 — narratives prefer the username over the raw id).
type LogRow struct {
	Log
	ActorUsername  string  `json:"actor_username" gorm:"column:actor_username"`
	TargetUsername *string `json:"target_username" gorm:"column:target_username"`
}

// AuditFilter scopes an audit log read (D4.3). All fields optional.
type AuditFilter struct {
	ActorID    int
	Action     string
	TargetType string
	TargetID   string
	Since      *time.Time
	Until      *time.Time
}

// Entry is the input for recording an audit log.
type Entry struct {
	ActorID    int
	Action     string // e.g. "user.delete", "post.delete", "channel.update"
	TargetType string // "user" | "post" | "channel" | "session"
	TargetID   string
	Metadata   map[string]any
	IP         string
}

// Repository defines the interface for audit log data access.
type Repository interface {
	Record(ctx context.Context, e Entry) error
	// List returns audit logs (newest first) matching the filter, with the
	// actor username joined, plus the total count for pagination.
	List(ctx context.Context, filter AuditFilter, offset, limit int) ([]LogRow, int64, error)
	// ActionCounts returns the number of log rows per action under the filter
	// (D4.3 筛选计数 facets).
	ActionCounts(ctx context.Context, filter AuditFilter) (map[string]int64, error)
}
