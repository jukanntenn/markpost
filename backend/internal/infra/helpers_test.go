package infra

import (
	"context"
	"errors"
	"strings"
	"testing"

	"markpost/internal/domain"
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
		result, err := findFirst[post.Post](context.Background(), db.Where("qid = ?", "p-1"), domain.ErrNotFound)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.QID != "p-1" {
			t.Errorf("QID = %q, want %q", result.QID, "p-1")
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := findFirst[post.Post](context.Background(), db.Where("qid = ?", "nonexistent"), domain.ErrNotFound)
		if err == nil {
			t.Fatal("expected error")
		}
		if !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("error = %v, want wrapping of domain.ErrNotFound", err)
		}
	})

	t.Run("error wrapping", func(t *testing.T) {
		_, err := findFirst[post.Post](context.Background(), db.Table("nonexistent"), domain.ErrNotFound)
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

func TestUpdateByID(t *testing.T) {
	db := setupTestDB(t)

	t.Run("updates existing row", func(t *testing.T) {
		err := updateByID[post.Post](context.Background(), db, 1, map[string]any{"title": "updated"}, "label")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		var p post.Post
		db.First(&p, 1)
		if p.Title != "updated" {
			t.Errorf("title = %q, want %q", p.Title, "updated")
		}
	})

	t.Run("not found", func(t *testing.T) {
		err := updateByID[post.Post](context.Background(), db, 999, map[string]any{"title": "x"}, "label")
		if err == nil {
			t.Fatal("expected error")
		}
		if !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("error = %v, want wrapping of domain.ErrNotFound", err)
		}
	})

	t.Run("error wrapping", func(t *testing.T) {
		err := updateByID[post.Post](context.Background(), db.Table("nonexistent"), 1, map[string]any{"title": "x"}, "MyLabel")
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

func TestBuildSearchCondition(t *testing.T) {
	t.Run("empty search", func(t *testing.T) {
		cond, args := buildSearchCondition("", "title")
		if cond != "" {
			t.Errorf("cond = %q, want empty", cond)
		}
		if args != nil {
			t.Errorf("args = %v, want nil", args)
		}
	})

	t.Run("empty fields", func(t *testing.T) {
		cond, args := buildSearchCondition("hello")
		if cond != "" {
			t.Errorf("cond = %q, want empty", cond)
		}
		if args != nil {
			t.Errorf("args = %v, want nil", args)
		}
	})

	t.Run("single field", func(t *testing.T) {
		cond, args := buildSearchCondition("test", "title")
		want := "title LIKE ?"
		if cond != want {
			t.Errorf("cond = %q, want %q", cond, want)
		}
		if len(args) != 1 || args[0] != likeContains("test") {
			t.Errorf("args = %v, want [%q]", args, likeContains("test"))
		}
	})

	t.Run("multiple fields", func(t *testing.T) {
		cond, args := buildSearchCondition("test", "title", "body")
		want := "title LIKE ? OR body LIKE ?"
		if cond != want {
			t.Errorf("cond = %q, want %q", cond, want)
		}
		if len(args) != 2 {
			t.Fatalf("len(args) = %d, want 2", len(args))
		}
		for i, a := range args {
			if a != likeContains("test") {
				t.Errorf("args[%d] = %v, want %q", i, a, likeContains("test"))
			}
		}
	})

	t.Run("special characters escaped", func(t *testing.T) {
		cond, args := buildSearchCondition("50%", "field")
		if cond != "field LIKE ?" {
			t.Errorf("cond = %q, want %q", cond, "field LIKE ?")
		}
		if len(args) != 1 {
			t.Fatalf("len(args) = %d, want 1", len(args))
		}
		if args[0] != likeContains("50%") {
			t.Errorf("args[0] = %v, want %q", args[0], likeContains("50%"))
		}
	})
}

func TestApplySearch(t *testing.T) {
	db := setupTestDB(t)

	tests := []struct {
		name   string
		search string
		fields []string
		want   int
	}{
		{name: "empty search", search: "", fields: []string{"title"}, want: 3},
		{name: "single field match", search: "a", fields: []string{"title"}, want: 1},
		{name: "single field no match", search: "z", fields: []string{"title"}, want: 0},
		{name: "multi field - match title", search: "a", fields: []string{"title", "body"}, want: 1},
		{name: "multi field - match body", search: "body", fields: []string{"title", "body"}, want: 3},
		{name: "no fields", search: "a", fields: []string{}, want: 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := applySearch(db.Model(&post.Post{}), tt.search, tt.fields...)
			results, err := findMany[post.Post](context.Background(), query, 0, 100, "test")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(results) != tt.want {
				t.Errorf("got %d results, want %d", len(results), tt.want)
			}
		})
	}
}
