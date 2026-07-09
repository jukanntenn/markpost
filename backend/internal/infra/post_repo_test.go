package infra

import (
	"context"
	"errors"
	"testing"
	"time"

	"markpost/internal/domain"
	"markpost/internal/domain/post"
)

func TestPostRepository_Create(t *testing.T) {
	db := SetupTestDB(t)
	repo := NewPostRepository(db)
	ctx := context.Background()

	p, err := repo.Create(ctx, "Test Title", "Test Body", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.QID == "" {
		t.Error("expected non-empty QID")
	}
	if p.Title != "Test Title" {
		t.Errorf("title = %q, want %q", p.Title, "Test Title")
	}
	if p.Body != "Test Body" {
		t.Errorf("body = %q, want %q", p.Body, "Test Body")
	}
	if p.UserID != 1 {
		t.Errorf("user_id = %d, want 1", p.UserID)
	}
}

func TestPostRepository_GetByQID(t *testing.T) {
	db := SetupTestDB(t)
	repo := NewPostRepository(db)
	ctx := context.Background()

	created, _ := repo.Create(ctx, "Title", "Body", 1)

	t.Run("finds existing post", func(t *testing.T) {
		p, err := repo.GetByQID(ctx, created.QID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if p.Title != "Title" {
			t.Errorf("title = %q, want %q", p.Title, "Title")
		}
	})

	t.Run("returns ErrNotFound for missing", func(t *testing.T) {
		_, err := repo.GetByQID(ctx, "nonexistent")
		if !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("expected ErrNotFound, got: %v", err)
		}
	})
}

func TestPostRepository_GetByID(t *testing.T) {
	db := SetupTestDB(t)
	repo := NewPostRepository(db)
	ctx := context.Background()

	created, _ := repo.Create(ctx, "Title", "Body", 1)

	p, err := repo.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.QID != created.QID {
		t.Errorf("qid = %q, want %q", p.QID, created.QID)
	}
}

func TestPostRepository_CountByUserID(t *testing.T) {
	db := SetupTestDB(t)
	repo := NewPostRepository(db)
	ctx := context.Background()

	_, _ = repo.Create(ctx, "T1", "B1", 1)
	_, _ = repo.Create(ctx, "T2", "B2", 1)
	_, _ = repo.Create(ctx, "T3", "B3", 2)

	count, err := repo.CountByUserID(ctx, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}
}

func TestPostRepository_GetByUserID(t *testing.T) {
	db := SetupTestDB(t)
	repo := NewPostRepository(db)
	ctx := context.Background()

	_, _ = repo.Create(ctx, "T1", "B1", 1)
	_, _ = repo.Create(ctx, "T2", "B2", 1)
	_, _ = repo.Create(ctx, "T3", "B3", 2)

	posts, err := repo.GetByUserID(ctx, 1, 0, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(posts) != 2 {
		t.Errorf("got %d posts, want 2", len(posts))
	}
}

func TestPostRepository_ListAll(t *testing.T) {
	db := SetupTestDB(t)
	repo := NewPostRepository(db)
	ctx := context.Background()

	_, _ = repo.Create(ctx, "Alpha", "Body", 1)
	_, _ = repo.Create(ctx, "Beta", "Body", 2)

	t.Run("returns all posts", func(t *testing.T) {
		posts, err := repo.ListAll(ctx, "", 0, 10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(posts) != 2 {
			t.Errorf("got %d posts, want 2", len(posts))
		}
	})

	t.Run("filters by search", func(t *testing.T) {
		posts, err := repo.ListAll(ctx, "Alpha", 0, 10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(posts) != 1 {
			t.Errorf("got %d posts, want 1", len(posts))
		}
	})
}

func TestPostRepository_CountAll(t *testing.T) {
	db := SetupTestDB(t)
	repo := NewPostRepository(db)
	ctx := context.Background()

	_, _ = repo.Create(ctx, "Alpha", "Body", 1)
	_, _ = repo.Create(ctx, "Beta", "Body", 2)

	count, err := repo.CountAll(ctx, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}
}

func TestPostRepository_DeleteByID(t *testing.T) {
	db := SetupTestDB(t)
	repo := NewPostRepository(db)
	ctx := context.Background()

	created, _ := repo.Create(ctx, "Title", "Body", 1)

	affected, err := repo.DeleteByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if affected != 1 {
		t.Errorf("affected = %d, want 1", affected)
	}

	_, err = repo.GetByID(ctx, created.ID)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

func TestPostRepository_DeleteByQID(t *testing.T) {
	t.Run("owner-scoped delete by correct owner removes the row", func(t *testing.T) {
		db := SetupTestDB(t)
		repo := NewPostRepository(db)
		ctx := context.Background()

		created, _ := repo.Create(ctx, "Title", "Body", 1)
		affected, err := repo.DeleteByQID(ctx, created.QID, 1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if affected != 1 {
			t.Fatalf("affected = %d, want 1", affected)
		}
		if _, err := repo.GetByQID(ctx, created.QID); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("expected ErrNotFound after delete, got: %v", err)
		}
	})

	t.Run("owner-scoped delete by wrong owner affects 0 rows", func(t *testing.T) {
		db := SetupTestDB(t)
		repo := NewPostRepository(db)
		ctx := context.Background()

		created, _ := repo.Create(ctx, "Title", "Body", 1)
		affected, err := repo.DeleteByQID(ctx, created.QID, 2)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if affected != 0 {
			t.Errorf("affected = %d, want 0 (wrong owner)", affected)
		}
		if _, err := repo.GetByQID(ctx, created.QID); err != nil {
			t.Errorf("post must still exist after wrong-owner delete: %v", err)
		}
	})

	t.Run("admin delete (ownerID=0) removes any owner's row", func(t *testing.T) {
		db := SetupTestDB(t)
		repo := NewPostRepository(db)
		ctx := context.Background()

		created, _ := repo.Create(ctx, "Title", "Body", 1)
		affected, err := repo.DeleteByQID(ctx, created.QID, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if affected != 1 {
			t.Fatalf("affected = %d, want 1", affected)
		}
		if _, err := repo.GetByQID(ctx, created.QID); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("expected ErrNotFound after admin delete, got: %v", err)
		}
	})

	t.Run("nonexistent QID affects 0 rows", func(t *testing.T) {
		db := SetupTestDB(t)
		repo := NewPostRepository(db)
		ctx := context.Background()

		affected, err := repo.DeleteByQID(ctx, "p-missing", 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if affected != 0 {
			t.Errorf("affected = %d, want 0 for missing QID", affected)
		}
	})
}

func TestPostRepository_CreateBatch(t *testing.T) {
	db := SetupTestDB(t)
	repo := NewPostRepository(db)
	ctx := context.Background()

	t.Run("creates multiple posts", func(t *testing.T) {
		posts := []post.Post{
			{QID: "p-batch1", Title: "T1", Body: "B1", UserID: 1},
			{QID: "p-batch2", Title: "T2", Body: "B2", UserID: 1},
		}
		count, err := repo.CreateBatch(ctx, posts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if count != 2 {
			t.Errorf("count = %d, want 2", count)
		}
	})

	t.Run("empty batch returns 0", func(t *testing.T) {
		count, err := repo.CreateBatch(ctx, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if count != 0 {
			t.Errorf("count = %d, want 0", count)
		}
	})
}

func TestPostRepository_PruneExpired(t *testing.T) {
	db := SetupTestDB(t)
	repo := NewPostRepository(db)
	ctx := context.Background()

	// Create a post with an old timestamp
	p, _ := repo.Create(ctx, "Old Post", "Body", 1)
	db.Model(&p).Update("created_at", time.Now().AddDate(0, 0, -10))

	_, _ = repo.Create(ctx, "New Post", "Body", 1)

	pruned, err := repo.PruneExpired(ctx, 7, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pruned) != 1 || pruned[0] != p.QID {
		t.Errorf("pruned QIDs = %v, want [%s]", pruned, p.QID)
	}

	count, _ := repo.CountAll(ctx, "")
	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}
}

func TestPostRepository_CountExpired(t *testing.T) {
	db := SetupTestDB(t)
	repo := NewPostRepository(db)
	ctx := context.Background()

	p, _ := repo.Create(ctx, "Old Post", "Body", 1)
	db.Model(&p).Update("created_at", time.Now().AddDate(0, 0, -10))

	_, _ = repo.Create(ctx, "New Post", "Body", 1)

	count, err := repo.CountExpired(ctx, 7)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}
}
