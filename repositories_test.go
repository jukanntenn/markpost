package main

import (
	"database/sql"
	"testing"
	"time"
)

func setupTestDB(t *testing.T) *Database {
	t.Helper()
	db, err := NewTestDatabase("")
	if err != nil {
		t.Fatalf("NewTestDatabase error: %v", err)
	}
	return db
}

func teardownTestDB(t *testing.T, db *Database) {
	t.Helper()
	sqlDB, err := db.GetDB().DB()
	if err == nil {
		_ = sqlDB.Close()
	}
}

func TestUserRepository_GetUserByGitHubID(t *testing.T) {
	t.Run("returns expected record for valid GitHubID", func(t *testing.T) {
		db := setupTestDB(t)
		defer teardownTestDB(t, db)

		repo := db.GetUserRepository()
		githubID := int64(12345)
		postKey, _ := GeneratePostKey(8)
		u := User{Username: "alice", PostKey: postKey, GitHubID: &githubID}
		if err := db.GetDB().Create(&u).Error; err != nil {
			t.Fatalf("seed user error: %v", err)
		}

		got, err := repo.GetUserByGitHubID(githubID)
		if err != nil {
			t.Fatalf("GetUserByGitHubID error: %v", err)
		}
		if got == nil || got.ID == 0 || got.Username != "alice" || got.GitHubID == nil || *got.GitHubID != githubID {
			t.Fatalf("unexpected user: %+v", got)
		}
	})

	t.Run("returns sql.ErrNoRows for wrong GitHubID", func(t *testing.T) {
		db := setupTestDB(t)
		defer teardownTestDB(t, db)

		repo := db.GetUserRepository()
		githubID := int64(99999)
		postKey, _ := GeneratePostKey(8)
		u := User{Username: "bob", PostKey: postKey, GitHubID: &githubID}
		if err := db.GetDB().Create(&u).Error; err != nil {
			t.Fatalf("seed user error: %v", err)
		}

		_, err := repo.GetUserByGitHubID(int64(123))
		if err == nil || err != sql.ErrNoRows {
			t.Fatalf("expected sql.ErrNoRows, got %v", err)
		}
	})
}

func TestUserRepository_GetUserByPostKey(t *testing.T) {
	t.Run("returns expected record for valid postKey", func(t *testing.T) {
		db := setupTestDB(t)
		defer teardownTestDB(t, db)

		repo := db.GetUserRepository()
		githubID := int64(321)
		u := User{Username: "charlie", PostKey: "pk-123", GitHubID: &githubID}
		if err := db.GetDB().Create(&u).Error; err != nil {
			t.Fatalf("seed user error: %v", err)
		}

		got, err := repo.GetUserByPostKey("pk-123")
		if err != nil {
			t.Fatalf("GetUserByPostKey error: %v", err)
		}
		if got == nil || got.Username != "charlie" || got.PostKey != "pk-123" {
			t.Fatalf("unexpected user: %+v", got)
		}
	})

	t.Run("returns sql.ErrNoRows for wrong postKey", func(t *testing.T) {
		db := setupTestDB(t)
		defer teardownTestDB(t, db)

		repo := db.GetUserRepository()
		githubID := int64(888)
		u := User{Username: "dave", PostKey: "pk-888", GitHubID: &githubID}
		if err := db.GetDB().Create(&u).Error; err != nil {
			t.Fatalf("seed user error: %v", err)
		}

		_, err := repo.GetUserByPostKey("not-exist")
		if err == nil || err != sql.ErrNoRows {
			t.Fatalf("expected sql.ErrNoRows, got %v", err)
		}
	})
}

