package infra

import (
	"context"
	"testing"
	"time"

	"markpost/internal/domain/delivery"
	"markpost/internal/domain/post"
	"markpost/internal/domain/user"
)

func setupAttemptRepoTestDB(t *testing.T) (*AttemptRepository, []delivery.Attempt) {
	t.Helper()
	db := SetupTestDB(t)
	repo := NewAttemptRepository(db).(*AttemptRepository)

	u := &user.User{Email: "a@b.c", Username: "a", Password: "x", PostKey: "pk"}
	if err := db.Create(u).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	p := &post.Post{QID: "p-1", Title: "t", Body: "b", UserID: u.ID}
	if err := db.Create(p).Error; err != nil {
		t.Fatalf("seed post: %v", err)
	}
	ch := &delivery.Channel{UserID: u.ID, Kind: delivery.ChannelKindFeishu, Name: "c", Configuration: delivery.ChannelConfiguration{}}
	if err := db.Create(ch).Error; err != nil {
		t.Fatalf("seed channel: %v", err)
	}

	now := time.Now()
	old := now.Add(-2 * time.Hour)
	attempts := []delivery.Attempt{
		{UserID: u.ID, PostID: p.ID, ChannelID: ch.ID, Status: delivery.StatusPending, NextAt: now.UnixMilli(), CreatedAt: old, UpdatedAt: old},
		{UserID: u.ID, PostID: p.ID, ChannelID: ch.ID, Status: delivery.StatusPending, NextAt: now.UnixMilli(), CreatedAt: now, UpdatedAt: now},
		{UserID: u.ID, PostID: p.ID, ChannelID: ch.ID, Status: delivery.StatusDelivered, NextAt: now.UnixMilli(), CreatedAt: now, UpdatedAt: now},
	}
	if err := db.Create(&attempts).Error; err != nil {
		t.Fatalf("seed attempts: %v", err)
	}
	return repo, attempts
}

func TestAttemptRepository_MarkExpiredBatchesOldPending(t *testing.T) {
	repo, attempts := setupAttemptRepoTestDB(t)
	ctx := context.Background()

	// Force the first attempt's created_at into the past via raw SQL — GORM's
	// autoCreateTime populates created_at on Create and autoUpdateTime on
	// Updates, so a struct/Update path cannot set a historical timestamp.
	old := time.Now().Add(-2 * time.Hour)
	if err := repo.db.Exec(
		"UPDATE delivery_attempts SET created_at = ? WHERE id = ?",
		old, attempts[0].ID,
	).Error; err != nil {
		t.Fatalf("set old created_at: %v", err)
	}

	wallBefore := time.Now().Add(-1 * time.Hour).UnixMilli()
	expired, err := repo.MarkExpired(ctx, wallBefore, 64)
	if err != nil {
		t.Fatalf("MarkExpired: %v", err)
	}
	if len(expired) != 1 {
		t.Fatalf("expected 1 expired, got %d", len(expired))
	}
	if expired[0].ID != attempts[0].ID {
		t.Errorf("expired id = %d, want %d (the old pending row)", expired[0].ID, attempts[0].ID)
	}

	var got delivery.Attempt
	repo.db.First(&got, attempts[0].ID)
	if got.Status != delivery.StatusExpired {
		t.Errorf("status = %d, want %d (expired)", got.Status, delivery.StatusExpired)
	}
}

func TestAttemptRepository_CountByStatus(t *testing.T) {
	repo, _ := setupAttemptRepoTestDB(t)
	counts, err := repo.CountByStatus(context.Background())
	if err != nil {
		t.Fatalf("CountByStatus: %v", err)
	}
	if counts[delivery.StatusPending] != 2 {
		t.Errorf("pending = %d, want 2", counts[delivery.StatusPending])
	}
	if counts[delivery.StatusDelivered] != 1 {
		t.Errorf("delivered = %d, want 1", counts[delivery.StatusDelivered])
	}
}

func TestAttemptRepository_PruneHistorySubqueryLimit(t *testing.T) {
	db := SetupTestDB(t)
	repo := NewAttemptRepository(db).(*AttemptRepository)

	old := time.Now().Add(-2 * 24 * time.Hour)
	recent := time.Now()
	// Insert via raw SQL so created_at holds a historical value (GORM's
	// autoCreateTime would otherwise stamp created_at = now). user_id/post_id/
	// channel_id are nullable, so the rows need no FK targets.
	rows := []struct {
		status delivery.Status
		when   time.Time
	}{
		{delivery.StatusDelivered, old},
		{delivery.StatusFailed, old},
		{delivery.StatusDelivered, recent},
	}
	for _, r := range rows {
		if err := db.Exec(
			"INSERT INTO delivery_history (status, last_error, created_at) VALUES (?, '', ?)",
			r.status, r.when,
		).Error; err != nil {
			t.Fatalf("seed history: %v", err)
		}
	}

	deleted, err := repo.PruneHistory(context.Background(), 24*time.Hour, 1000)
	if err != nil {
		t.Fatalf("PruneHistory: %v", err)
	}
	if deleted != 2 {
		t.Errorf("deleted = %d, want 2", deleted)
	}

	var remaining []delivery.History
	db.Find(&remaining)
	if len(remaining) != 1 {
		t.Errorf("remaining = %d, want 1", len(remaining))
	}
}

