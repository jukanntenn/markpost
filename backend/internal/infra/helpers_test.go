package infra

import (
	"context"
	"errors"
	"strings"
	"testing"

	"markpost/internal/domain/post"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	if err := db.AutoMigrate(&post.Post{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	db.Create(&post.Post{QID: "p-1", Title: "a", Body: "body", UserID: 1})
	db.Create(&post.Post{QID: "p-2", Title: "b", Body: "body", UserID: 1})
	db.Create(&post.Post{QID: "p-3", Title: "c", Body: "body", UserID: 2})
	return db
}

func TestCountQuery(t *testing.T) {
	db := setupTestDB(t)

	tests := []struct {
		name  string
		query *gorm.DB
		want  int64
	}{
		{name: "all rows", query: db.Model(&post.Post{}), want: 3},
		{name: "filtered", query: db.Model(&post.Post{}).Where("title = ?", "a"), want: 1},
		{name: "no match", query: db.Model(&post.Post{}).Where("title = ?", "none"), want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := countQuery(context.Background(), tt.query, "label")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("countQuery() = %d, want %d", got, tt.want)
			}
		})
	}

	t.Run("error wrapping", func(t *testing.T) {
		_, err := countQuery(context.Background(), db.Table("nonexistent"), "MyLabel")
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "MyLabel") {
			t.Errorf("error %q should contain label %q", err.Error(), "MyLabel")
		}
	})
}

func TestFindMany(t *testing.T) {
	db := setupTestDB(t)

	t.Run("all rows", func(t *testing.T) {
		results, err := findMany[post.Post](context.Background(), db.Model(&post.Post{}), 0, 10, "label")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) != 3 {
			t.Errorf("got %d results, want 3", len(results))
		}
	})

	t.Run("with offset and limit", func(t *testing.T) {
		results, err := findMany[post.Post](context.Background(), db.Model(&post.Post{}), 1, 1, "label")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) != 1 {
			t.Errorf("got %d results, want 1", len(results))
		}
	})

	t.Run("no match", func(t *testing.T) {
		results, err := findMany[post.Post](context.Background(), db.Where("title = ?", "none"), 0, 10, "label")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) != 0 {
			t.Errorf("got %d results, want 0", len(results))
		}
	})

	t.Run("error wrapping", func(t *testing.T) {
		_, err := findMany[post.Post](context.Background(), db.Table("nonexistent"), 0, 10, "MyLabel")
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "MyLabel") {
			t.Errorf("error %q should contain label %q", err.Error(), "MyLabel")
		}
	})
}

func TestFindFirst(t *testing.T) {
	db := setupTestDB(t)

	t.Run("found", func(t *testing.T) {
		result, err := findFirst[post.Post](context.Background(), db.Where("qid = ?", "p-1"), post.ErrNotFound)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.QID != "p-1" {
			t.Errorf("QID = %q, want %q", result.QID, "p-1")
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := findFirst[post.Post](context.Background(), db.Where("qid = ?", "nonexistent"), post.ErrNotFound)
		if err == nil {
			t.Fatal("expected error")
		}
		if !errors.Is(err, post.ErrNotFound) {
			t.Errorf("error = %v, want wrapping of post.ErrNotFound", err)
		}
	})

	t.Run("error wrapping", func(t *testing.T) {
		_, err := findFirst[post.Post](context.Background(), db.Table("nonexistent"), post.ErrNotFound)
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestExistsBy(t *testing.T) {
	db := setupTestDB(t)

	t.Run("exists", func(t *testing.T) {
		ok, err := existsBy[post.Post](context.Background(), db, "qid", "p-1", "label")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !ok {
			t.Error("expected true")
		}
	})

	t.Run("not exists", func(t *testing.T) {
		ok, err := existsBy[post.Post](context.Background(), db, "qid", "nonexistent", "label")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ok {
			t.Error("expected false")
		}
	})

	t.Run("error wrapping", func(t *testing.T) {
		_, err := existsBy[post.Post](context.Background(), db.Table("nonexistent"), "qid", "val", "MyLabel")
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "MyLabel") {
			t.Errorf("error %q should contain label %q", err.Error(), "MyLabel")
		}
	})
}

func TestDeleteWhere(t *testing.T) {
	t.Run("deletes matching rows", func(t *testing.T) {
		db := setupTestDB(t)
		affected, err := deleteWhere[post.Post](context.Background(), db.Where("qid = ?", "p-1"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if affected != 1 {
			t.Errorf("rows affected = %d, want 1", affected)
		}
		var count int64
		db.Model(&post.Post{}).Count(&count)
		if count != 2 {
			t.Errorf("remaining rows = %d, want 2", count)
		}
	})

	t.Run("no matching rows", func(t *testing.T) {
		db := setupTestDB(t)
		affected, err := deleteWhere[post.Post](context.Background(), db.Where("qid = ?", "nonexistent"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if affected != 0 {
			t.Errorf("rows affected = %d, want 0", affected)
		}
	})

	t.Run("error", func(t *testing.T) {
		db := setupTestDB(t)
		_, err := deleteWhere[post.Post](context.Background(), db.Table("nonexistent").Where("1=1"))
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestLikeContains(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "plain", in: "hello", want: "%hello%"},
		{name: "percent", in: "50%", want: `%50\%%`},
		{name: "underscore", in: "a_b", want: `%a\_b%`},
		{name: "backslash", in: `a\b`, want: `%a\\b%`},
		{name: "combined", in: `a%b_c\d`, want: `%a\%b\_c\\d%`},
		{name: "empty", in: "", want: "%%"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := likeContains(tt.in)
			if got != tt.want {
				t.Errorf("likeContains(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