func TestUserRepository_GetUserByID(t *testing.T) {
	t.Run("returns expected record for valid ID", func(t *testing.T) {
		db := setupTestDB(t)
		defer teardownTestDB(t, db)

		repo := db.GetUserRepository()
		postKey, _ := GeneratePostKey(8)
		u := User{Username: "eve", PostKey: postKey}
		if err := db.GetDB().Create(&u).Error; err != nil {
			t.Fatalf("seed user error: %v", err)
		}

		got, err := repo.GetUserByID(u.ID)
		if err != nil {
			t.Fatalf("GetUserByID error: %v", err)
		}
		if got == nil || got.ID != u.ID || got.Username != "eve" {
			t.Fatalf("unexpected user: %+v", got)
		}
	})

	t.Run("returns sql.ErrNoRows for wrong ID", func(t *testing.T) {
		db := setupTestDB(t)
		defer teardownTestDB(t, db)

		repo := db.GetUserRepository()
		postKey, _ := GeneratePostKey(8)
		u := User{Username: "frank", PostKey: postKey}
		if err := db.GetDB().Create(&u).Error; err != nil {
			t.Fatalf("seed user error: %v", err)
		}

		_, err := repo.GetUserByID(123456)
		if err == nil || err != sql.ErrNoRows {
			t.Fatalf("expected sql.ErrNoRows, got %v", err)
		}
	})
}

func TestUserRepository_GetUserByUsername(t *testing.T) {
	t.Run("returns expected record for valid username", func(t *testing.T) {
		db := setupTestDB(t)
		defer teardownTestDB(t, db)

		repo := db.GetUserRepository()
		postKey, _ := GeneratePostKey(8)
		u := User{Username: "grace", PostKey: postKey}
		if err := db.GetDB().Create(&u).Error; err != nil {
			t.Fatalf("seed user error: %v", err)
		}

		got, err := repo.GetUserByUsername("grace")
		if err != nil {
			t.Fatalf("GetUserByUsername error: %v", err)
		}
		if got == nil || got.ID == 0 || got.Username != "grace" {
			t.Fatalf("unexpected user: %+v", got)
		}
	})

	t.Run("returns sql.ErrNoRows for wrong username", func(t *testing.T) {
		db := setupTestDB(t)
		defer teardownTestDB(t, db)

		repo := db.GetUserRepository()
		postKey, _ := GeneratePostKey(8)
		u := User{Username: "harry", PostKey: postKey}
		if err := db.GetDB().Create(&u).Error; err != nil {
			t.Fatalf("seed user error: %v", err)
		}

		_, err := repo.GetUserByUsername("unknown")
		if err == nil || err != sql.ErrNoRows {
			t.Fatalf("expected sql.ErrNoRows, got %v", err)
		}
	})
}

func TestUserRepository_CreateUserFromGitHubUser(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	repo := db.GetUserRepository()
	gh := &GitHubUser{ID: 7777, Login: "ivy"}
	got, err := repo.CreateUserFromGitHubUser(gh)
	if err != nil {
		t.Fatalf("CreateUserFromGitHubUser error: %v", err)
	}
	if got == nil || got.ID == 0 || got.Username != "ivy" || got.GitHubID == nil || *got.GitHubID != 7777 || got.PostKey == "" {
		t.Fatalf("unexpected user: %+v", got)
	}

	got2, err := repo.GetUserByGitHubID(7777)
	if err != nil || got2 == nil || got2.Username != "ivy" {
		t.Fatalf("user not persisted: %v %+v", err, got2)
	}
}

