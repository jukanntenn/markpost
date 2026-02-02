package repositories

import (
	"testing"

	"markpost/models"
)

func setupDeliveryChannelTestDatabase(t *testing.T) *models.Database {
	t.Helper()

	database, err := models.NewTestDatabase()
	if err != nil {
		t.Fatalf("NewTestDatabase error: %v", err)
	}

	return database
}

func TestDeliveryChannelRepository_CRUDAndScoping(t *testing.T) {
	database := setupDeliveryChannelTestDatabase(t)

	userRepo := NewUserRepo(database)
	channelRepo := NewDeliveryChannelRepo(database)

	user1, err := userRepo.CreateUser("user1", "password")
	if err != nil {
		t.Fatalf("seed user1 error: %v", err)
	}
	user2, err := userRepo.CreateUser("user2", "password")
	if err != nil {
		t.Fatalf("seed user2 error: %v", err)
	}

	created, err := channelRepo.Create(
		user1.ID,
		models.DeliveryChannelKindFeishu,
		"feishu-1",
		"https://open.feishu.cn/open-apis/bot/v2/hook/abcdef",
		"keyword1, keyword2",
		true,
	)
	if err != nil {
		t.Fatalf("create channel error: %v", err)
	}
	if created.ID == 0 || created.UserID != user1.ID || created.Kind != models.DeliveryChannelKindFeishu {
		t.Fatalf("unexpected created channel: %+v", created)
	}
	if created.Keywords != "keyword1, keyword2" {
		t.Fatalf("unexpected created keywords: %q", created.Keywords)
	}

	list, err := channelRepo.ListByUserID(user1.ID)
	if err != nil {
		t.Fatalf("list channels error: %v", err)
	}
	if len(list) != 1 || list[0].ID != created.ID {
		t.Fatalf("unexpected list result: %+v", list)
	}

	got, err := channelRepo.GetByIDAndUserID(created.ID, user1.ID)
	if err != nil {
		t.Fatalf("get by id error: %v", err)
	}
	if got.WebhookURL != created.WebhookURL {
		t.Fatalf("unexpected webhook url: %s", got.WebhookURL)
	}

	_, err = channelRepo.GetByIDAndUserID(created.ID, user2.ID)
	if err != models.ErrNotFound {
		t.Fatalf("expected ErrNotFound for wrong user scope, got: %v", err)
	}

	got.Name = "feishu-updated"
	got.Enabled = false
	got.Keywords = "a,b"
	if err := channelRepo.Update(got); err != nil {
		t.Fatalf("update error: %v", err)
	}

	updated, err := channelRepo.GetByIDAndUserID(created.ID, user1.ID)
	if err != nil {
		t.Fatalf("get updated error: %v", err)
	}
	if updated.Name != "feishu-updated" || updated.Enabled != false {
		t.Fatalf("unexpected updated channel: %+v", updated)
	}
	if updated.Keywords != "a,b" {
		t.Fatalf("unexpected updated keywords: %q", updated.Keywords)
	}

	rows, err := channelRepo.DeleteByIDAndUserID(created.ID, user2.ID)
	if err != nil {
		t.Fatalf("delete wrong scope error: %v", err)
	}
	if rows != 0 {
		t.Fatalf("expected 0 rows deleted, got %d", rows)
	}

	rows, err = channelRepo.DeleteByIDAndUserID(created.ID, user1.ID)
	if err != nil {
		t.Fatalf("delete error: %v", err)
	}
	if rows != 1 {
		t.Fatalf("expected 1 row deleted, got %d", rows)
	}
}
