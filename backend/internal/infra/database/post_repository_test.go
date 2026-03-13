package database

import (
	"testing"

	"markpost/internal/domain/post"
)

func TestPostRepository_Create(t *testing.T) {
	database := setupUserTestDatabase(t)
	userRepo := NewUserRepository(database.DB())
	postRepo := NewPostRepository(database.DB())

	u, err := userRepo.Create(nil, "test@example.com", "testuser", "password")
	if err != nil {
		t.Fatalf("Create user error: %v", err)
	}

	p, err := postRepo.Create(nil, "Test Title", "Test Body", u.ID)
	if err != nil {
		t.Fatalf("Create post error: %v", err)
	}

	if p == nil || p.QID == "" || p.Title != "Test Title" || p.Body != "Test Body" {
		t.Fatalf("unexpected post: %+v", p)
	}
}

func TestPostRepository_GetByQID(t *testing.T) {
	database := setupUserTestDatabase(t)
	userRepo := NewUserRepository(database.DB())
	postRepo := NewPostRepository(database.DB())

	u, err := userRepo.Create(nil, "test@example.com", "testuser", "password")
	if err != nil {
		t.Fatalf("Create user error: %v", err)
	}

	created, err := postRepo.Create(nil, "Test Title", "Test Body", u.ID)
	if err != nil {
		t.Fatalf("Create post error: %v", err)
	}

	p, err := postRepo.GetByQID(nil, created.QID)
	if err != nil {
		t.Fatalf("GetByQID error: %v", err)
	}

	if p == nil || p.QID != created.QID {
		t.Fatalf("unexpected post: %+v", p)
	}

	_, err = postRepo.GetByQID(nil, "nonexistent")
	if err == nil || err != post.ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestPostRepository_GetByUserID(t *testing.T) {
	database := setupUserTestDatabase(t)
	userRepo := NewUserRepository(database.DB())
	postRepo := NewPostRepository(database.DB())

	u, err := userRepo.Create(nil, "test@example.com", "testuser", "password")
	if err != nil {
		t.Fatalf("Create user error: %v", err)
	}

	_, err = postRepo.Create(nil, "Title 1", "Body 1", u.ID)
	if err != nil {
		t.Fatalf("Create post 1 error: %v", err)
	}

	_, err = postRepo.Create(nil, "Title 2", "Body 2", u.ID)
	if err != nil {
		t.Fatalf("Create post 2 error: %v", err)
	}

	posts, err := postRepo.GetByUserID(nil, u.ID, 0, 10)
	if err != nil {
		t.Fatalf("GetByUserID error: %v", err)
	}

	if len(posts) != 2 {
		t.Fatalf("expected 2 posts, got %d", len(posts))
	}
}

func TestPostRepository_CountByUserID(t *testing.T) {
	database := setupUserTestDatabase(t)
	userRepo := NewUserRepository(database.DB())
	postRepo := NewPostRepository(database.DB())

	u, err := userRepo.Create(nil, "test@example.com", "testuser", "password")
	if err != nil {
		t.Fatalf("Create user error: %v", err)
	}

	_, err = postRepo.Create(nil, "Title 1", "Body 1", u.ID)
	if err != nil {
		t.Fatalf("Create post 1 error: %v", err)
	}

	_, err = postRepo.Create(nil, "Title 2", "Body 2", u.ID)
	if err != nil {
		t.Fatalf("Create post 2 error: %v", err)
	}

	count, err := postRepo.CountByUserID(nil, u.ID)
	if err != nil {
		t.Fatalf("CountByUserID error: %v", err)
	}

	if count != 2 {
		t.Fatalf("expected count 2, got %d", count)
	}
}
