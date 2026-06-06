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
	"markpost/internal/infra"
	"markpost/internal/service/post"
)

func TestPostTitleMatchesAllKeywords(t *testing.T) {
	tests := []struct {
		name     string
		title    string
		keywords string
		want     bool
	}{
		{"no keywords matches anything", "Any Title", "", true},
		{"single keyword match", "Server Alert", "alert", true},
		{"single keyword no match", "Server OK", "alert", false},
		{"multiple keywords all match", "Alert Error Report", "alert,error", true},
		{"multiple keywords partial match", "Alert only", "alert,error", false},
		{"case insensitive", "ALERT Error", "alert,error", true},
		{"empty title with keywords", "", "alert", false},
		{"whitespace keywords ignored", "Title", "  ,  , ", true},
		{"keyword with spaces", "Server Alert", "  alert  ", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := postTitleMatchesAllKeywords(tt.title, tt.keywords)
			if got != tt.want {
				t.Errorf("postTitleMatchesAllKeywords(%q, %q) = %v, want %v", tt.title, tt.keywords, got, tt.want)
			}
		})
	}
}

func TestParseCommaSeparatedKeywords(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want int
	}{
		{"empty", "", 0},
		{"single", "alert", 1},
		{"multiple", "alert,error,warning", 3},
		{"with spaces", "  alert , error  , warning ", 3},
		{"empty parts", "alert,,error,", 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseCommaSeparatedKeywords(tt.raw)
			if len(got) != tt.want {
				t.Errorf("parseCommaSeparatedKeywords(%q) returned %d items, want %d", tt.raw, len(got), tt.want)
			}
		})
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

func TestPostDeliveryService_Deliver(t *testing.T) {
	t.Run("sends to matching channels", func(t *testing.T) {
		db := infra.SetupTestDB(t)
		repo := infra.NewDeliveryChannelRepository(db)
		ctx := context.Background()

		_ = repo.Create(ctx, &delivery.Channel{
			UserID:        1,
			Kind:          delivery.ChannelKindFeishu,
			Name:          "Alert Channel",
			Enabled:       true,
			Configuration: makeFeishuChannelConfig("https://example.com/webhook"),
			Keywords:      "alert",
		})

		svc := &PostDeliveryService{repo: repo, feishu: NewFeishuClient(5 * time.Second)}
		svc.Deliver(ctx, post.DeliveryJob{
			UserID:  1,
			PostQID: "p-test",
			Title:   "Server Alert",
			Body:    "Something happened",
		})
	})

	t.Run("skips disabled channels", func(t *testing.T) {
		db := infra.SetupTestDB(t)
		repo := infra.NewDeliveryChannelRepository(db)
		ctx := context.Background()

		_ = repo.Create(ctx, &delivery.Channel{
			UserID:        1,
			Kind:          delivery.ChannelKindFeishu,
			Name:          "Disabled",
			Enabled:       false,
			Configuration: makeFeishuChannelConfig("https://example.com/webhook"),
			Keywords:      "",
		})

		svc := &PostDeliveryService{repo: repo, feishu: NewFeishuClient(5 * time.Second)}
		svc.Deliver(ctx, post.DeliveryJob{
			UserID:  1,
			PostQID: "p-test",
			Title:   "Test",
			Body:    "Body",
		})
	})

	t.Run("skips non-matching keywords", func(t *testing.T) {
		db := infra.SetupTestDB(t)
		repo := infra.NewDeliveryChannelRepository(db)
		ctx := context.Background()

		_ = repo.Create(ctx, &delivery.Channel{
			UserID:        1,
			Kind:          delivery.ChannelKindFeishu,
			Name:          "Alert Only",
			Enabled:       true,
			Configuration: makeFeishuChannelConfig("https://example.com/webhook"),
			Keywords:      "alert",
		})

		svc := &PostDeliveryService{repo: repo, feishu: NewFeishuClient(5 * time.Second)}
		svc.Deliver(ctx, post.DeliveryJob{
			UserID:  1,
			PostQID: "p-test",
			Title:   "Normal Post",
			Body:    "Nothing special",
		})
	})

	t.Run("sends card to feishu webhook", func(t *testing.T) {
		var received bool
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			received = true
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"code":0}`))
		}))
		defer server.Close()

		db := infra.SetupTestDB(t)
		repo := infra.NewDeliveryChannelRepository(db)
		ctx := context.Background()

		_ = repo.Create(ctx, &delivery.Channel{
			UserID:        1,
			Kind:          delivery.ChannelKindFeishu,
			Name:          "Test",
			Enabled:       true,
			Configuration: makeFeishuChannelConfig(server.URL),
			Keywords:      "",
		})

		svc := &PostDeliveryService{repo: repo, feishu: NewFeishuClient(5 * time.Second)}
		svc.Deliver(ctx, post.DeliveryJob{
			UserID:  1,
			PostQID: "p-test",
			Title:   "Test",
			Body:    "Body",
		})

		if !received {
			t.Error("expected feishu webhook to be called")
		}
	})
}

func TestNewPostDeliveryService(t *testing.T) {
	config.ResetForTest()

	tmpFile, err := os.CreateTemp("", "test-config-*.toml")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	tmpFile.WriteString(`
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
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	if err := config.Load(tmpFile.Name()); err != nil {
		t.Fatalf("config.Load error: %v", err)
	}

	db := infra.SetupTestDB(t)
	repo := infra.NewDeliveryChannelRepository(db)

	svc := NewPostDeliveryService(repo)
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
	if svc.repo == nil {
		t.Error("expected non-nil repo")
	}
	if svc.feishu == nil {
		t.Error("expected non-nil feishu client")
	}
}

func TestDeliveryDispatcher(t *testing.T) {
	t.Run("enqueues and processes jobs", func(t *testing.T) {
		processed := make(chan post.DeliveryJob, 1)
		mockHandler := &mockDeliveryHandler{processed: processed}

		dispatcher := NewDeliveryDispatcher(mockHandler, 256)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		dispatcher.Start(ctx)

		dispatcher.Enqueue(post.DeliveryJob{
			UserID:  1,
			PostQID: "p-test",
			Title:   "Test",
			Body:    "Body",
		})

		select {
		case job := <-processed:
			if job.Title != "Test" {
				t.Errorf("job title = %q, want %q", job.Title, "Test")
			}
		case <-time.After(time.Second):
			t.Fatal("timeout waiting for job to be processed")
		}
	})

	t.Run("drops jobs when queue is full", func(t *testing.T) {
		dispatcher := NewDeliveryDispatcher(&mockDeliveryHandler{processed: make(chan post.DeliveryJob, 1)}, 1)

		dispatcher.Enqueue(post.DeliveryJob{UserID: 1, PostQID: "p-1", Title: "First"})
		dispatcher.Enqueue(post.DeliveryJob{UserID: 2, PostQID: "p-2", Title: "Second"})

		if len(dispatcher.jobs) != 1 {
			t.Errorf("queue length = %d, want 1", len(dispatcher.jobs))
		}
	})

	t.Run("uses default buffer size when zero", func(t *testing.T) {
		dispatcher := NewDeliveryDispatcher(&mockDeliveryHandler{processed: make(chan post.DeliveryJob)}, 0)
		if cap(dispatcher.jobs) != 256 {
			t.Errorf("buffer size = %d, want 256", cap(dispatcher.jobs))
		}
	})
}

type mockDeliveryHandler struct {
	processed chan post.DeliveryJob
}

func (m *mockDeliveryHandler) Deliver(_ context.Context, job post.DeliveryJob) {
	m.processed <- job
}
