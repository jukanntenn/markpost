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
	uid := createTestUser(t, db)

	ch := &delivery.Channel{
		UserID:  uid,
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
	uid1 := createTestUser(t, db, 0)
	uid2 := createTestUser(t, db, 1)

	_ = createTestDeliveryChannel(ctx, repo, uid1, "Ch1")
	_ = createTestDeliveryChannel(ctx, repo, uid1, "Ch2")
	_ = createTestDeliveryChannel(ctx, repo, uid2, "Ch3")

	channels, err := repo.GetByUserID(ctx, uid1)
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
	uid1 := createTestUser(t, db, 0)
	uid2 := createTestUser(t, db, 1)

	ch1 := createTestDeliveryChannel(ctx, repo, uid1, "Ch1")
	_ = createTestDeliveryChannel(ctx, repo, uid2, "Ch2")

	t.Run("finds own channel", func(t *testing.T) {
		ch, err := repo.GetByIDAndUserID(ctx, ch1.ID, uid1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ch.Name != "Ch1" {
			t.Errorf("name = %q, want %q", ch.Name, "Ch1")
		}
	})

	t.Run("returns not found for other user's channel", func(t *testing.T) {
		_, err := repo.GetByIDAndUserID(ctx, ch1.ID, uid2)
		if !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("expected ErrNotFound, got: %v", err)
		}
	})
}

func TestDeliveryChannelRepository_Update(t *testing.T) {
	db := SetupTestDB(t)
	repo := NewDeliveryChannelRepository(db)
	ctx := context.Background()
	uid := createTestUser(t, db)

	existing := createTestDeliveryChannel(ctx, repo, uid, "Old")

	ch := &delivery.Channel{
		ID:      existing.ID,
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

	fetched, _ := repo.GetByIDAndUserID(ctx, existing.ID, uid)
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
	uid := createTestUser(t, db)

	ch := createTestDeliveryChannel(ctx, repo, uid, "Ch")

	t.Run("deletes own channel", func(t *testing.T) {
		affected, err := repo.DeleteByIDAndUserID(ctx, ch.ID, uid)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if affected != 1 {
			t.Errorf("affected = %d, want 1", affected)
		}
	})

	t.Run("returns 0 for non-existent", func(t *testing.T) {
		affected, err := repo.DeleteByIDAndUserID(ctx, 999, uid)
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
	uid1 := createTestUser(t, db, 0)
	uid2 := createTestUser(t, db, 1)

	_ = createTestDeliveryChannel(ctx, repo, uid1, "Ch1")
	_ = createTestDeliveryChannel(ctx, repo, uid2, "Ch2")

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
	uid1 := createTestUser(t, db, 0)
	uid2 := createTestUser(t, db, 1)

	_ = createTestDeliveryChannel(ctx, repo, uid1, "Ch1")
	_ = createTestDeliveryChannel(ctx, repo, uid2, "Ch2")

	count, err := repo.CountAll(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}
}
