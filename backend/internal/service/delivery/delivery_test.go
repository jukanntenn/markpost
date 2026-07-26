package delivery

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"markpost/internal/domain/delivery"
	"markpost/internal/infra"
	"markpost/internal/service"

	"gorm.io/gorm"
)

func createTestUser(t *testing.T, db *gorm.DB, idx ...int) int {
	t.Helper()
	i := 0
	if len(idx) > 0 {
		i = idx[0]
	}
	userRepo := infra.NewUserRepository(db, 16)
	u, err := userRepo.Create(context.Background(), fmt.Sprintf("user%d@example.com", i), fmt.Sprintf("user%d", i), "password")
	if err != nil {
		t.Fatalf("createTestUser(%d): %v", i, err)
	}
	return u.ID
}

func setupDeliveryService(t *testing.T) (*Service, delivery.Repository, *gorm.DB) {
	t.Helper()
	db := infra.SetupTestDB(t)
	repo := infra.NewDeliveryChannelRepository(db)
	svc := NewService(repo, nil)
	return svc, repo, db
}

// setupDeliveryServiceWithHistory builds a Service whose attemptRepo is wired
// to the same in-memory DB, so LatestPerChannel can be exercised end-to-end.
func setupDeliveryServiceWithHistory(t *testing.T) (*Service, delivery.Repository, delivery.AttemptRepository, *gorm.DB) {
	t.Helper()
	db := infra.SetupTestDB(t)
	repo := infra.NewDeliveryChannelRepository(db)
	attemptRepo := infra.NewAttemptRepository(db)
	svc := NewService(repo, attemptRepo)
	return svc, repo, attemptRepo, db
}

func feishuConfigJSON(webhookURL, cardLinkURL string) json.RawMessage {
	cfg := delivery.ChannelConfiguration{
		"webhook_url":   webhookURL,
		"card_link_url": cardLinkURL,
	}
	b, _ := json.Marshal(cfg)
	return b
}

