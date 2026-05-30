package infra

import (
	"context"
	"errors"
	"testing"

	"markpost/internal/domain"
	"markpost/internal/domain/delivery"
)

func createTestDeliveryChannel(ctx context.Context, repo delivery.Repository, userID int, name string) *delivery.Channel {
	ch := &delivery.Channel{
		UserID:  userID,
		Kind:    delivery.ChannelKindFeishu,
		Name:    name,
		Enabled: true,
		Configuration: delivery.ChannelConfiguration{
			"webhook_url":   "https://example.com/webhook",
			"card_link_url": "",
		},
		Keywords: "test",
	}
	_ = repo.Create(ctx, ch)
	return ch
}

func TestDeliveryChannelRepository_Create(t *testing.T) {
	db := SetupTestDB(t)
	repo := NewDeliveryChannelRepository(db)
	ctx := context.Background()

	ch := &delivery.Channel{
		UserID:  1,
		Kind:    delivery.ChannelKindFeishu,
		Name:    "Test Channel",
		Enabled: true,
		Configuration: delivery.ChannelConfiguration{
			"webhook_url":   "https://example.com/webhook",
			"card_link_url": "",
		},
		Keywords: "alert,error",
	}

	err := repo.Create(ctx, ch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ch.ID == 0 {
		t.Error("expected ID to be set after create")
	}
}

func TestDeliveryChannelRepository_GetByUserID(t *testing.T) {
	db := SetupTestDB(t)
	repo := NewDeliveryChannelRepository(db)
	ctx := context.Background()

	_ = createTestDeliveryChannel(ctx, repo, 1, "Ch1")
	_ = createTestDeliveryChannel(ctx, repo, 1, "Ch2")
	_ = createTestDeliveryChannel(ctx, repo, 2, "Ch3")

	channels, err := repo.GetByUserID(ctx, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(channels) != 2 {
		t.Errorf("got %d channels, want 2", len(channels))
	}
}

func TestDeliveryChannelRepository_GetByIDAndUserID(t *testing.T) {
	db := SetupTestDB(t)
	repo := NewDeliveryChannelRepository(db)
	ctx := context.Background()

	_ = createTestDeliveryChannel(ctx, repo, 1, "Ch1")
	_ = createTestDeliveryChannel(ctx, repo, 2, "Ch2")

	t.Run("finds own channel", func(t *testing.T) {
		ch, err := repo.GetByIDAndUserID(ctx, 1, 1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ch.Name != "Ch1" {
			t.Errorf("name = %q, want %q", ch.Name, "Ch1")
		}
	})

	t.Run("returns not found for other user's channel", func(t *testing.T) {
		_, err := repo.GetByIDAndUserID(ctx, 1, 2)
		if !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("expected ErrNotFound, got: %v", err)
		}
	})
}

func TestDeliveryChannelRepository_Update(t *testing.T) {
	db := SetupTestDB(t)
	repo := NewDeliveryChannelRepository(db)
	ctx := context.Background()

	_ = createTestDeliveryChannel(ctx, repo, 1, "Old")

	ch := &delivery.Channel{
		ID:      1,
		Kind:    delivery.ChannelKindFeishu,
		Name:    "New",
		Enabled: false,
		Configuration: delivery.ChannelConfiguration{
			"webhook_url":   "https://new.com",
			"card_link_url": "https://custom.com/{{.QID}}",
		},
		Keywords: "updated",
	}

	err := repo.Update(ctx, ch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	fetched, _ := repo.GetByIDAndUserID(ctx, 1, 1)
	if fetched.Name != "New" {
		t.Errorf("name = %q, want %q", fetched.Name, "New")
	}
	feishuConfig := fetched.Configuration.Feishu()
	if feishuConfig.WebhookURL != "https://new.com" {
		t.Errorf("webhook_url = %q, want %q", feishuConfig.WebhookURL, "https://new.com")
	}
	if feishuConfig.CardLinkURL != "https://custom.com/{{.QID}}" {
		t.Errorf("card_link_url = %q, want %q", feishuConfig.CardLinkURL, "https://custom.com/{{.QID}}")
	}
}

func TestDeliveryChannelRepository_DeleteByIDAndUserID(t *testing.T) {
	db := SetupTestDB(t)
	repo := NewDeliveryChannelRepository(db)
	ctx := context.Background()

	_ = createTestDeliveryChannel(ctx, repo, 1, "Ch")

	t.Run("deletes own channel", func(t *testing.T) {
		affected, err := repo.DeleteByIDAndUserID(ctx, 1, 1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if affected != 1 {
			t.Errorf("affected = %d, want 1", affected)
		}
	})

	t.Run("returns 0 for non-existent", func(t *testing.T) {
		affected, err := repo.DeleteByIDAndUserID(ctx, 999, 1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if affected != 0 {
			t.Errorf("affected = %d, want 0", affected)
		}
	})
}

func TestDeliveryChannelRepository_ListAll(t *testing.T) {
	db := SetupTestDB(t)
	repo := NewDeliveryChannelRepository(db)
	ctx := context.Background()

	_ = createTestDeliveryChannel(ctx, repo, 1, "Ch1")
	_ = createTestDeliveryChannel(ctx, repo, 2, "Ch2")

	channels, err := repo.ListAll(ctx, 0, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(channels) != 2 {
		t.Errorf("got %d channels, want 2", len(channels))
	}
}

func TestDeliveryChannelRepository_CountAll(t *testing.T) {
	db := SetupTestDB(t)
	repo := NewDeliveryChannelRepository(db)
	ctx := context.Background()

	_ = createTestDeliveryChannel(ctx, repo, 1, "Ch1")
	_ = createTestDeliveryChannel(ctx, repo, 2, "Ch2")

	count, err := repo.CountAll(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}
}