func TestUserRepository_GetOrCreateUserFromGitHubUser(t *testing.T) {
	t.Run("returns existing user for valid GitHubUser", func(t *testing.T) {
		db := setupTestDB(t)
		defer teardownTestDB(t, db)

		repo := db.GetUserRepository()
		githubID := int64(1000)
		postKey, _ := GeneratePostKey(8)
		u := User{Username: "jack", PostKey: postKey, GitHubID: &githubID}
		if err := db.GetDB().Create(&u).Error; err != nil {
			t.Fatalf("seed user error: %v", err)
		}

		gh := &GitHubUser{ID: githubID, Login: "other"}
		got, err := repo.GetOrCreateUserFromGitHubUser(gh)
		if err != nil {
			t.Fatalf("GetOrCreateUserFromGitHubUser error: %v", err)
		}
		if got == nil || got.ID != u.ID || got.Username != "jack" || got.GitHubID == nil || *got.GitHubID != githubID {
			t.Fatalf("unexpected user: %+v", got)
		}
	})

	t.Run("creates new user for non-existing GitHubID", func(t *testing.T) {
		db := setupTestDB(t)
		defer teardownTestDB(t, db)

		repo := db.GetUserRepository()
		gh := &GitHubUser{ID: 2000, Login: "kate"}
		got, err := repo.GetOrCreateUserFromGitHubUser(gh)
		if err != nil {
			t.Fatalf("GetOrCreateUserFromGitHubUser error: %v", err)
		}
		if got == nil || got.Username != "kate" || got.GitHubID == nil || *got.GitHubID != 2000 || got.PostKey == "" {
			t.Fatalf("unexpected user: %+v", got)
		}

		got2, err := repo.GetUserByGitHubID(2000)
		if err != nil || got2 == nil || got2.Username != "kate" {
			t.Fatalf("user not persisted: %v %+v", err, got2)
		}
	})
}

func TestUserRepository_CreateUserWithPassword(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	repo := db.GetUserRepository()
	got, err := repo.CreateUserWithPassword("lucy", "p@ssw0rd")
	if err != nil {
		t.Fatalf("CreateUserWithPassword error: %v", err)
	}
	if got == nil || got.ID == 0 || got.Username != "lucy" || got.Password == "" || got.PostKey == "" {
		t.Fatalf("unexpected user: %+v", got)
	}
	if err := CheckPassword("p@ssw0rd", got.Password); err != nil {
		t.Fatalf("password not matched: %v", err)
	}

	got2, err := repo.GetUserByUsername("lucy")
	if err != nil || got2 == nil || got2.Username != "lucy" {
		t.Fatalf("user not persisted: %v %+v", err, got2)
	}
}

func TestUserRepository_ValidateUserPassword(t *testing.T) {
	t.Run("user not found returns expected error", func(t *testing.T) {
		db := setupTestDB(t)
		defer teardownTestDB(t, db)

		repo := db.GetUserRepository()
		_, err := repo.ValidateUserPassword("no-user", "any")
		if err == nil || err != sql.ErrNoRows {
			t.Fatalf("expected sql.ErrNoRows, got %v", err)
		}
	})

	t.Run("empty password returns expected error", func(t *testing.T) {
		db := setupTestDB(t)
		defer teardownTestDB(t, db)

		repo := db.GetUserRepository()
		postKey, _ := GeneratePostKey(8)
		u := User{Username: "np", Password: "", PostKey: postKey}
		if err := db.GetDB().Create(&u).Error; err != nil {
			t.Fatalf("seed user error: %v", err)
		}
		_, err := repo.ValidateUserPassword("np", "any")
		if err == nil || err.Error() != "user does not have password set" {
			t.Fatalf("expected 'user does not have password set', got %v", err)
		}
	})

	t.Run("password mismatch returns expected error", func(t *testing.T) {
		db := setupTestDB(t)
		defer teardownTestDB(t, db)

		repo := db.GetUserRepository()
		hashed, _ := HashPassword("secret")
		postKey, _ := GeneratePostKey(8)
		u := User{Username: "tom", Password: hashed, PostKey: postKey}
		if err := db.GetDB().Create(&u).Error; err != nil {
			t.Fatalf("seed user error: %v", err)
		}
		_, err := repo.ValidateUserPassword("tom", "wrong")
		if err == nil || err.Error() != "invalid password" {
			t.Fatalf("expected 'invalid password', got %v", err)
		}
	})

	t.Run("password match returns user with nil error", func(t *testing.T) {
		db := setupTestDB(t)
		defer teardownTestDB(t, db)

		repo := db.GetUserRepository()
		hashed, _ := HashPassword("secret")
		postKey, _ := GeneratePostKey(8)
		u := User{Username: "zoe", Password: hashed, PostKey: postKey}
		if err := db.GetDB().Create(&u).Error; err != nil {
			t.Fatalf("seed user error: %v", err)
		}
		got, err := repo.ValidateUserPassword("zoe", "secret")
		if err != nil {
			t.Fatalf("ValidateUserPassword error: %v", err)
		}
		if got == nil || got.Username != "zoe" || got.ID == 0 {
			t.Fatalf("unexpected user: %+v", got)
		}
	})
}

