package main

import (
	"database/sql"
	"testing"
	"time"
)

func setupTestDB(t *testing.T) *Database {
	t.Helper()
	db, err := NewTestDatabase()
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
		_, err := makeUser("alice", "password", &githubID)
		if err != nil {
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
		_, err := makeUser("alice", "password", &githubID)
		if err != nil {
			t.Fatalf("seed user error: %v", err)
		}

		_, err = repo.GetUserByGitHubID(int64(123))
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
		u, err := makeUser("alice", "password", nil)
		if err != nil {
			t.Fatalf("seed user error: %v", err)
		}

		got, err := repo.GetUserByPostKey(u.PostKey)
		if err != nil {
			t.Fatalf("GetUserByPostKey error: %v", err)
		}

		if got == nil || got.Username != "alice" || got.PostKey != u.PostKey {
			t.Fatalf("unexpected user: %+v", got)
		}
	})

	t.Run("returns sql.ErrNoRows for wrong postKey", func(t *testing.T) {
		db := setupTestDB(t)
		defer teardownTestDB(t, db)

		repo := db.GetUserRepository()
		_, err := makeUser("alice", "password", nil)
		if err != nil {
			t.Fatalf("seed user error: %v", err)
		}

		_, err = repo.GetUserByPostKey("not-exist")
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
		u, err := makeUser("alice", "password", nil)
		if err != nil {
			t.Fatalf("seed user error: %v", err)
		}

		got, err := repo.GetUserByID(u.ID)
		if err != nil {
			t.Fatalf("GetUserByID error: %v", err)
		}

		if got == nil || got.ID != u.ID || got.Username != "alice" {
			t.Fatalf("unexpected user: %+v", got)
		}
	})

	t.Run("returns sql.ErrNoRows for wrong ID", func(t *testing.T) {
		db := setupTestDB(t)
		defer teardownTestDB(t, db)

		repo := db.GetUserRepository()
		_, err := makeUser("alice", "password", nil)
		if err != nil {
			t.Fatalf("seed user error: %v", err)
		}

		_, err = repo.GetUserByID(123456)
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
		_, err := makeUser("alice", "password", nil)
		if err != nil {
			t.Fatalf("seed user error: %v", err)
		}

		got, err := repo.GetUserByUsername("alice")
		if err != nil {
			t.Fatalf("GetUserByUsername error: %v", err)
		}

		if got == nil || got.ID == 0 || got.Username != "alice" {
			t.Fatalf("unexpected user: %+v", got)
		}
	})

	t.Run("returns sql.ErrNoRows for wrong username", func(t *testing.T) {
		db := setupTestDB(t)
		defer teardownTestDB(t, db)

		repo := db.GetUserRepository()
		_, err := makeUser("alice", "password", nil)
		if err != nil {
			t.Fatalf("seed user error: %v", err)
		}

		_, err = repo.GetUserByUsername("unknown")
		if err == nil || err != sql.ErrNoRows {
			t.Fatalf("expected sql.ErrNoRows, got %v", err)
		}
	})
}

func TestUserRepository_CreateUserFromGitHub(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	repo := db.GetUserRepository()
	gu := &GitHubUser{ID: 7777, Login: "alice"}
	got, err := repo.CreateUserFromGitHub(gu)
	if err != nil {
		t.Fatalf("CreateUserFromGitHub error: %v", err)
	}
	if got == nil || got.ID == 0 || got.Username != "alice" || got.Password != "" || got.GitHubID == nil || *got.GitHubID != 7777 || got.PostKey == "" {
		t.Fatalf("unexpected user: %+v", got)
	}

	got2, err := repo.GetUserByGitHubID(7777)
	if err != nil || got2 == nil || got2.Username != "alice" {
		t.Fatalf("user not persisted: %v %+v", err, got2)
	}
}

func TestUserRepository_GetOrCreateUserFromGitHub(t *testing.T) {
	t.Run("returns existing user for valid GitHubUser", func(t *testing.T) {
		db := setupTestDB(t)
		defer teardownTestDB(t, db)

		repo := db.GetUserRepository()
		githubID := int64(1000)
		u, err := makeUser("alice", "password", &githubID)
		if err != nil {
			t.Fatalf("seed user error: %v", err)
		}

		gu := &GitHubUser{ID: githubID, Login: "other"}
		got, err := repo.GetOrCreateUserFromGitHub(gu)
		if err != nil {
			t.Fatalf("GetOrCreateUserFromGitHub error: %v", err)
		}
		if got == nil || got.ID != u.ID || got.Username != "alice" || got.GitHubID == nil || *got.GitHubID != githubID {
			t.Fatalf("unexpected user: %+v", got)
		}
	})

	t.Run("creates new user for non-existing GitHubID", func(t *testing.T) {
		db := setupTestDB(t)
		defer teardownTestDB(t, db)

		repo := db.GetUserRepository()
		gu := &GitHubUser{ID: 2000, Login: "alice"}
		got, err := repo.GetOrCreateUserFromGitHub(gu)
		if err != nil {
			t.Fatalf("GetOrCreateUserFromGitHub error: %v", err)
		}
		if got == nil || got.Username != "alice" || got.GitHubID == nil || *got.GitHubID != 2000 || got.PostKey == "" {
			t.Fatalf("unexpected user: %+v", got)
		}

		got2, err := repo.GetUserByGitHubID(2000)
		if err != nil || got2 == nil || got2.Username != "alice" {
			t.Fatalf("user not persisted: %v %+v", err, got2)
		}
	})
}

