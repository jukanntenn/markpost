package infra

import (
	"context"
	"encoding/json"
	"testing"

	"markpost/internal/domain/audit"
)

func TestAuditRepository_Record(t *testing.T) {
	db := SetupTestDB(t)
	repo := NewAuditRepository(db)
	ctx := context.Background()

	t.Run("records audit log with metadata", func(t *testing.T) {
		entry := audit.Entry{
			ActorID:    1,
			Action:     "user.create",
			TargetType: "user",
			TargetID:   "42",
			Metadata:   map[string]any{"role": "admin"},
			IP:         "127.0.0.1",
		}
		if err := repo.Record(ctx, entry); err != nil {
			t.Fatalf("Record: %v", err)
		}

		logs, total, err := repo.List(ctx, 0, 10)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if total != 1 {
			t.Errorf("expected 1 log, got %d", total)
		}
		if len(logs) != 1 {
			t.Fatalf("expected 1 log entry, got %d", len(logs))
		}
		log := logs[0]
		if log.ActorID != 1 {
			t.Errorf("ActorID = %d, want 1", log.ActorID)
		}
		if log.Action != "user.create" {
			t.Errorf("Action = %q, want %q", log.Action, "user.create")
		}
		if log.TargetType != "user" {
			t.Errorf("TargetType = %q, want %q", log.TargetType, "user")
		}
		if log.TargetID != "42" {
			t.Errorf("TargetID = %q, want %q", log.TargetID, "42")
		}
		if log.IP != "127.0.0.1" {
			t.Errorf("IP = %q, want %q", log.IP, "127.0.0.1")
		}
		var meta map[string]any
		if err := json.Unmarshal(log.Metadata, &meta); err != nil {
			t.Fatalf("failed to unmarshal metadata: %v", err)
		}
		if meta["role"] != "admin" {
			t.Errorf("metadata.role = %v, want %q", meta["role"], "admin")
		}
	})

	t.Run("records audit log without metadata", func(t *testing.T) {
		entry := audit.Entry{
			ActorID:    2,
			Action:     "user.delete",
			TargetType: "user",
			TargetID:   "99",
		}
		if err := repo.Record(ctx, entry); err != nil {
			t.Fatalf("Record: %v", err)
		}

		_, total, err := repo.List(ctx, 0, 10)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if total != 2 {
			t.Errorf("expected 2 logs, got %d", total)
		}
	})

	t.Run("records audit log with empty metadata map", func(t *testing.T) {
		entry := audit.Entry{
			ActorID:    3,
			Action:     "post.delete",
			TargetType: "post",
			TargetID:   "1",
			Metadata:   map[string]any{},
		}
		if err := repo.Record(ctx, entry); err != nil {
			t.Fatalf("Record: %v", err)
		}

		logs, total, err := repo.List(ctx, 0, 10)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if total != 3 {
			t.Errorf("expected 3 logs, got %d", total)
		}
		var meta map[string]any
		if err := json.Unmarshal(logs[0].Metadata, &meta); err != nil {
			t.Fatalf("failed to unmarshal metadata: %v", err)
		}
		if len(meta) != 0 {
			t.Errorf("expected empty metadata, got %v", meta)
		}
	})
}

func TestAuditRepository_List(t *testing.T) {
	db := SetupTestDB(t)
	repo := NewAuditRepository(db)
	ctx := context.Background()

	// Clean up any existing audit logs from other tests sharing the same DB.
	db.Exec("DELETE FROM audit_logs")

	t.Run("returns empty list when no logs", func(t *testing.T) {
		logs, total, err := repo.List(ctx, 0, 10)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if total != 0 {
			t.Errorf("expected 0, got %d", total)
		}
		if len(logs) != 0 {
			t.Errorf("expected 0 logs, got %d", len(logs))
		}
	})

	t.Run("paginates correctly", func(t *testing.T) {
		for i := range 5 {
			_ = repo.Record(ctx, audit.Entry{
				ActorID:    1,
				Action:     "test.action",
				TargetType: "test",
				TargetID:   string(rune('A' + i)),
			})
		}

		logs, total, err := repo.List(ctx, 0, 3)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if total != 5 {
			t.Errorf("expected total 5, got %d", total)
		}
		if len(logs) != 3 {
			t.Errorf("expected 3 logs, got %d", len(logs))
		}

		logs2, total2, err := repo.List(ctx, 3, 3)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if total2 != 5 {
			t.Errorf("expected total 5, got %d", total2)
		}
		if len(logs2) != 2 {
			t.Errorf("expected 2 logs, got %d", len(logs2))
		}
	})

	t.Run("returns empty when offset exceeds total", func(t *testing.T) {
		logs, total, err := repo.List(ctx, 100, 10)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if total != 5 {
			t.Errorf("expected total 5, got %d", total)
		}
		if len(logs) != 0 {
			t.Errorf("expected 0 logs, got %d", len(logs))
		}
	})

	t.Run("orders by created_at DESC", func(t *testing.T) {
		logs, _, err := repo.List(ctx, 0, 10)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		for i := 1; i < len(logs); i++ {
			if logs[i-1].CreatedAt.Before(logs[i].CreatedAt) {
				t.Errorf("logs not ordered DESC: log[%d].CreatedAt=%v < log[%d].CreatedAt=%v",
					i-1, logs[i-1].CreatedAt, i, logs[i].CreatedAt)
			}
		}
	})
}