func TestPostRepository_CreatePost(t *testing.T) {
	t.Run("creates post without user", func(t *testing.T) {
		db := setupTestDB(t)
		defer teardownTestDB(t, db)

		repo := db.GetPostRepository()
		p, err := repo.CreatePost("title", "body")
		if err != nil {
			t.Fatalf("CreatePost error: %v", err)
		}
		if p == nil || p.ID == "" || p.Title != "title" || p.Body != "body" || p.UserID != nil {
			t.Fatalf("unexpected post: %+v", p)
		}

		var count int64
		if err := db.GetDB().Model(&Post{}).Where("id = ?", p.ID).Count(&count).Error; err != nil || count != 1 {
			t.Fatalf("post not persisted: %v count=%d", err, count)
		}
	})

	t.Run("creates post with valid user", func(t *testing.T) {
		db := setupTestDB(t)
		defer teardownTestDB(t, db)

		userRepo := db.GetUserRepository()
		u, err := userRepo.CreateUserWithPassword("poster", "pwd")
		if err != nil {
			t.Fatalf("seed user error: %v", err)
		}

		postRepo := db.GetPostRepository()
		p, err := postRepo.CreatePost("title2", "body2", u.ID)
		if err != nil {
			t.Fatalf("CreatePost error: %v", err)
		}
		if p == nil || p.ID == "" || p.UserID == nil || *p.UserID != u.ID {
			t.Fatalf("unexpected post: %+v", p)
		}
	})
}

func TestPostRepository_CreatePostWithUser(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	userRepo := db.GetUserRepository()
	u, err := userRepo.CreateUserWithPassword("author", "pwd")
	if err != nil {
		t.Fatalf("seed user error: %v", err)
	}

	postRepo := db.GetPostRepository()
	p, err := postRepo.CreatePostWithUser("t", "b", u.ID)
	if err != nil {
		t.Fatalf("CreatePostWithUser error: %v", err)
	}
	if p == nil || p.ID == "" || p.UserID == nil || *p.UserID != u.ID {
		t.Fatalf("unexpected post: %+v", p)
	}
}

func TestPostRepository_GetPostByID(t *testing.T) {
	t.Run("returns expected record for valid id", func(t *testing.T) {
		db := setupTestDB(t)
		defer teardownTestDB(t, db)

		postRepo := db.GetPostRepository()
		p, err := postRepo.CreatePost("t1", "b1")
		if err != nil {
			t.Fatalf("CreatePost error: %v", err)
		}
		got, err := postRepo.GetPostByID(p.ID)
		if err != nil {
			t.Fatalf("GetPostByID error: %v", err)
		}
		if got == nil || got.ID != p.ID || got.Title != "t1" || got.Body != "b1" {
			t.Fatalf("unexpected post: %+v", got)
		}
	})

	t.Run("returns sql.ErrNoRows for wrong id", func(t *testing.T) {
		db := setupTestDB(t)
		defer teardownTestDB(t, db)

		postRepo := db.GetPostRepository()
		_, err := postRepo.GetPostByID("non-existent-id")
		if err == nil || err != sql.ErrNoRows {
			t.Fatalf("expected sql.ErrNoRows, got %v", err)
		}
	})
}