func TestService_ListByUserID(t *testing.T) {
	svc, repo, db := setupDeliveryService(t)
	ctx := context.Background()
	uid1 := createTestUser(t, db, 0)
	uid2 := createTestUser(t, db, 1)

	_ = repo.Create(ctx, &delivery.Channel{UserID: uid1, Kind: delivery.ChannelKindFeishu, Name: "Ch1", Configuration: delivery.ChannelConfiguration{"webhook_url": "https://a.com", "card_link_url": ""}})
	_ = repo.Create(ctx, &delivery.Channel{UserID: uid1, Kind: delivery.ChannelKindFeishu, Name: "Ch2", Configuration: delivery.ChannelConfiguration{"webhook_url": "https://b.com", "card_link_url": ""}})
	_ = repo.Create(ctx, &delivery.Channel{UserID: uid2, Kind: delivery.ChannelKindFeishu, Name: "Ch3", Configuration: delivery.ChannelConfiguration{"webhook_url": "https://c.com", "card_link_url": ""}})

	channels, err := svc.ListByUserID(ctx, uid1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(channels) != 2 {
		t.Errorf("got %d channels, want 2", len(channels))
	}
}

func TestService_Create(t *testing.T) {
	svc, _, db := setupDeliveryService(t)
	ctx := context.Background()
	uid := createTestUser(t, db)

	t.Run("creates channel successfully", func(t *testing.T) {
		ch, err := svc.Create(ctx, uid, UpdateChannelParams{
			Kind:          "feishu",
			Name:          "My Channel",
			Configuration: feishuConfigJSON("https://example.com/webhook", ""),
			Keywords:      ptrTo("alert,error"),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ch.Name != "My Channel" {
			t.Errorf("name = %q, want %q", ch.Name, "My Channel")
		}
		if ch.Kind != delivery.ChannelKindFeishu {
			t.Errorf("kind = %q, want %q", ch.Kind, delivery.ChannelKindFeishu)
		}
		if !ch.Enabled {
			t.Error("expected channel to be enabled")
		}
		feishu := ch.Configuration.Feishu()
		if feishu.WebhookURL != "https://example.com/webhook" {
			t.Errorf("webhook_url = %q, want %q", feishu.WebhookURL, "https://example.com/webhook")
		}
	})

	t.Run("creates channel with card_link_url", func(t *testing.T) {
		ch, err := svc.Create(ctx, uid, UpdateChannelParams{
			Kind:          "feishu",
			Name:          "Card Link Channel",
			Configuration: feishuConfigJSON("https://example.com/webhook", "https://custom.example.com/{{.QID}}"),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		feishu := ch.Configuration.Feishu()
		if feishu.CardLinkURL != "https://custom.example.com/{{.QID}}" {
			t.Errorf("card_link_url = %q, want %q", feishu.CardLinkURL, "https://custom.example.com/{{.QID}}")
		}
	})

	t.Run("rejects empty name", func(t *testing.T) {
		_, err := svc.Create(ctx, uid, UpdateChannelParams{Kind: "feishu", Configuration: feishuConfigJSON("https://example.com", "")})
		if err == nil {
			t.Fatal("expected error for empty name")
		}
		se, _ := service.AsError(err)
		if se.Code != service.ErrValidation {
			t.Errorf("expected code %q, got %q", service.ErrValidation.Value, se.Code.Value)
		}
	})

	t.Run("rejects empty configuration", func(t *testing.T) {
		_, err := svc.Create(ctx, uid, UpdateChannelParams{Kind: "feishu", Name: "Test"})
		if err == nil {
			t.Fatal("expected error for empty configuration")
		}
	})

	t.Run("rejects missing webhook URL", func(t *testing.T) {
		_, err := svc.Create(ctx, uid, UpdateChannelParams{
			Kind:          "feishu",
			Name:          "Test",
			Configuration: feishuConfigJSON("", ""),
		})
		if err == nil {
			t.Fatal("expected error for empty webhook URL")
		}
	})

	t.Run("rejects invalid webhook URL", func(t *testing.T) {
		_, err := svc.Create(ctx, uid, UpdateChannelParams{
			Kind:          "feishu",
			Name:          "Test",
			Configuration: feishuConfigJSON("ftp://invalid", ""),
		})
		if err == nil {
			t.Fatal("expected error for invalid webhook URL")
		}
	})

	t.Run("rejects invalid configuration JSON", func(t *testing.T) {
		_, err := svc.Create(ctx, uid, UpdateChannelParams{
			Kind:          "feishu",
			Name:          "Test",
			Configuration: json.RawMessage(`{invalid`),
		})
		if err == nil {
			t.Fatal("expected error for invalid JSON")
		}
	})

	t.Run("rejects unsupported kind", func(t *testing.T) {
		_, err := svc.Create(ctx, uid, UpdateChannelParams{Kind: "slack", Name: "Test", Configuration: feishuConfigJSON("https://example.com", "")})
		if err == nil {
			t.Fatal("expected error for unsupported kind")
		}
		se, _ := service.AsError(err)
		if se.Code != service.ErrValidation {
			t.Errorf("expected code %q, got %q", service.ErrValidation.Value, se.Code.Value)
		}
	})
}

func TestService_Update(t *testing.T) {
	svc, repo, db := setupDeliveryService(t)
	ctx := context.Background()
	uid := createTestUser(t, db)

	ch := &delivery.Channel{UserID: uid, Kind: delivery.ChannelKindFeishu, Name: "Old", Configuration: delivery.ChannelConfiguration{"webhook_url": "https://old.com", "card_link_url": ""}, Keywords: "old"}
	_ = repo.Create(ctx, ch)

	t.Run("updates channel name successfully", func(t *testing.T) {
		newName := "New Name"
		ch, err := svc.Update(ctx, uid, ch.ID, UpdateChannelParams{Name: newName})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ch.Name != "New Name" {
			t.Errorf("name = %q, want %q", ch.Name, "New Name")
		}
	})

	t.Run("returns error for non-existent channel", func(t *testing.T) {
		_, err := svc.Update(ctx, uid, 999, UpdateChannelParams{Name: "New"})
		if err == nil {
			t.Fatal("expected error for non-existent channel")
		}
		se, _ := service.AsError(err)
		if se.Code != service.ErrNotFound {
			t.Errorf("expected code %q, got %q", service.ErrNotFound.Value, se.Code.Value)
		}
	})

	t.Run("updates configuration", func(t *testing.T) {
		ch, err := svc.Update(ctx, uid, ch.ID, UpdateChannelParams{
			Configuration: feishuConfigJSON("https://new.com/webhook", "https://custom.com/{{.QID}}"),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		feishu := ch.Configuration.Feishu()
		if feishu.WebhookURL != "https://new.com/webhook" {
			t.Errorf("webhook_url = %q, want %q", feishu.WebhookURL, "https://new.com/webhook")
		}
		if feishu.CardLinkURL != "https://custom.com/{{.QID}}" {
			t.Errorf("card_link_url = %q, want %q", feishu.CardLinkURL, "https://custom.com/{{.QID}}")
		}
	})

	t.Run("updates enabled status", func(t *testing.T) {
		enabled := false
		ch, err := svc.Update(ctx, uid, ch.ID, UpdateChannelParams{Enabled: &enabled})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ch.Enabled {
			t.Error("expected channel to be disabled")
		}
	})

	t.Run("updates keywords", func(t *testing.T) {
		ch, err := svc.Update(ctx, uid, ch.ID, UpdateChannelParams{Keywords: ptrTo("new,keywords")})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ch.Keywords != "new,keywords" {
			t.Errorf("keywords = %q, want %q", ch.Keywords, "new,keywords")
		}
	})

	t.Run("clears keywords with empty string", func(t *testing.T) {
		ch, err := svc.Update(ctx, uid, ch.ID, UpdateChannelParams{Keywords: ptrTo("")})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ch.Keywords != "" {
			t.Errorf("keywords = %q, want %q", ch.Keywords, "")
		}
	})

	t.Run("rejects invalid keywords on update", func(t *testing.T) {
		_, err := svc.Update(ctx, uid, ch.ID, UpdateChannelParams{Keywords: ptrTo("a,,b")})
		if err == nil {
			t.Fatal("expected error for invalid keywords")
		}
		se, _ := service.AsError(err)
		if se.Code != service.ErrValidation {
			t.Errorf("expected code %q, got %q", service.ErrValidation.Value, se.Code.Value)
		}
	})

	t.Run("rejects invalid webhook URL on update", func(t *testing.T) {
		_, err := svc.Update(ctx, uid, ch.ID, UpdateChannelParams{
			Configuration: feishuConfigJSON("ftp://invalid", ""),
		})
		if err == nil {
			t.Fatal("expected error for invalid webhook URL")
		}
	})

	t.Run("rejects invalid kind on update", func(t *testing.T) {
		_, err := svc.Update(ctx, uid, ch.ID, UpdateChannelParams{Kind: "slack"})
		if err == nil {
			t.Fatal("expected error for invalid kind")
		}
	})
}

func TestService_Delete(t *testing.T) {
	svc, repo, db := setupDeliveryService(t)
	uid := createTestUser(t, db)
	ctx := context.Background()

	ch := &delivery.Channel{UserID: uid, Kind: delivery.ChannelKindFeishu, Name: "Ch", Configuration: delivery.ChannelConfiguration{"webhook_url": "https://a.com", "card_link_url": ""}}
	_ = repo.Create(ctx, ch)

	t.Run("deletes channel successfully", func(t *testing.T) {
		err := svc.Delete(ctx, uid, ch.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("returns error for non-existent channel", func(t *testing.T) {
		err := svc.Delete(ctx, uid, 999)
		if err == nil {
			t.Fatal("expected error for non-existent channel")
		}
		se, _ := service.AsError(err)
		if se.Code != service.ErrNotFound {
			t.Errorf("expected code %q, got %q", service.ErrNotFound.Value, se.Code.Value)
		}
	})
}

func TestService_ListAll(t *testing.T) {
	svc, repo, db := setupDeliveryService(t)
	uid1 := createTestUser(t, db, 0)
	uid2 := createTestUser(t, db, 1)
	ctx := context.Background()

	_ = repo.Create(ctx, &delivery.Channel{UserID: uid1, Kind: delivery.ChannelKindFeishu, Name: "Ch1", Configuration: delivery.ChannelConfiguration{"webhook_url": "https://a.com", "card_link_url": ""}})
	_ = repo.Create(ctx, &delivery.Channel{UserID: uid2, Kind: delivery.ChannelKindFeishu, Name: "Ch2", Configuration: delivery.ChannelConfiguration{"webhook_url": "https://b.com", "card_link_url": ""}})

	channels, total, err := svc.ListAll(ctx, 0, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(channels) != 2 {
		t.Errorf("got %d channels, want 2", len(channels))
	}
	if total != 2 {
		t.Errorf("total = %d, want 2", total)
	}
}

func TestNormalizeAndValidateKind(t *testing.T) {
	tests := []struct {
		name    string
		kind    string
		want    delivery.ChannelKind
		wantErr bool
	}{
		{"lowercase feishu", "feishu", delivery.ChannelKindFeishu, false},
		{"mixed case Feishu", "Feishu", delivery.ChannelKindFeishu, false},
		{"whitespace padded", "  feishu  ", delivery.ChannelKindFeishu, false},
		{"unsupported kind", "slack", "", true},
		{"empty string", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeAndValidateKind(tt.kind)
			if (err != nil) != tt.wantErr {
				t.Fatalf("normalizeAndValidateKind(%q) error = %v, wantErr %v", tt.kind, err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("normalizeAndValidateKind(%q) = %q, want %q", tt.kind, got, tt.want)
			}
		})
	}
}

func TestValidateConfiguration(t *testing.T) {
	t.Run("valid feishu configuration", func(t *testing.T) {
		config, err := validateConfiguration(delivery.ChannelKindFeishu, feishuConfigJSON("https://example.com/hook", ""))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		feishu := config.Feishu()
		if feishu.WebhookURL != "https://example.com/hook" {
			t.Errorf("webhook_url = %q, want %q", feishu.WebhookURL, "https://example.com/hook")
		}
	})

	t.Run("valid feishu configuration with card_link_url", func(t *testing.T) {
		config, err := validateConfiguration(delivery.ChannelKindFeishu, feishuConfigJSON("https://example.com/hook", "https://custom.com/{{.QID}}"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		feishu := config.Feishu()
		if feishu.CardLinkURL != "https://custom.com/{{.QID}}" {
			t.Errorf("card_link_url = %q, want %q", feishu.CardLinkURL, "https://custom.com/{{.QID}}")
		}
	})

	t.Run("defaults card_link_url to empty", func(t *testing.T) {
		raw := json.RawMessage(`{"webhook_url":"https://example.com/hook"}`)
		config, err := validateConfiguration(delivery.ChannelKindFeishu, raw)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		feishu := config.Feishu()
		if feishu.CardLinkURL != "" {
			t.Errorf("card_link_url = %q, want empty", feishu.CardLinkURL)
		}
	})

	t.Run("rejects invalid JSON", func(t *testing.T) {
		_, err := validateConfiguration(delivery.ChannelKindFeishu, json.RawMessage(`not json`))
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("rejects empty webhook URL", func(t *testing.T) {
		_, err := validateConfiguration(delivery.ChannelKindFeishu, feishuConfigJSON("", ""))
		if err == nil {
			t.Fatal("expected error for empty webhook URL")
		}
	})

	t.Run("rejects invalid webhook URL scheme", func(t *testing.T) {
		_, err := validateConfiguration(delivery.ChannelKindFeishu, feishuConfigJSON("ftp://example.com", ""))
		if err == nil {
			t.Fatal("expected error for invalid URL scheme")
		}
	})

	t.Run("trims whitespace from webhook URL", func(t *testing.T) {
		config, err := validateConfiguration(delivery.ChannelKindFeishu, feishuConfigJSON("  https://example.com/hook  ", ""))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		feishu := config.Feishu()
		if feishu.WebhookURL != "https://example.com/hook" {
			t.Errorf("webhook_url = %q, want %q", feishu.WebhookURL, "https://example.com/hook")
		}
	})
}

func TestService_LatestPerChannel(t *testing.T) {
	ctx := context.Background()
	svc, repo, _, db := setupDeliveryServiceWithHistory(t)
	uid := createTestUser(t, db)

	ch1 := &delivery.Channel{
		UserID:        uid,
		Kind:          delivery.ChannelKindFeishu,
		Name:          "ch1",
		Enabled:       true,
		Configuration: delivery.ChannelConfiguration{"webhook_url": "https://example.com/h1", "card_link_url": ""},
	}
	if err := repo.Create(ctx, ch1); err != nil {
		t.Fatalf("create ch1: %v", err)
	}
	ch2 := &delivery.Channel{
		UserID:        uid,
		Kind:          delivery.ChannelKindFeishu,
		Name:          "ch2",
		Enabled:       true,
		Configuration: delivery.ChannelConfiguration{"webhook_url": "https://example.com/h2", "card_link_url": ""},
	}
	if err := repo.Create(ctx, ch2); err != nil {
		t.Fatalf("create ch2: %v", err)
	}

	older := time.Now().Add(-2 * time.Hour)
	recent := time.Now().Add(-1 * time.Minute)
	ch1ID, ch2ID, uID := ch1.ID, ch2.ID, uid
	insertHistory := func(channelID, userID int, status delivery.Status, when time.Time) {
		if err := db.Exec(
			"INSERT INTO delivery_history (status, last_error, user_id, channel_id, created_at) VALUES (?, '', ?, ?, ?)",
			status, userID, channelID, when,
		).Error; err != nil {
			t.Fatalf("seed history: %v", err)
		}
	}
	insertHistory(ch1ID, uID, delivery.StatusDelivered, older)
	insertHistory(ch1ID, uID, delivery.StatusFailed, recent)
	insertHistory(ch2ID, uID, delivery.StatusDelivered, recent)

	rows, err := svc.LatestPerChannel(ctx, uID)
	if err != nil {
		t.Fatalf("LatestPerChannel: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}

	byChannel := make(map[int]delivery.Status, len(rows))
	for _, r := range rows {
		if r.ChannelID == nil {
			t.Fatal("channel_id must be set")
		}
		byChannel[*r.ChannelID] = r.Status
	}
	if byChannel[ch1ID] != delivery.StatusFailed {
		t.Errorf("ch1 latest status = %d, want %d (failed, the newer row)", byChannel[ch1ID], delivery.StatusFailed)
	}
	if byChannel[ch2ID] != delivery.StatusDelivered {
		t.Errorf("ch2 latest status = %d, want %d", byChannel[ch2ID], delivery.StatusDelivered)
	}
}

func TestService_SendTest(t *testing.T) {
	ctx := context.Background()

	t.Run("sends test card to configured webhook", func(t *testing.T) {
		var receivedTitle string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			var payload map[string]any
			_ = json.Unmarshal(body, &payload)
			card, _ := payload["card"].(map[string]any)
			header, _ := card["header"].(map[string]any)
			titleObj, _ := header["title"].(map[string]any)
			receivedTitle, _ = titleObj["content"].(string)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"code":0}`))
		}))
		defer server.Close()

		svc, repo, _, db := setupDeliveryServiceWithHistory(t)
		uid := createTestUser(t, db)
		ch := &delivery.Channel{
			UserID:        uid,
			Kind:          delivery.ChannelKindFeishu,
			Name:          "test",
			Enabled:       true,
			Configuration: delivery.ChannelConfiguration{"webhook_url": server.URL, "card_link_url": ""},
		}
		if err := repo.Create(ctx, ch); err != nil {
			t.Fatalf("create channel: %v", err)
		}

		if err := svc.SendTest(ctx, uid, ch.ID); err != nil {
			t.Fatalf("SendTest: %v", err)
		}
		if receivedTitle != testCardTitle {
			t.Errorf("received card title = %q, want %q", receivedTitle, testCardTitle)
		}
	})

	t.Run("returns not found for missing channel", func(t *testing.T) {
		svc, _, _, db := setupDeliveryServiceWithHistory(t)
		uid := createTestUser(t, db)
		err := svc.SendTest(ctx, uid, 9999)
		if err == nil {
			t.Fatal("expected error for missing channel")
		}
		se, ok := service.AsError(err)
		if !ok {
			t.Fatalf("expected service error, got %T: %v", err, err)
		}
		if se.Code != service.ErrNotFound {
			t.Errorf("code = %v, want %v", se.Code, service.ErrNotFound)
		}
	})

	t.Run("wraps webhook failure as internal error", func(t *testing.T) {
		svc, repo, _, db := setupDeliveryServiceWithHistory(t)
		uid := createTestUser(t, db)
		ch := &delivery.Channel{
			UserID:        uid,
			Kind:          delivery.ChannelKindFeishu,
			Name:          "bad",
			Enabled:       true,
			Configuration: delivery.ChannelConfiguration{"webhook_url": "http://192.0.2.1:1/unreachable", "card_link_url": ""},
		}
		if err := repo.Create(ctx, ch); err != nil {
			t.Fatalf("create channel: %v", err)
		}

		err := svc.SendTest(ctx, uid, ch.ID)
		if err == nil {
			t.Fatal("expected error for unreachable webhook")
		}
		se, ok := service.AsError(err)
		if !ok {
			t.Fatalf("expected service error, got %T: %v", err, err)
		}
		if se.Code != service.ErrInternal {
			t.Errorf("code = %v, want %v", se.Code, service.ErrInternal)
		}
	})
}

func ptrTo(s string) *string { return &s }
