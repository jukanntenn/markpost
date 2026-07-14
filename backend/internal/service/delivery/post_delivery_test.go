package delivery

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"markpost/internal/config"
	"markpost/internal/domain/delivery"
	domainpost "markpost/internal/domain/post"
	"markpost/internal/domain/user"
	"markpost/internal/infra"
	"markpost/internal/service/delivery/filter"
)

func TestKeywordFilterMatch(t *testing.T) {
	tests := []struct {
		name     string
		title    string
		keywords string
		want     bool
	}{
		{"no keywords matches anything", "Any Title", "", true},
		{"single keyword match", "Server Alert", "alert", true},
		{"single keyword no match", "Server OK", "alert", false},
		{"comma is OR: any keyword matches", "Alert only", "alert,error", true},
		{"comma is OR: none match", "Normal Post", "alert,error", false},
		{"AND via ampersand requires all", "Alert Error Report", "alert & error", true},
		{"AND missing one fails", "Alert only", "alert & error", false},
		{"case insensitive", "ALERT Error", "alert,error", true},
		{"empty title with keywords", "", "alert", false},
		{"whitespace-only keywords matches all", "Title", "   ", true},
		{"keyword with surrounding spaces", "Server Alert", "  alert  ", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := filter.Compile(tt.keywords)
			if err != nil {
				t.Fatalf("filter.Compile(%q) error: %v", tt.keywords, err)
			}
			got := m.Match(tt.title)
			if got != tt.want {
				t.Errorf("keywords=%q title=%q = %v, want %v", tt.keywords, tt.title, got, tt.want)
			}
		})
	}
}

func TestKeywordFilterRejectsInvalid(t *testing.T) {
	for _, raw := range []string{"a,,b", "a && b", "!", "(a", `"abc`} {
		if _, err := filter.Compile(raw); err == nil {
			t.Errorf("expected invalid keywords %q to be rejected", raw)
		}
	}
}

func TestTruncateRunes(t *testing.T) {
	tests := []struct {
		name string
		s    string
		max  int
		want string
	}{
		{"empty string", "", 10, ""},
		{"zero max", "hello", 0, ""},
		{"negative max", "hello", -1, ""},
		{"shorter than max", "hello", 10, "hello"},
		{"equal to max", "hello", 5, "hello"},
		{"longer than max", "hello world", 5, "hello"},
		{"unicode", "你好世界", 2, "你好"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateRunes(tt.s, tt.max)
			if got != tt.want {
				t.Errorf("truncateRunes(%q, %d) = %q, want %q", tt.s, tt.max, got, tt.want)
			}
		})
	}
}

func TestBuildPostURL(t *testing.T) {
	tests := []struct {
		name      string
		publicURL string
		host      string
		port      uint16
		qid       string
		want      string
	}{
		{"with public URL", "https://example.com", "", 0, "p-abc", "https://example.com/p-abc"},
		{"public URL trailing slash", "https://example.com/", "", 0, "p-abc", "https://example.com/p-abc"},
		{"fallback to host:port", "", "192.168.1.1", 8080, "p-abc", "http://192.168.1.1:8080/p-abc"},
		{"fallback 0.0.0.0 to 127.0.0.1", "", "0.0.0.0", 8080, "p-abc", "http://127.0.0.1:8080/p-abc"},
		{"empty host fallback", "", "", 8080, "p-abc", "http://127.0.0.1:8080/p-abc"},
		{"no qid", "https://example.com", "", 0, "", "https://example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildPostURL(tt.publicURL, tt.host, tt.port, tt.qid)
			if got != tt.want {
				t.Errorf("buildPostURL(%q, %q, %d, %q) = %q, want %q", tt.publicURL, tt.host, tt.port, tt.qid, got, tt.want)
			}
		})
	}
}

func TestBuildBodyPreview(t *testing.T) {
	tests := []struct {
		name string
		body string
		max  int
		want string
	}{
		{"empty body", "", 200, ""},
		{"short body", "Hello", 200, "Hello"},
		{"long body truncated", "A very long body that should be truncated", 10, "A very lon…"},
		{"exact length", "Hello", 5, "Hello"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildBodyPreview(tt.body, tt.max)
			if got != tt.want {
				t.Errorf("buildBodyPreview(%q, %d) = %q, want %q", tt.body, tt.max, got, tt.want)
			}
		})
	}
}

func makeFeishuChannelConfig(webhookURL string) delivery.ChannelConfiguration {
	return delivery.ChannelConfiguration{
		"webhook_url":   webhookURL,
		"card_link_url": "",
	}
}

