package delivery

import (
	"context"
	"encoding/json"
	"testing"

	"markpost/internal/domain/delivery"
	"markpost/internal/infra"
	"markpost/internal/service"
)

func setupDeliveryService(t *testing.T) (*Service, delivery.Repository) {
	t.Helper()
	db := infra.SetupTestDB(t)
	repo := infra.NewDeliveryChannelRepository(db)
	svc := NewService(repo)
	return svc, repo
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
	svc, repo := setupDeliveryService(t)
	ctx := context.Background()

	_ = repo.Create(ctx, &delivery.Channel{UserID: 1, Kind: delivery.ChannelKindFeishu, Name: "Ch1", Configuration: delivery.ChannelConfiguration{"webhook_url": "https://a.com", "card_link_url": ""}})
	_ = repo.Create(ctx, &delivery.Channel{UserID: 1, Kind: delivery.ChannelKindFeishu, Name: "Ch2", Configuration: delivery.ChannelConfiguration{"webhook_url": "https://b.com", "card_link_url": ""}})
	_ = repo.Create(ctx, &delivery.Channel{UserID: 2, Kind: delivery.ChannelKindFeishu, Name: "Ch3", Configuration: delivery.ChannelConfiguration{"webhook_url": "https://c.com", "card_link_url": ""}})

	channels, err := svc.ListByUserID(ctx, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(channels) != 2 {
		t.Errorf("got %d channels, want 2", len(channels))
	}
}

func TestService_Create(t *testing.T) {
	svc, _ := setupDeliveryService(t)
	ctx := context.Background()

	t.Run("creates channel successfully", func(t *testing.T) {
		ch, err := svc.Create(ctx, 1, UpdateChannelParams{
			Kind:          "feishu",
			Name:          "My Channel",
			Configuration: feishuConfigJSON("https://example.com/webhook", ""),
			Keywords:      "alert,error",
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
		ch, err := svc.Create(ctx, 1, UpdateChannelParams{
			Kind:          "feishu",
			Name:          "Card Link Channel",
			Configuration: feishuConfigJSON("https://example.com/webhook", "https://custom.example.com/{{.QID}}"),
			Keywords:      "",
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
		_, err := svc.Create(ctx, 1, UpdateChannelParams{Kind: "feishu", Configuration: feishuConfigJSON("https://example.com", "")})
		if err == nil {
			t.Fatal("expected error for empty name")
		}
		se, _ := service.AsServiceError(err)
		if se.Code != service.ErrValidation {
			t.Errorf("expected code %q, got %q", service.ErrValidation, se.Code)
		}
	})

	t.Run("rejects empty configuration", func(t *testing.T) {
		_, err := svc.Create(ctx, 1, UpdateChannelParams{Kind: "feishu", Name: "Test"})
		if err == nil {
			t.Fatal("expected error for empty configuration")
		}
	})

	t.Run("rejects missing webhook URL", func(t *testing.T) {
		_, err := svc.Create(ctx, 1, UpdateChannelParams{
			Kind:          "feishu",
			Name:          "Test",
			Configuration: feishuConfigJSON("", ""),
		})
		if err == nil {
			t.Fatal("expected error for empty webhook URL")
		}
	})

	t.Run("rejects invalid webhook URL", func(t *testing.T) {
		_, err := svc.Create(ctx, 1, UpdateChannelParams{
			Kind:          "feishu",
			Name:          "Test",
			Configuration: feishuConfigJSON("ftp://invalid", ""),
		})
		if err == nil {
			t.Fatal("expected error for invalid webhook URL")
		}
	})

	t.Run("rejects invalid configuration JSON", func(t *testing.T) {
		_, err := svc.Create(ctx, 1, UpdateChannelParams{
			Kind:          "feishu",
			Name:          "Test",
			Configuration: json.RawMessage(`{invalid`),
		})
		if err == nil {
			t.Fatal("expected error for invalid JSON")
		}
	})

	t.Run("rejects unsupported kind", func(t *testing.T) {
		_, err := svc.Create(ctx, 1, UpdateChannelParams{Kind: "slack", Name: "Test", Configuration: feishuConfigJSON("https://example.com", "")})
		if err == nil {
			t.Fatal("expected error for unsupported kind")
		}
		se, _ := service.AsServiceError(err)
		if se.Code != service.ErrValidation {
			t.Errorf("expected code %q, got %q", service.ErrValidation, se.Code)
		}
	})
}

func TestService_Update(t *testing.T) {
	svc, repo := setupDeliveryService(t)
	ctx := context.Background()

	_ = repo.Create(ctx, &delivery.Channel{UserID: 1, Kind: delivery.ChannelKindFeishu, Name: "Old", Configuration: delivery.ChannelConfiguration{"webhook_url": "https://old.com", "card_link_url": ""}, Keywords: "old"})

	t.Run("updates channel name successfully", func(t *testing.T) {
		newName := "New Name"
		ch, err := svc.Update(ctx, 1, 1, UpdateChannelParams{Name: newName})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ch.Name != "New Name" {
			t.Errorf("name = %q, want %q", ch.Name, "New Name")
		}
	})

	t.Run("returns error for non-existent channel", func(t *testing.T) {
		_, err := svc.Update(ctx, 1, 999, UpdateChannelParams{Name: "New"})
		if err == nil {
			t.Fatal("expected error for non-existent channel")
		}
		se, _ := service.AsServiceError(err)
		if se.Code != service.ErrNotFound {
			t.Errorf("expected code %q, got %q", service.ErrNotFound, se.Code)
		}
	})

	t.Run("updates configuration", func(t *testing.T) {
		ch, err := svc.Update(ctx, 1, 1, UpdateChannelParams{
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
		ch, err := svc.Update(ctx, 1, 1, UpdateChannelParams{Enabled: &enabled})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ch.Enabled {
			t.Error("expected channel to be disabled")
		}
	})

	t.Run("updates keywords", func(t *testing.T) {
		ch, err := svc.Update(ctx, 1, 1, UpdateChannelParams{Keywords: "new,keywords"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ch.Keywords != "new,keywords" {
			t.Errorf("keywords = %q, want %q", ch.Keywords, "new,keywords")
		}
	})

	t.Run("rejects invalid webhook URL on update", func(t *testing.T) {
		_, err := svc.Update(ctx, 1, 1, UpdateChannelParams{
			Configuration: feishuConfigJSON("ftp://invalid", ""),
		})
		if err == nil {
			t.Fatal("expected error for invalid webhook URL")
		}
	})

	t.Run("rejects invalid kind on update", func(t *testing.T) {
		_, err := svc.Update(ctx, 1, 1, UpdateChannelParams{Kind: "slack"})
		if err == nil {
			t.Fatal("expected error for invalid kind")
		}
	})
}

func TestService_Delete(t *testing.T) {
	svc, repo := setupDeliveryService(t)
	ctx := context.Background()

	_ = repo.Create(ctx, &delivery.Channel{UserID: 1, Kind: delivery.ChannelKindFeishu, Name: "Ch", Configuration: delivery.ChannelConfiguration{"webhook_url": "https://a.com", "card_link_url": ""}})

	t.Run("deletes channel successfully", func(t *testing.T) {
		err := svc.Delete(ctx, 1, 1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("returns error for non-existent channel", func(t *testing.T) {
		err := svc.Delete(ctx, 1, 999)
		if err == nil {
			t.Fatal("expected error for non-existent channel")
		}
		se, _ := service.AsServiceError(err)
		if se.Code != service.ErrNotFound {
			t.Errorf("expected code %q, got %q", service.ErrNotFound, se.Code)
		}
	})
}

func TestService_ListAll(t *testing.T) {
	svc, repo := setupDeliveryService(t)
	ctx := context.Background()

	_ = repo.Create(ctx, &delivery.Channel{UserID: 1, Kind: delivery.ChannelKindFeishu, Name: "Ch1", Configuration: delivery.ChannelConfiguration{"webhook_url": "https://a.com", "card_link_url": ""}})
	_ = repo.Create(ctx, &delivery.Channel{UserID: 2, Kind: delivery.ChannelKindFeishu, Name: "Ch2", Configuration: delivery.ChannelConfiguration{"webhook_url": "https://b.com", "card_link_url": ""}})

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