func TestUserRepository_CreateUser(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	repo := db.GetUserRepository()
	got, err := repo.CreateUser("alice", "p@ssw0rd")
	if err != nil {
		t.Fatalf("CreateUser error: %v", err)
	}
	if got == nil || got.ID == 0 || got.Username != "alice" || got.Password == "" || got.PostKey == "" {
		t.Fatalf("unexpected user: %+v", got)
	}
	if CheckPassword("p@ssw0rd", got.Password) != nil {
		t.Fatalf("password not matched: %v", err)
	}

	got2, err := repo.GetUserByUsername("alice")
	if err != nil || got2 == nil || got2.Username != "alice" {
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
		_, err := makeUser("alice", "", nil)
		if err != nil {
			t.Fatalf("seed user error: %v", err)
		}
		_, err = repo.ValidateUserPassword("alice", "any")
		if err == nil || err.Error() != "user has no password set" {
			t.Fatalf("expected 'user has no password set', got %v", err)
		}
	})

	t.Run("password mismatch returns expected error", func(t *testing.T) {
		db := setupTestDB(t)
		defer teardownTestDB(t, db)

		repo := db.GetUserRepository()
		_, err := makeUser("alice", "right", nil)
		if err != nil {
			t.Fatalf("seed user error: %v", err)
		}
		_, err = repo.ValidateUserPassword("alice", "wrong")
		if err == nil || err.Error() != "invalid password" {
			t.Fatalf("expected 'invalid password', got %v", err)
		}
	})

	t.Run("password match returns user with nil error", func(t *testing.T) {
		db := setupTestDB(t)
		defer teardownTestDB(t, db)

		repo := db.GetUserRepository()
		_, err := makeUser("alice", "password", nil)
		if err != nil {
			t.Fatalf("seed user error: %v", err)
		}
		got, err := repo.ValidateUserPassword("alice", "password")
		if err != nil {
			t.Fatalf("ValidateUserPassword error: %v", err)
		}
		if got == nil || got.Username != "alice" || got.ID == 0 {
			t.Fatalf("unexpected user: %+v", got)
		}
	})
}

func TestUserRepository_UpdateUserPassword(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	repo := db.GetUserRepository()
	user, err := repo.CreateUser("alice", "password")
	if err != nil {
		t.Fatalf("seed user error: %v", err)
	}

	hashed, err := HashPassword("newpassword")
	if err != nil {
		t.Fatalf("HashPassword error: %v", err)
	}
	if err = repo.UpdateUserPassword(user.ID, hashed); err != nil {
		t.Fatalf("UpdatePassword error: %v", err)
	}

	u2, err := repo.GetUserByID(user.ID)
	if err != nil {
		t.Fatalf("GetUserByID error: %v", err)
	}
	if CheckPassword("newpassword", u2.Password) != nil {
		t.Fatalf("password not updated")
	}
}

func TestPostRepository_CreatePost(t *testing.T) {
	t.Run("creates post with invalid user", func(t *testing.T) {
		db := setupTestDB(t)
		defer teardownTestDB(t, db)

		postRepo := db.GetPostRepository()

		invalidUserID := 99999
		_, err := postRepo.CreatePost("title", "body", invalidUserID)
		if err == nil {
			t.Fatalf("expected error when creating post with invalid user ID, got nil")
		}
	})

	t.Run("creates post with valid user", func(t *testing.T) {
		db := setupTestDB(t)
		defer teardownTestDB(t, db)

		userRepo := db.GetUserRepository()
		u, err := userRepo.CreateUser("alice", "password")
		if err != nil {
			t.Fatalf("seed user error: %v", err)
		}

		postRepo := db.GetPostRepository()
		p, err := postRepo.CreatePost("title", "body", u.ID)
		if err != nil {
			t.Fatalf("CreatePost error: %v", err)
		}
		if p == nil || p.QID == "" || p.Title != "title" || p.Body != "body" || p.UserID != u.ID {
			t.Fatalf("unexpected post: %+v", p)
		}
	})
}