func loadDeliveryTestConfig(t *testing.T) {
	t.Helper()
	config.ResetForTest()

	tmpFile, err := os.CreateTemp("", "test-config-*.toml")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	_, _ = tmpFile.WriteString(`
server.host = "127.0.0.1"
server.port = 7330
[db]
driver = "sqlite"
dsn = ":memory:"
[admin]
initial_username = "markpost"
initial_password = "markpost"
[jwt]
access_signing_key = "test-access-key-at-least-32-characters"
refresh_signing_key = "test-refresh-key-at-least-32-characters"
[delivery]
request_timeout = "5s"
`)
	_ = tmpFile.Close()
	t.Cleanup(func() { _ = os.Remove(tmpFile.Name()) })

	if err := config.Load(tmpFile.Name()); err != nil {
		t.Fatalf("config.Load error: %v", err)
	}
}

func TestPostDeliveryService_Send(t *testing.T) {
	loadDeliveryTestConfig(t)

	t.Run("sends feishu card to webhook", func(t *testing.T) {
		var received bool
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			received = true
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"code":0}`))
		}))
		defer server.Close()

		channel := &delivery.Channel{
			UserID:        1,
			Kind:          delivery.ChannelKindFeishu,
			Name:          "Test",
			Enabled:       true,
			Configuration: makeFeishuChannelConfig(server.URL),
			Keywords:      "",
		}
		p := &domainpost.Post{ID: 1, QID: "p-test", Title: "Test", Body: "Body"}

		svc := &PostDeliveryService{feishu: NewFeishuClient(5 * time.Second)}
		if err := svc.Send(context.Background(), p, channel); err != nil {
			t.Fatalf("Send error: %v", err)
		}
		if !received {
			t.Error("expected feishu webhook to be called")
		}
	})

	t.Run("returns error on feishu failure", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		channel := &delivery.Channel{
			UserID:        1,
			Kind:          delivery.ChannelKindFeishu,
			Configuration: makeFeishuChannelConfig(server.URL),
		}
		p := &domainpost.Post{ID: 1, QID: "p-test", Title: "Test"}

		svc := &PostDeliveryService{feishu: NewFeishuClient(5 * time.Second)}
		if err := svc.Send(context.Background(), p, channel); err == nil {
			t.Fatal("expected error from failed webhook")
		}
	})

	t.Run("returns error for unsupported channel kind", func(t *testing.T) {
		channel := &delivery.Channel{UserID: 1, Kind: "unknown"}
		p := &domainpost.Post{ID: 1, QID: "p-test", Title: "Test"}

		svc := &PostDeliveryService{feishu: NewFeishuClient(5 * time.Second)}
		err := svc.Send(context.Background(), p, channel)
		if err == nil {
			t.Fatal("expected error for unsupported kind")
		}
	})
}

func TestNewPostDeliveryService(t *testing.T) {
	loadDeliveryTestConfig(t)

	svc := NewPostDeliveryService()
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
	if svc.feishu == nil {
		t.Error("expected non-nil feishu client")
	}
}

// seedUserPostChannel inserts a user, a post owned by that user, and a feishu
// delivery channel for the user, returning their IDs for attempt setup.
func seedUserPostChannel(t *testing.T, db *infra.Database) (userID, postID, channelID int) {
	t.Helper()

	u := &user.User{
		Email:    "alice@example.com",
		Username: "alice",
		Password: "x",
		PostKey:  "pk-seed",
	}
	if err := db.DB().Create(u).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}

	p := &domainpost.Post{QID: "p-seed", Title: "Server Alert", Body: "boom", UserID: u.ID}
	if err := db.DB().Create(p).Error; err != nil {
		t.Fatalf("seed post: %v", err)
	}

	ch := &delivery.Channel{
		UserID:        u.ID,
		Kind:          delivery.ChannelKindFeishu,
		Name:          "alerts",
		Enabled:       true,
		Configuration: makeFeishuChannelConfig("https://example.com/webhook"),
		Keywords:      "alert",
	}
	if err := db.DB().Create(ch).Error; err != nil {
		t.Fatalf("seed channel: %v", err)
	}
	return u.ID, p.ID, ch.ID
}

func TestDispatcher_EnqueueInsertsMatchingAttempts(t *testing.T) {
	loadDeliveryTestConfig(t)
	database, err := infra.NewTestDatabase()
	if err != nil {
		t.Fatalf("NewTestDatabase: %v", err)
	}
	defer func() { _ = database.Close() }()

	uid, pid, cid := seedUserPostChannel(t, database)
	// also seed a non-matching channel to confirm it produces no attempt.
	_ = database.DB().Create(&delivery.Channel{
		UserID:        uid,
		Kind:          delivery.ChannelKindFeishu,
		Name:          "no-match",
		Enabled:       true,
		Configuration: makeFeishuChannelConfig("https://example.com/webhook"),
		Keywords:      "nomatch",
	}).Error

	channelRepo := infra.NewDeliveryChannelRepository(database.DB())
	attemptRepo := infra.NewAttemptRepository(database.DB())
	postRepo := infra.NewPostRepository(database.DB())

	dispatcher := NewDispatcher(attemptRepo, channelRepo, postRepo, &recordingSender{})
	dispatcher.Enqueue(domainpost.DeliveryJob{
		UserID:  uid,
		PostID:  pid,
		PostQID: "p-seed",
		Title:   "Server Alert",
		Body:    "boom",
	})

	var attempts []delivery.Attempt
	if err := database.DB().Find(&attempts).Error; err != nil {
		t.Fatalf("find attempts: %v", err)
	}
	if len(attempts) != 1 {
		t.Fatalf("expected 1 matching attempt, got %d", len(attempts))
	}
	a := attempts[0]
	if a.Status != delivery.StatusPending {
		t.Errorf("status = %d, want %d", a.Status, delivery.StatusPending)
	}
	if a.ChannelID != cid {
		t.Errorf("channel_id = %d, want %d", a.ChannelID, cid)
	}
	if a.PostID != pid {
		t.Errorf("post_id = %d, want %d", a.PostID, pid)
	}
}

func TestDispatcher_ClaimExecuteArchive(t *testing.T) {
	loadDeliveryTestConfig(t)
	database, err := infra.NewTestDatabase()
	if err != nil {
		t.Fatalf("NewTestDatabase: %v", err)
	}
	defer func() { _ = database.Close() }()

	uid, pid, cid := seedUserPostChannel(t, database)

	ctx := context.Background()
	now := time.Now()
	attempt := &delivery.Attempt{
		UserID:    uid,
		PostID:    pid,
		ChannelID: cid,
		Status:    delivery.StatusPending,
		NextAt:    now.UnixMilli(),
		CreatedAt: now,
		UpdatedAt: now,
	}
	attemptRepo := infra.NewAttemptRepository(database.DB())
	if err := attemptRepo.Create(ctx, []*delivery.Attempt{attempt}); err != nil {
		t.Fatalf("create attempt: %v", err)
	}

	// Claim returns the one due attempt.
	claimed, err := attemptRepo.ClaimDue(ctx, now.UnixMilli(), now.Add(time.Minute).UnixMilli(), 10)
	if err != nil {
		t.Fatalf("ClaimDue: %v", err)
	}
	if len(claimed) != 1 {
		t.Fatalf("expected 1 claimed, got %d", len(claimed))
	}
	if claimed[0].ID != attempt.ID {
		t.Errorf("claimed id = %d, want %d", claimed[0].ID, attempt.ID)
	}

	// A second claim right away must not re-claim the reserved row.
	again, err := attemptRepo.ClaimDue(ctx, now.UnixMilli(), now.Add(time.Minute).UnixMilli(), 10)
	if err != nil {
		t.Fatalf("ClaimDue second: %v", err)
	}
	if len(again) != 0 {
		t.Errorf("expected 0 re-claimed, got %d", len(again))
	}

	// Archive as delivered: attempt row gone, history row written.
	if err := attemptRepo.ArchiveAndDelete(ctx, claimed[0], delivery.StatusDelivered, ""); err != nil {
		t.Fatalf("ArchiveAndDelete: %v", err)
	}

	var remaining []delivery.Attempt
	database.DB().Find(&remaining)
	if len(remaining) != 0 {
		t.Errorf("expected attempt deleted, %d remain", len(remaining))
	}
	var history []delivery.History
	database.DB().Find(&history)
	if len(history) != 1 || history[0].Status != delivery.StatusDelivered {
		t.Errorf("expected 1 delivered history row, got %+v", history)
	}
}

func TestDispatcher_MarkRetryAdvancesNextAt(t *testing.T) {
	loadDeliveryTestConfig(t)
	database, err := infra.NewTestDatabase()
	if err != nil {
		t.Fatalf("NewTestDatabase: %v", err)
	}
	defer func() { _ = database.Close() }()

	uid, pid, cid := seedUserPostChannel(t, database)
	ctx := context.Background()
	now := time.Now()
	attempt := &delivery.Attempt{
		UserID:    uid,
		PostID:    pid,
		ChannelID: cid,
		Status:    delivery.StatusPending,
		NextAt:    now.UnixMilli(),
		CreatedAt: now,
		UpdatedAt: now,
	}
	attemptRepo := infra.NewAttemptRepository(database.DB())
	if err := attemptRepo.Create(ctx, []*delivery.Attempt{attempt}); err != nil {
		t.Fatalf("create attempt: %v", err)
	}

	wantAt := now.Add(time.Minute).UnixMilli()
	if err := attemptRepo.MarkRetry(ctx, attempt.ID, 1, "boom", wantAt); err != nil {
		t.Fatalf("MarkRetry: %v", err)
	}

	var got delivery.Attempt
	database.DB().First(&got, attempt.ID)
	if got.Attempts != 1 {
		t.Errorf("attempts = %d, want 1", got.Attempts)
	}
	if got.NextAt != wantAt {
		t.Errorf("next_at = %d, want %d", got.NextAt, wantAt)
	}
	if got.LastError != "boom" {
		t.Errorf("last_error = %q, want %q", got.LastError, "boom")
	}
}

type recordingSender struct{}

func (recordingSender) Send(_ context.Context, _ *domainpost.Post, _ *delivery.Channel) error {
	return nil
}
