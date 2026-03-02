package services

import (
	"testing"

	"markpost/models"
	"markpost/repositories"
)

func setupAdminTestDB(t *testing.T) *models.Database {
	t.Helper()

	db, err := models.NewTestDatabase()
	if err != nil {
		t.Fatalf("NewTestDatabase error: %v", err)
	}
	return db
}

func TestAdminService_UpdateUserRole(t *testing.T) {
	db := setupAdminTestDB(t)

	user := &models.User{Username: "u1", Password: "x", PostKey: "pk1", Role: models.RoleUser}
	if err := user.Create(db); err != nil {
		t.Fatalf("Create user error: %v", err)
	}

	svc := NewAdminService(repositories.NewUserRepo(db), repositories.NewPostRepo(db), repositories.NewDeliveryChannelRepo(db))

	updated, err := svc.UpdateUserRole(user.ID, models.RoleAdmin)
	if err != nil {
		t.Fatalf("UpdateUserRole error: %v", err)
	}
	if updated.Role != models.RoleAdmin {
		t.Fatalf("expected role admin, got %s", updated.Role)
	}
}

func TestAdminService_DeleteUser_Cascade(t *testing.T) {
	db := setupAdminTestDB(t)

	user := &models.User{Username: "u2", Password: "x", PostKey: "pk2", Role: models.RoleUser}
	if err := user.Create(db); err != nil {
		t.Fatalf("Create user error: %v", err)
	}

	post := &models.Post{QID: "qid1", Title: "t1", Body: "b1", UserID: user.ID}
	if err := post.Create(db); err != nil {
		t.Fatalf("Create post error: %v", err)
	}

	ch := &models.DeliveryChannel{
		UserID:     user.ID,
		Kind:       models.DeliveryChannelKindFeishu,
		Name:       "c1",
		Enabled:    true,
		WebhookURL: "https://open.feishu.cn/open-apis/bot/v2/hook/xxx",
		Keywords:   "",
	}
	if err := ch.Create(db); err != nil {
		t.Fatalf("Create channel error: %v", err)
	}

	svc := NewAdminService(repositories.NewUserRepo(db), repositories.NewPostRepo(db), repositories.NewDeliveryChannelRepo(db))

	if err := svc.DeleteUser(user.ID); err != nil {
		t.Fatalf("DeleteUser error: %v", err)
	}

	var postsCount int64
	if err := db.DB().Model(&models.Post{}).Where("user_id = ?", user.ID).Count(&postsCount).Error; err != nil {
		t.Fatalf("Count posts error: %v", err)
	}
	if postsCount != 0 {
		t.Fatalf("expected posts cascade delete, got count=%d", postsCount)
	}

	var channelsCount int64
	if err := db.DB().Model(&models.DeliveryChannel{}).Where("user_id = ?", user.ID).Count(&channelsCount).Error; err != nil {
		t.Fatalf("Count channels error: %v", err)
	}
	if channelsCount != 0 {
		t.Fatalf("expected channels cascade delete, got count=%d", channelsCount)
	}
}

func TestAdminService_ListAllPosts_Search(t *testing.T) {
	db := setupAdminTestDB(t)

	user := &models.User{Username: "u3", Password: "x", PostKey: "pk3", Role: models.RoleUser}
	if err := user.Create(db); err != nil {
		t.Fatalf("Create user error: %v", err)
	}

	if err := (&models.Post{QID: "qid2", Title: "hello", Body: "b2", UserID: user.ID}).Create(db); err != nil {
		t.Fatalf("Create post error: %v", err)
	}
	if err := (&models.Post{QID: "qid3", Title: "world", Body: "b3", UserID: user.ID}).Create(db); err != nil {
		t.Fatalf("Create post error: %v", err)
	}

	svc := NewAdminService(repositories.NewUserRepo(db), repositories.NewPostRepo(db), repositories.NewDeliveryChannelRepo(db))

	posts, total, err := svc.ListAllPosts("hello", 0, 10)
	if err != nil {
		t.Fatalf("ListAllPosts error: %v", err)
	}
	if total != 1 {
		t.Fatalf("expected total=1, got %d", total)
	}
	if len(posts) != 1 || posts[0].Title != "hello" {
		t.Fatalf("unexpected posts: %+v", posts)
	}
}

func TestAdminService_UpdateDeliveryChannel(t *testing.T) {
	db := setupAdminTestDB(t)

	user := &models.User{Username: "u4", Password: "x", PostKey: "pk4", Role: models.RoleUser}
	if err := user.Create(db); err != nil {
		t.Fatalf("Create user error: %v", err)
	}

	ch := &models.DeliveryChannel{
		UserID:     user.ID,
		Kind:       models.DeliveryChannelKindFeishu,
		Name:       "old",
		Enabled:    true,
		WebhookURL: "https://open.feishu.cn/open-apis/bot/v2/hook/xxx",
		Keywords:   "",
	}
	if err := ch.Create(db); err != nil {
		t.Fatalf("Create channel error: %v", err)
	}

	svc := NewAdminService(repositories.NewUserRepo(db), repositories.NewPostRepo(db), repositories.NewDeliveryChannelRepo(db))

	newName := "new"
	newKeywords := "k1,k2"
	enabled := false
	updated, err := svc.UpdateDeliveryChannel(ch.ID, &newName, nil, &newKeywords, &enabled)
	if err != nil {
		t.Fatalf("UpdateDeliveryChannel error: %v", err)
	}
	if updated.Name != newName || updated.Keywords != newKeywords || updated.Enabled != enabled {
		t.Fatalf("unexpected channel: %+v", updated)
	}
	if updated.User.Username != "u4" {
		t.Fatalf("expected preloaded user, got: %+v", updated.User)
	}
}
