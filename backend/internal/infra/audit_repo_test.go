package infra

import (
	"context"
	"encoding/json"
	"strconv"
	"testing"
	"time"

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

		logs, total, err := repo.List(ctx, audit.AuditFilter{}, 0, 10)
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

		_, total, err := repo.List(ctx, audit.AuditFilter{}, 0, 10)
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

		logs, total, err := repo.List(ctx, audit.AuditFilter{}, 0, 10)
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
		logs, total, err := repo.List(ctx, audit.AuditFilter{}, 0, 10)
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

		logs, total, err := repo.List(ctx, audit.AuditFilter{}, 0, 3)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if total != 5 {
			t.Errorf("expected total 5, got %d", total)
		}
		if len(logs) != 3 {
			t.Errorf("expected 3 logs, got %d", len(logs))
		}

		logs2, total2, err := repo.List(ctx, audit.AuditFilter{}, 3, 3)
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
		logs, total, err := repo.List(ctx, audit.AuditFilter{}, 100, 10)
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
		logs, _, err := repo.List(ctx, audit.AuditFilter{}, 0, 10)
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

// TestAuditRepository_TimeFilter is a regression test for the SQLSTATE 42702
// (ambiguous column) bug: List joins users (which also has a created_at), so a
// Since/Until filter on an unqualified created_at exploded. Both the count and
// the row query must qualify the column with the "l." alias.
func TestAuditRepository_TimeFilter(t *testing.T) {
	db := SetupTestDB(t)
	repo := NewAuditRepository(db)
	ctx := context.Background()

	db.Exec("DELETE FROM audit_logs")

	now := time.Now().UTC()
	past := now.Add(-2 * time.Hour)
	future := now.Add(2 * time.Hour)

	for range 3 {
		if err := repo.Record(ctx, audit.Entry{
			ActorID:    1,
			Action:     "user.set_active",
			TargetType: "user",
			TargetID:   "1",
		}); err != nil {
			t.Fatalf("Record: %v", err)
		}
	}

	t.Run("Since filter does not 500", func(t *testing.T) {
		logs, total, err := repo.List(ctx, audit.AuditFilter{Since: &past}, 0, 10)
		if err != nil {
			t.Fatalf("List with Since returned error (regression): %v", err)
		}
		if total != 3 {
			t.Errorf("total = %d, want 3", total)
		}
		if len(logs) != 3 {
			t.Errorf("len = %d, want 3", len(logs))
		}
	})

	t.Run("Until filter does not 500", func(t *testing.T) {
		logs, total, err := repo.List(ctx, audit.AuditFilter{Until: &future}, 0, 10)
		if err != nil {
			t.Fatalf("List with Until returned error (regression): %v", err)
		}
		if total != 3 {
			t.Errorf("total = %d, want 3", total)
		}
		if len(logs) != 3 {
			t.Errorf("len = %d, want 3", len(logs))
		}
	})

	t.Run("Since in the future excludes everything", func(t *testing.T) {
		logs, total, err := repo.List(ctx, audit.AuditFilter{Since: &future}, 0, 10)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if total != 0 || len(logs) != 0 {
			t.Errorf("expected 0, got total=%d len=%d", total, len(logs))
		}
	})

	t.Run("ActionCounts honors time filter", func(t *testing.T) {
		counts, err := repo.ActionCounts(ctx, audit.AuditFilter{Since: &past})
		if err != nil {
			t.Fatalf("ActionCounts with Since returned error (regression): %v", err)
		}
		if counts["user.set_active"] != 3 {
			t.Errorf("action count = %d, want 3", counts["user.set_active"])
		}
	})
}

// TestAuditRepository_TargetUsername verifies the DEV-1 target-username JOIN:
// user-targeted rows resolve to the target's username, while non-user targets
// leave TargetUsername nil.
func TestAuditRepository_TargetUsername(t *testing.T) {
	db := SetupTestDB(t)
	repo := NewAuditRepository(db)
	ctx := context.Background()

	db.Exec("DELETE FROM audit_logs")

	actor := createTestUser(t, db, 10)  // username "user10"
	target := createTestUser(t, db, 11) // username "user11"

	if err := repo.Record(ctx, audit.Entry{
		ActorID: actor, Action: "user.set_active", TargetType: "user", TargetID: strconv.Itoa(target),
	}); err != nil {
		t.Fatalf("Record user-target: %v", err)
	}
	if err := repo.Record(ctx, audit.Entry{
		ActorID: actor, Action: "post.delete", TargetType: "post", TargetID: "mpk-xyz",
	}); err != nil {
		t.Fatalf("Record post-target: %v", err)
	}

	logs, _, err := repo.List(ctx, audit.AuditFilter{}, 0, 10)
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	var userRow, postRow *audit.LogRow
	for i := range logs {
		switch logs[i].Action {
		case "user.set_active":
			userRow = &logs[i]
		case "post.delete":
			postRow = &logs[i]
		}
	}
	if userRow == nil || postRow == nil {
		t.Fatalf("missing rows: userRow=%v postRow=%v", userRow, postRow)
	}

	if userRow.TargetUsername == nil || *userRow.TargetUsername != "user11" {
		got := "<nil>"
		if userRow.TargetUsername != nil {
			got = *userRow.TargetUsername
		}
		t.Errorf("user-target TargetUsername = %q, want %q", got, "user11")
	}
	if postRow.TargetUsername != nil {
		t.Errorf("post-target TargetUsername = %v, want nil", *postRow.TargetUsername)
	}
}