func TestPostRepository_GetPostsByUserID(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	userRepo := db.GetUserRepository()
	u1, _ := userRepo.CreateUserWithPassword("u1", "p1")
	u2, _ := userRepo.CreateUserWithPassword("u2", "p2")

	postRepo := db.GetPostRepository()
	if _, err := postRepo.CreatePost("a", "A", u1.ID); err != nil {
		t.Fatalf("CreatePost error: %v", err)
	}
	if _, err := postRepo.CreatePost("b", "B", u1.ID); err != nil {
		t.Fatalf("CreatePost error: %v", err)
	}
	if _, err := postRepo.CreatePost("c", "C", u2.ID); err != nil {
		t.Fatalf("CreatePost error: %v", err)
	}

	posts, err := postRepo.GetPostsByUserID(u1.ID)
	if err != nil {
		t.Fatalf("GetPostsByUserID error: %v", err)
	}
	if len(posts) != 2 {
		t.Fatalf("unexpected posts length: %d", len(posts))
	}
	for _, p := range posts {
		if p.UserID == nil || *p.UserID != u1.ID {
			t.Fatalf("unexpected post userID: %+v", p)
		}
	}
}

func TestPostRepository_GetExpiredPostsCount(t *testing.T) {
	t.Run("returns error when retentionDays <= 0", func(t *testing.T) {
		db := setupTestDB(t)
		defer teardownTestDB(t, db)

		postRepo := db.GetPostRepository()
		_, err := postRepo.GetExpiredPostsCount(0)
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
	})

	t.Run("counts expired posts correctly", func(t *testing.T) {
		db := setupTestDB(t)
		defer teardownTestDB(t, db)

		now := time.Now().UTC()
		expired1 := now.AddDate(0, 0, -10)
		expired2 := now.AddDate(0, 0, -8)
		notExpired := now.AddDate(0, 0, -3)

		if err := db.GetDB().Create(&Post{ID: "p1", Title: "t1", Body: "b1", CreatedAt: expired1}).Error; err != nil {
			t.Fatalf("seed post error: %v", err)
		}
		if err := db.GetDB().Create(&Post{ID: "p2", Title: "t2", Body: "b2", CreatedAt: expired2}).Error; err != nil {
			t.Fatalf("seed post error: %v", err)
		}
		if err := db.GetDB().Create(&Post{ID: "p3", Title: "t3", Body: "b3", CreatedAt: notExpired}).Error; err != nil {
			t.Fatalf("seed post error: %v", err)
		}

		postRepo := db.GetPostRepository()
		cnt, err := postRepo.GetExpiredPostsCount(7)
		if err != nil {
			t.Fatalf("GetExpiredPostsCount error: %v", err)
		}
		if cnt != 2 {
			t.Fatalf("unexpected count: %d", cnt)
		}
	})
}

func TestPostRepository_PreviewExpiredPosts(t *testing.T) {
	t.Run("returns error when retentionDays <= 0", func(t *testing.T) {
		db := setupTestDB(t)
		defer teardownTestDB(t, db)

		postRepo := db.GetPostRepository()
		_, err := postRepo.PreviewExpiredPosts(0, 5)
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
	})

	t.Run("returns expired posts ordered and limited", func(t *testing.T) {
		db := setupTestDB(t)
		defer teardownTestDB(t, db)

		now := time.Now().UTC()
		e1 := now.AddDate(0, 0, -12)
		e2 := now.AddDate(0, 0, -10)
		e3 := now.AddDate(0, 0, -9)
		ne := now.AddDate(0, 0, -2)

		if err := db.GetDB().Create(&Post{ID: "q1", Title: "t1", Body: "b1", CreatedAt: e1}).Error; err != nil {
			t.Fatalf("seed post error: %v", err)
		}
		if err := db.GetDB().Create(&Post{ID: "q2", Title: "t2", Body: "b2", CreatedAt: e2}).Error; err != nil {
			t.Fatalf("seed post error: %v", err)
		}
		if err := db.GetDB().Create(&Post{ID: "q3", Title: "t3", Body: "b3", CreatedAt: e3}).Error; err != nil {
			t.Fatalf("seed post error: %v", err)
		}
		if err := db.GetDB().Create(&Post{ID: "q4", Title: "t4", Body: "b4", CreatedAt: ne}).Error; err != nil {
			t.Fatalf("seed post error: %v", err)
		}

		postRepo := db.GetPostRepository()
		posts, err := postRepo.PreviewExpiredPosts(7, 2)
		if err != nil {
			t.Fatalf("PreviewExpiredPosts error: %v", err)
		}
		if len(posts) != 2 {
			t.Fatalf("unexpected posts length: %d", len(posts))
		}
		if !posts[0].CreatedAt.Before(posts[1].CreatedAt) {
			t.Fatalf("posts not ordered ascending by created_at: %+v", posts)
		}
	})
}