func TestPostRepository_GetPostByQID(t *testing.T) {
	t.Run("returns expected record for valid qid", func(t *testing.T) {
		db := setupTestDB(t)
		defer teardownTestDB(t, db)

		userRepo := db.GetUserRepository()
		u, err := userRepo.CreateUser("alice", "password")
		if err != nil {
			t.Fatalf("seed user error: %v", err)
		}

		postRepo := db.GetPostRepository()
		p, err := postRepo.CreatePost("title", "body", u.ID)
		if err != nil {
			t.Fatalf("CreatePost error: %v", err)
		}
		got, err := postRepo.GetPostByQID(p.QID)
		if err != nil {
			t.Fatalf("GetPostByQID error: %v", err)
		}
		if got == nil || got.QID != p.QID || got.Title != "title" || got.Body != "body" || got.UserID != u.ID {
			t.Fatalf("unexpected post: %+v", got)
		}
	})

	t.Run("returns sql.ErrNoRows for wrong qid", func(t *testing.T) {
		db := setupTestDB(t)
		defer teardownTestDB(t, db)

		postRepo := db.GetPostRepository()
		_, err := postRepo.GetPostByQID("non-existent-qid")
		if err == nil || err != sql.ErrNoRows {
			t.Fatalf("expected sql.ErrNoRows, got %v", err)
		}
	})
}

func TestPostRepository_GetPostsByUserID(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	userRepo := db.GetUserRepository()
	u1, _ := userRepo.CreateUser("alice", "password")
	u2, _ := userRepo.CreateUser("bob", "password")

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
		if p.UserID != u1.ID {
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

		userRepo := db.GetUserRepository()
		u, _ := userRepo.CreateUser("u", "p")

		now := time.Now().UTC()
		expired1 := now.AddDate(0, 0, -10)
		expired2 := now.AddDate(0, 0, -8)
		notExpired := now.AddDate(0, 0, -3)

		if err := db.GetDB().Create(&Post{QID: "p1", Title: "t1", Body: "b1", CreatedAt: expired1, UserID: u.ID}).Error; err != nil {
			t.Fatalf("seed post error: %v", err)
		}
		if err := db.GetDB().Create(&Post{QID: "p2", Title: "t2", Body: "b2", CreatedAt: expired2, UserID: u.ID}).Error; err != nil {
			t.Fatalf("seed post error: %v", err)
		}
		if err := db.GetDB().Create(&Post{QID: "p3", Title: "t3", Body: "b3", CreatedAt: notExpired, UserID: u.ID}).Error; err != nil {
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

		userRepo := db.GetUserRepository()
		u, _ := userRepo.CreateUser("u", "p")

		now := time.Now().UTC()
		e1 := now.AddDate(0, 0, -12)
		e2 := now.AddDate(0, 0, -10)
		e3 := now.AddDate(0, 0, -9)
		ne := now.AddDate(0, 0, -2)

		if err := db.GetDB().Create(&Post{QID: "q1", Title: "t1", Body: "b1", CreatedAt: e1, UserID: u.ID}).Error; err != nil {
			t.Fatalf("seed post error: %v", err)
		}
		if err := db.GetDB().Create(&Post{QID: "q2", Title: "t2", Body: "b2", CreatedAt: e2, UserID: u.ID}).Error; err != nil {
			t.Fatalf("seed post error: %v", err)
		}
		if err := db.GetDB().Create(&Post{QID: "q3", Title: "t3", Body: "b3", CreatedAt: e3, UserID: u.ID}).Error; err != nil {
			t.Fatalf("seed post error: %v", err)
		}
		if err := db.GetDB().Create(&Post{QID: "q4", Title: "t4", Body: "b4", CreatedAt: ne, UserID: u.ID}).Error; err != nil {
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

		userRepo := db.GetUserRepository()
		u, _ := userRepo.CreateUser("u", "p")

		now := time.Now().UTC()
		e1 := now.AddDate(0, 0, -20)
		e2 := now.AddDate(0, 0, -15)
		ne := now.AddDate(0, 0, -3)

		if err := db.GetDB().Create(&Post{QID: "r1", Title: "t1", Body: "b1", CreatedAt: e1, UserID: u.ID}).Error; err != nil {
			t.Fatalf("seed post error: %v", err)
		}
		if err := db.GetDB().Create(&Post{QID: "r2", Title: "t2", Body: "b2", CreatedAt: e2, UserID: u.ID}).Error; err != nil {
			t.Fatalf("seed post error: %v", err)
		}
		if err := db.GetDB().Create(&Post{QID: "r3", Title: "t3", Body: "b3", CreatedAt: ne, UserID: u.ID}).Error; err != nil {
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
		if err := db.GetDB().Where("qid = ?", "r3").First(&remaining).Error; err != nil {
			t.Fatalf("remaining post fetch error: %v", err)
		}
		if remaining.QID != "r3" {
			t.Fatalf("unexpected remaining post: %+v", remaining)
		}
	})
}