func TestAttemptRepository_ClaimDueDialect(t *testing.T) {
	repo, _ := setupAttemptRepoTestDB(t)
	name := repo.db.Dialector.Name()
	rl := repo.rowLockingDialect()
	if name == "sqlite" {
		if rl {
			t.Error("sqlite should NOT advertise row locking")
		}
	} else if !rl {
		t.Errorf("dialect %s should advertise row locking", name)
	}
}

func TestAttemptRepository_ListHistoryJoinsAndNulls(t *testing.T) {
	db := SetupTestDB(t)
	repo := NewAttemptRepository(db).(*AttemptRepository)
	ctx := context.Background()

	u := &user.User{Email: "hist@b.c", Username: "histuser", Password: "x", PostKey: "hpk"}
	if err := db.Create(u).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	p := &post.Post{QID: "p-hist", Title: "History Post", Body: "b", UserID: u.ID}
	if err := db.Create(p).Error; err != nil {
		t.Fatalf("seed post: %v", err)
	}
	ch := &delivery.Channel{UserID: u.ID, Kind: delivery.ChannelKindFeishu, Name: "hist-channel", Configuration: delivery.ChannelConfiguration{}}
	if err := db.Create(ch).Error; err != nil {
		t.Fatalf("seed channel: %v", err)
	}

	insertHistory := func(postID, channelID, userID *int, status delivery.Status) {
		if err := db.Exec(
			"INSERT INTO delivery_history (status, last_error, post_id, channel_id, user_id) VALUES (?, '', ?, ?, ?)",
			status, postID, channelID, userID,
		).Error; err != nil {
			t.Fatalf("seed history: %v", err)
		}
	}

	pid, chid := p.ID, ch.ID
	// Row 1: all references alive (full JOIN).
	insertHistory(&pid, &chid, &u.ID, delivery.StatusDelivered)
	// Row 2: post deleted (post_id nulled after a delete → simulate directly).
	insertHistory(nil, &chid, &u.ID, delivery.StatusFailed)

	// User-scoped list.
	rows, err := repo.ListHistory(ctx, delivery.HistoryFilter{OwnerID: u.ID}, 0, 50)
	if err != nil {
		t.Fatalf("ListHistory: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}

	// Newest first; find the fully-joined row (post_title set).
	var full, partial *delivery.HistoryRow
	for i := range rows {
		if rows[i].PostTitle != nil {
			full = rows[i]
		} else {
			partial = rows[i]
		}
	}
	if full == nil {
		t.Fatal("expected a row with joined post title")
	}
	if full.PostTitle == nil || *full.PostTitle != "History Post" {
		t.Errorf("post_title = %v, want \"History Post\"", full.PostTitle)
	}
	if full.PostQID == nil || *full.PostQID != "p-hist" {
		t.Errorf("post_qid = %v, want \"p-hist\"", full.PostQID)
	}
	if full.ChannelName == nil || *full.ChannelName != "hist-channel" {
		t.Errorf("channel_name = %v, want \"hist-channel\"", full.ChannelName)
	}
	if full.Username == nil || *full.Username != "histuser" {
		t.Errorf("username = %v, want \"histuser\"", full.Username)
	}
	if full.Status != delivery.StatusDelivered {
		t.Errorf("status = %d, want %d", full.Status, delivery.StatusDelivered)
	}

	if partial == nil {
		t.Fatal("expected a row with a null post reference")
	}
	if partial.PostTitle != nil || partial.PostQID != nil {
		t.Errorf("deleted-post row should have nil post fields, got title=%v qid=%v", partial.PostTitle, partial.PostQID)
	}
	// Channel is still alive on the partial row.
	if partial.ChannelName == nil || *partial.ChannelName != "hist-channel" {
		t.Errorf("channel_name = %v, want \"hist-channel\"", partial.ChannelName)
	}
	if partial.Status != delivery.StatusFailed {
		t.Errorf("status = %d, want %d", partial.Status, delivery.StatusFailed)
	}

	// Count matches the list.
	count, err := repo.CountHistory(ctx, delivery.HistoryFilter{OwnerID: u.ID})
	if err != nil {
		t.Fatalf("CountHistory: %v", err)
	}
	if count != 2 {
		t.Errorf("CountHistory = %d, want 2", count)
	}
}

func TestAttemptRepository_ListHistoryAdminViewIncludesAnonymized(t *testing.T) {
	db := SetupTestDB(t)
	repo := NewAttemptRepository(db).(*AttemptRepository)
	ctx := context.Background()

	u := &user.User{Email: "anon@b.c", Username: "anonuser", Password: "x", PostKey: "apk"}
	if err := db.Create(u).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	// A history row owned by this user.
	if err := db.Exec(
		"INSERT INTO delivery_history (status, last_error, user_id) VALUES (?, '', ?)",
		delivery.StatusDelivered, u.ID,
	).Error; err != nil {
		t.Fatalf("seed owned history: %v", err)
	}
	// An anonymized row (user deleted → user_id NULL).
	if err := db.Exec(
		"INSERT INTO delivery_history (status, last_error, user_id) VALUES (?, '', NULL)",
		delivery.StatusExpired,
	).Error; err != nil {
		t.Fatalf("seed anonymized history: %v", err)
	}

	// User-scoped: only the owned row.
	userRows, err := repo.ListHistory(ctx, delivery.HistoryFilter{OwnerID: u.ID}, 0, 50)
	if err != nil {
		t.Fatalf("ListHistory(user): %v", err)
	}
	if len(userRows) != 1 {
		t.Errorf("user-scoped rows = %d, want 1 (anonymized excluded)", len(userRows))
	}

	// Admin view (zero filter): both rows including the anonymized one.
	adminRows, err := repo.ListHistory(ctx, delivery.HistoryFilter{}, 0, 50)
	if err != nil {
		t.Fatalf("ListHistory(admin): %v", err)
	}
	if len(adminRows) != 2 {
		t.Errorf("admin rows = %d, want 2 (anonymized included)", len(adminRows))
	}

	adminCount, err := repo.CountHistory(ctx, delivery.HistoryFilter{})
	if err != nil {
		t.Fatalf("CountHistory(admin): %v", err)
	}
	if adminCount != 2 {
		t.Errorf("admin CountHistory = %d, want 2", adminCount)
	}
}

func TestAttemptRepository_ListHistoryByChannel(t *testing.T) {
	db := SetupTestDB(t)
	repo := NewAttemptRepository(db).(*AttemptRepository)
	ctx := context.Background()

	u := &user.User{Email: "ch@b.c", Username: "chuser", Password: "x", PostKey: "cpk"}
	if err := db.Create(u).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	ch1 := &delivery.Channel{UserID: u.ID, Kind: delivery.ChannelKindFeishu, Name: "ch1", Configuration: delivery.ChannelConfiguration{}}
	ch2 := &delivery.Channel{UserID: u.ID, Kind: delivery.ChannelKindFeishu, Name: "ch2", Configuration: delivery.ChannelConfiguration{}}
	if err := db.Create([]*delivery.Channel{ch1, ch2}).Error; err != nil {
		t.Fatalf("seed channels: %v", err)
	}

	insertHistory := func(channelID int, status delivery.Status) {
		chid := channelID
		if err := db.Exec(
			"INSERT INTO delivery_history (status, last_error, user_id, channel_id) VALUES (?, '', ?, ?)",
			status, u.ID, chid,
		).Error; err != nil {
			t.Fatalf("seed history: %v", err)
		}
	}
	insertHistory(ch1.ID, delivery.StatusDelivered)
	insertHistory(ch1.ID, delivery.StatusFailed)
	insertHistory(ch2.ID, delivery.StatusExpired)

	// Filter by ch1: only the two ch1 rows.
	rows, err := repo.ListHistory(ctx, delivery.HistoryFilter{OwnerID: u.ID, ChannelID: ch1.ID}, 0, 50)
	if err != nil {
		t.Fatalf("ListHistory(ch1): %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("ch1 rows = %d, want 2", len(rows))
	}
	for _, r := range rows {
		if r.ChannelName == nil || *r.ChannelName != "ch1" {
			t.Errorf("channel_name = %v, want ch1", r.ChannelName)
		}
	}

	// Count agrees.
	ch1Count, err := repo.CountHistory(ctx, delivery.HistoryFilter{OwnerID: u.ID, ChannelID: ch1.ID})
	if err != nil {
		t.Fatalf("CountHistory(ch1): %v", err)
	}
	if ch1Count != 2 {
		t.Errorf("ch1 count = %d, want 2", ch1Count)
	}

	// Filter by ch2: only the one ch2 row.
	ch2Rows, err := repo.ListHistory(ctx, delivery.HistoryFilter{OwnerID: u.ID, ChannelID: ch2.ID}, 0, 50)
	if err != nil {
		t.Fatalf("ListHistory(ch2): %v", err)
	}
	if len(ch2Rows) != 1 {
		t.Fatalf("ch2 rows = %d, want 1", len(ch2Rows))
	}

	// No channel filter: all three rows.
	allRows, err := repo.ListHistory(ctx, delivery.HistoryFilter{OwnerID: u.ID}, 0, 50)
	if err != nil {
		t.Fatalf("ListHistory(all): %v", err)
	}
	if len(allRows) != 3 {
		t.Errorf("all rows = %d, want 3", len(allRows))
	}
}