func TestPostRepository_CleanupExpiredPosts(t *testing.T) {
	t.Run("returns error when retentionDays <= 0", func(t *testing.T) {
		db := setupTestDB(t)
		defer teardownTestDB(t, db)

		postRepo := db.GetPostRepository()
		if err := postRepo.CleanupExpiredPosts(0, 10); err == nil {
			t.Fatalf("expected error, got nil")
		}
	})

	t.Run("deletes only expired posts", func(t *testing.T) {
		db := setupTestDB(t)
		defer teardownTestDB(t, db)

		now := time.Now().UTC()
		e1 := now.AddDate(0, 0, -20)
		e2 := now.AddDate(0, 0, -15)
		ne := now.AddDate(0, 0, -3)

		if err := db.GetDB().Create(&Post{ID: "r1", Title: "t1", Body: "b1", CreatedAt: e1}).Error; err != nil {
			t.Fatalf("seed post error: %v", err)
		}
		if err := db.GetDB().Create(&Post{ID: "r2", Title: "t2", Body: "b2", CreatedAt: e2}).Error; err != nil {
			t.Fatalf("seed post error: %v", err)
		}
		if err := db.GetDB().Create(&Post{ID: "r3", Title: "t3", Body: "b3", CreatedAt: ne}).Error; err != nil {
			t.Fatalf("seed post error: %v", err)
		}

		postRepo := db.GetPostRepository()
		if err := postRepo.CleanupExpiredPosts(7, 1); err != nil {
			t.Fatalf("CleanupExpiredPosts error: %v", err)
		}

		var cnt int64
		if err := db.GetDB().Model(&Post{}).Count(&cnt).Error; err != nil {
			t.Fatalf("count posts error: %v", err)
		}
		if cnt != 1 {
			t.Fatalf("unexpected remaining posts count: %d", cnt)
		}

		var remaining Post
		if err := db.GetDB().Where("id = ?", "r3").First(&remaining).Error; err != nil {
			t.Fatalf("remaining post fetch error: %v", err)
		}
		if remaining.ID != "r3" {
			t.Fatalf("unexpected remaining post: %+v", remaining)
		}
	})
}

func TestUserRepository_UpdatePassword(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	repo := db.GetUserRepository()
	user, err := repo.CreateUserWithPassword("update_me", "oldpass")
	if err != nil {
		t.Fatalf("seed user error: %v", err)
	}

	hashed, err := HashPassword("newpass")
	if err != nil {
		t.Fatalf("HashPassword error: %v", err)
	}
	if err := repo.UpdatePassword(user.ID, hashed); err != nil {
		t.Fatalf("UpdatePassword error: %v", err)
	}

	u2, err := repo.GetUserByID(user.ID)
	if err != nil {
		t.Fatalf("GetUserByID error: %v", err)
	}
	if CheckPassword("newpass", u2.Password) != nil {
		t.Fatalf("password not updated")
	}
}
