package infra

import (
	"context"
	"testing"
	"time"

	"markpost/internal/domain/delivery"

	"gorm.io/gorm"
)

// B2.7/K.2：DailyStats / TodayCounts / ListPending 聚合正确性。
func TestAttemptRepository_StatsAndPending(t *testing.T) {
	db := SetupTestDB(t)
	attemptRepo := NewAttemptRepository(db)
	userRepo := NewUserRepository(db, 16)
	postRepo := NewPostRepository(db)
	channelRepo := NewDeliveryChannelRepository(db)
	ctx := context.Background()

	u, _ := userRepo.Create(ctx, "stats@example.com", "statsuser", "correctpass")
	other, _ := userRepo.Create(ctx, "other@example.com", "otheruser", "correctpass")
	p1, _ := postRepo.Create(ctx, "Post A", "Body", u.ID)
	p2, _ := postRepo.Create(ctx, "Post B", "Body", u.ID)
	ch1 := &delivery.Channel{UserID: u.ID, Kind: delivery.ChannelKindFeishu, Name: "Ch1", Enabled: true}
	_ = channelRepo.Create(ctx, ch1)
	ch2 := &delivery.Channel{UserID: u.ID, Kind: delivery.ChannelKindFeishu, Name: "Ch2", Enabled: true}
	_ = channelRepo.Create(ctx, ch2)

	now := time.Now().UTC()

	// 今日：1 delivered + 1 failed；昨日：1 expired。
	today := now
	yesterday := now.AddDate(0, 0, -1)
	archiveHistory(ctx, db, u.ID, p1.ID, ch1.ID, delivery.StatusDelivered, "", today)
	archiveHistory(ctx, db, u.ID, p2.ID, ch2.ID, delivery.StatusFailed, "boom", today)
	archiveHistory(ctx, db, u.ID, p1.ID, ch1.ID, delivery.StatusExpired, "wall", yesterday)

	// TodayCounts：delivered=1, failed=1, pending=0。
	counts, err := attemptRepo.TodayCounts(ctx, u.ID)
	if err != nil {
		t.Fatalf("today counts: %v", err)
	}
	if counts.Delivered != 1 || counts.Failed != 1 || counts.Pending != 0 {
		t.Errorf("unexpected today counts: %+v", counts)
	}

	// DailyStats 7 天：今天 delivered=1 failed=1，昨天 expired=1。
	stats, err := attemptRepo.DailyStats(ctx, u.ID, 7)
	if err != nil {
		t.Fatalf("daily stats: %v", err)
	}
	if len(stats) != 2 {
		t.Fatalf("expected 2 days, got %d: %+v", len(stats), stats)
	}
	byDay := map[string]*delivery.DailyStat{}
	for _, s := range stats {
		byDay[s.Day] = s
	}
	todayKey := today.Format("2006-01-02")
	yesterdayKey := yesterday.Format("2006-01-02")
	if d := byDay[todayKey]; d == nil || d.Delivered != 1 || d.Failed != 1 || d.Expired != 0 {
		t.Errorf("unexpected today stat: %+v", byDay[todayKey])
	}
	if d := byDay[yesterdayKey]; d == nil || d.Expired != 1 {
		t.Errorf("unexpected yesterday stat: %+v", byDay[yesterdayKey])
	}

	// 用户隔离：其他用户看不到。
	statsOther, err := attemptRepo.DailyStats(ctx, other.ID, 7)
	if err != nil {
		t.Fatalf("daily stats other: %v", err)
	}
	if len(statsOther) != 0 {
		t.Errorf("expected no stats for other user, got %d", len(statsOther))
	}

	// Pending：插入 2 条 pending attempts（一条属于 u，一条属于 other）。
	attempts := []*delivery.Attempt{
		{UserID: u.ID, PostID: p1.ID, ChannelID: ch1.ID, Status: delivery.StatusPending, NextAt: 0},
		{UserID: other.ID, PostID: p1.ID, ChannelID: ch1.ID, Status: delivery.StatusPending, NextAt: 0},
	}
	if err := attemptRepo.Create(ctx, attempts); err != nil {
		t.Fatalf("create attempts: %v", err)
	}
	pending, err := attemptRepo.ListPending(ctx, u.ID)
	if err != nil {
		t.Fatalf("list pending: %v", err)
	}
	if len(pending) != 1 {
		t.Fatalf("expected 1 pending attempt, got %d", len(pending))
	}
	if pending[0].PostQID != p1.QID || pending[0].ChannelName != "Ch1" {
		t.Errorf("unexpected pending row: %+v", pending[0])
	}

	// Pending 计入 today 计数。
	counts2, _ := attemptRepo.TodayCounts(ctx, u.ID)
	if counts2.Pending != 1 {
		t.Errorf("expected pending=1, got %d", counts2.Pending)
	}

	// Admin 版 DailyStatsAll 跨用户。
	all, err := attemptRepo.DailyStatsAll(ctx, 7)
	if err != nil {
		t.Fatalf("daily stats all: %v", err)
	}
	if len(all) == 0 {
		t.Error("expected admin stats across users")
	}
}

func archiveHistory(ctx context.Context, db *gorm.DB, userID, postID, channelID int, status delivery.Status, lastError string, createdAt time.Time) {
	h := &delivery.History{
		UserID:    &userID,
		PostID:    &postID,
		ChannelID: &channelID,
		Status:    status,
		LastError: lastError,
		CreatedAt: createdAt,
	}
	if err := db.WithContext(ctx).Create(h).Error; err != nil {
		panic(err)
	}
}
