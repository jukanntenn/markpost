package delivery

import (
	"context"
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

func TestService_ListByUserID(t *testing.T) {
	svc, repo := setupDeliveryService(t)
	ctx := context.Background()

	_ = repo.Create(ctx, &delivery.Channel{UserID: 1, Kind: delivery.ChannelKindFeishu, Name: "Ch1", WebhookURL: "https://a.com"})
	_ = repo.Create(ctx, &delivery.Channel{UserID: 1, Kind: delivery.ChannelKindFeishu, Name: "Ch2", WebhookURL: "https://b.com"})
	_ = repo.Create(ctx, &delivery.Channel{UserID: 2, Kind: delivery.ChannelKindFeishu, Name: "Ch3", WebhookURL: "https://c.com"})

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
			Kind:       "feishu",
			Name:       "My Channel",
			WebhookURL: "https://example.com/webhook",
			Keywords:   "alert,error",
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
	})

	t.Run("rejects empty name", func(t *testing.T) {
		_, err := svc.Create(ctx, 1, UpdateChannelParams{Kind: "feishu", WebhookURL: "https://example.com"})
		if err == nil {
			t.Fatal("expected error for empty name")
		}
		se, _ := service.AsServiceError(err)
		if se.Code != service.ErrValidation {
			t.Errorf("expected code %q, got %q", service.ErrValidation, se.Code)
		}
	})

	t.Run("rejects empty webhook URL", func(t *testing.T) {
		_, err := svc.Create(ctx, 1, UpdateChannelParams{Kind: "feishu", Name: "Test"})
		if err == nil {
			t.Fatal("expected error for empty webhook URL")
		}
	})

	t.Run("rejects invalid webhook URL", func(t *testing.T) {
		_, err := svc.Create(ctx, 1, UpdateChannelParams{Kind: "feishu", Name: "Test", WebhookURL: "ftp://invalid"})
		if err == nil {
			t.Fatal("expected error for invalid webhook URL")
		}
	})

	t.Run("rejects unsupported kind", func(t *testing.T) {
		_, err := svc.Create(ctx, 1, UpdateChannelParams{Kind: "slack", Name: "Test", WebhookURL: "https://example.com"})
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

	_ = repo.Create(ctx, &delivery.Channel{UserID: 1, Kind: delivery.ChannelKindFeishu, Name: "Old", WebhookURL: "https://old.com", Keywords: "old"})

	t.Run("updates channel successfully", func(t *testing.T) {
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

	t.Run("updates webhook URL", func(t *testing.T) {
		newURL := "https://new.com/webhook"
		ch, err := svc.Update(ctx, 1, 1, UpdateChannelParams{WebhookURL: newURL})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ch.WebhookURL != newURL {
			t.Errorf("webhook_url = %q, want %q", ch.WebhookURL, newURL)
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
		_, err := svc.Update(ctx, 1, 1, UpdateChannelParams{WebhookURL: "ftp://invalid"})
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

	_ = repo.Create(ctx, &delivery.Channel{UserID: 1, Kind: delivery.ChannelKindFeishu, Name: "Ch", WebhookURL: "https://a.com"})

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

	_ = repo.Create(ctx, &delivery.Channel{UserID: 1, Kind: delivery.ChannelKindFeishu, Name: "Ch1", WebhookURL: "https://a.com"})
	_ = repo.Create(ctx, &delivery.Channel{UserID: 2, Kind: delivery.ChannelKindFeishu, Name: "Ch2", WebhookURL: "https://b.com"})

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

func TestValidateWebhookURL(t *testing.T) {
	t.Run("valid URL", func(t *testing.T) {
		cleaned, err := validateWebhookURL("  https://example.com/webhook  ")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cleaned != "https://example.com/webhook" {
			t.Errorf("cleaned = %q, want %q", cleaned, "https://example.com/webhook")
		}
	})

	t.Run("invalid URL scheme", func(t *testing.T) {
		_, err := validateWebhookURL("ftp://example.com/hook")
		if err == nil {
			t.Fatal("expected error")
		}
		se, _ := service.AsServiceError(err)
		if se.Code != service.ErrValidation {
			t.Errorf("expected code %q, got %q", service.ErrValidation, se.Code)
		}
	})

	t.Run("unparseable URL", func(t *testing.T) {
		_, err := validateWebhookURL("://not-a-url")
		if err == nil {
			t.Fatal("expected error")
		}
	})
}
