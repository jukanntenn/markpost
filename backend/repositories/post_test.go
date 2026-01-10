package repositories

import (
	"fmt"
	"testing"
	"time"

	"markpost/models"
)

func setupPostTestDatabase(t *testing.T) *models.Database {
	t.Helper()

	database, err := models.NewTestDatabase()
	if err != nil {
		t.Fatalf("NewTestDatabase error: %v", err)
	}

	return database
}

func TestPostRepository_CreatePost(t *testing.T) {
	t.Run("creates post with valid data", func(t *testing.T) {
		database := setupPostTestDatabase(t)
		userRepo := NewUserRepo(database)
		postRepo := NewPostRepo(database)

		user, err := userRepo.CreateUser("alice", "password")
		if err != nil {
			t.Fatalf("seed user error: %v", err)
		}

		title := "Test Post Title"
		body := "Test post body content"
		post, err := postRepo.CreatePost(title, body, user.ID)
		if err != nil {
			t.Fatalf("CreatePost error: %v", err)
		}

		if post == nil || post.Title != title || post.Body != body || post.UserID != user.ID || post.QID == "" {
			t.Fatalf("unexpected post: %+v", post)
		}
	})

	t.Run("creates post with unique QID", func(t *testing.T) {
		database := setupPostTestDatabase(t)
		userRepo := NewUserRepo(database)
		postRepo := NewPostRepo(database)

		user, err := userRepo.CreateUser("alice", "password")
		if err != nil {
			t.Fatalf("seed user error: %v", err)
		}

		post1, err := postRepo.CreatePost("Title 1", "Body 1", user.ID)
		if err != nil {
			t.Fatalf("CreatePost error: %v", err)
		}

		post2, err := postRepo.CreatePost("Title 2", "Body 2", user.ID)
		if err != nil {
			t.Fatalf("CreatePost error: %v", err)
		}

		if post1.QID == post2.QID {
			t.Fatalf("QIDs should be unique: %s", post1.QID)
		}
	})
}

func TestPostRepository_GetPostByQID(t *testing.T) {
	t.Run("returns expected post for valid QID", func(t *testing.T) {
		database := setupPostTestDatabase(t)
		userRepo := NewUserRepo(database)
		postRepo := NewPostRepo(database)

		user, err := userRepo.CreateUser("alice", "password")
		if err != nil {
			t.Fatalf("seed user error: %v", err)
		}

		originalPost, err := postRepo.CreatePost("Test Title", "Test Body", user.ID)
		if err != nil {
			t.Fatalf("CreatePost error: %v", err)
		}

		got, err := postRepo.GetPostByQID(originalPost.QID)
		if err != nil {
			t.Fatalf("GetPostByQID error: %v", err)
		}

		if got == nil || got.ID != originalPost.ID || got.QID != originalPost.QID {
			t.Fatalf("unexpected post: %+v", got)
		}
	})

	t.Run("returns models.ErrNotFound for wrong QID", func(t *testing.T) {
		database := setupPostTestDatabase(t)
		postRepo := NewPostRepo(database)

		_, err := postRepo.GetPostByQID("not-exist")
		if err == nil || err != models.ErrNotFound {
			t.Fatalf("expected models.ErrNotFound, got %v", err)
		}
	})
}

func TestPostRepository_CountPostsByUserID(t *testing.T) {
	t.Run("returns correct count for user with posts", func(t *testing.T) {
		database := setupPostTestDatabase(t)
		userRepo := NewUserRepo(database)
		postRepo := NewPostRepo(database)

		user, err := userRepo.CreateUser("alice", "password")
		if err != nil {
			t.Fatalf("seed user error: %v", err)
		}

		_, err = postRepo.CreatePost("Post 1", "Body 1", user.ID)
		if err != nil {
			t.Fatalf("CreatePost error: %v", err)
		}

		_, err = postRepo.CreatePost("Post 2", "Body 2", user.ID)
		if err != nil {
			t.Fatalf("CreatePost error: %v", err)
		}

		count, err := postRepo.CountPostsByUserID(user.ID)
		if err != nil {
			t.Fatalf("CountPostsByUserID error: %v", err)
		}

		if count != 2 {
			t.Fatalf("expected count 2, got %d", count)
		}
	})

	t.Run("returns zero for user with no posts", func(t *testing.T) {
		database := setupPostTestDatabase(t)
		userRepo := NewUserRepo(database)
		postRepo := NewPostRepo(database)

		user, err := userRepo.CreateUser("alice", "password")
		if err != nil {
			t.Fatalf("seed user error: %v", err)
		}

		count, err := postRepo.CountPostsByUserID(user.ID)
		if err != nil {
			t.Fatalf("CountPostsByUserID error: %v", err)
		}

		if count != 0 {
			t.Fatalf("expected count 0, got %d", count)
		}
	})
}

func TestPostRepository_GetPostsByUserID(t *testing.T) {
	t.Run("returns posts for user with pagination", func(t *testing.T) {
		database := setupPostTestDatabase(t)
		userRepo := NewUserRepo(database)
		postRepo := NewPostRepo(database)

		user, err := userRepo.CreateUser("alice", "password")
		if err != nil {
			t.Fatalf("seed user error: %v", err)
		}

		_, err = postRepo.CreatePost("Post 1", "Body 1", user.ID)
		if err != nil {
			t.Fatalf("CreatePost error: %v", err)
		}

		_, err = postRepo.CreatePost("Post 2", "Body 2", user.ID)
		if err != nil {
			t.Fatalf("CreatePost error: %v", err)
		}

		posts, err := postRepo.GetPostsByUserID(user.ID, 0, 10)
		if err != nil {
			t.Fatalf("GetPostsByUserID error: %v", err)
		}

		if len(posts) != 2 {
			t.Fatalf("expected 2 posts, got %d", len(posts))
		}

		for _, post := range posts {
			if post.UserID != user.ID {
				t.Fatalf("post has wrong user ID: %d", post.UserID)
			}
		}
	})

	t.Run("returns empty slice for user with no posts", func(t *testing.T) {
		database := setupPostTestDatabase(t)
		userRepo := NewUserRepo(database)
		postRepo := NewPostRepo(database)

		user, err := userRepo.CreateUser("alice", "password")
		if err != nil {
			t.Fatalf("seed user error: %v", err)
		}

		posts, err := postRepo.GetPostsByUserID(user.ID, 0, 10)
		if err != nil {
			t.Fatalf("GetPostsByUserID error: %v", err)
		}

		if len(posts) != 0 {
			t.Fatalf("expected 0 posts, got %d", len(posts))
		}
	})

	t.Run("respects pagination limits", func(t *testing.T) {
		database := setupPostTestDatabase(t)
		userRepo := NewUserRepo(database)
		postRepo := NewPostRepo(database)

		user, err := userRepo.CreateUser("alice", "password")
		if err != nil {
			t.Fatalf("seed user error: %v", err)
		}

		for range 5 {
			_, err = postRepo.CreatePost("Post", "Body", user.ID)
			if err != nil {
				t.Fatalf("CreatePost error: %v", err)
			}
		}

		posts, err := postRepo.GetPostsByUserID(user.ID, 0, 2)
		if err != nil {
			t.Fatalf("GetPostsByUserID error: %v", err)
		}

		if len(posts) != 2 {
			t.Fatalf("expected 2 posts with limit 2, got %d", len(posts))
		}
	})
}

func TestPostRepository_CountExpiredPosts(t *testing.T) {
	t.Run("returns error for invalid retention days", func(t *testing.T) {
		database := setupPostTestDatabase(t)
		postRepo := NewPostRepo(database)

		_, err := postRepo.CountExpiredPosts(0)
		if err == nil || err.Error() != "retention days must be positive, got: 0" {
			t.Fatalf("expected 'retention days must be positive' error, got %v", err)
		}
	})

	t.Run("counts expired posts correctly", func(t *testing.T) {
		database := setupPostTestDatabase(t)
		userRepo := NewUserRepo(database)
		postRepo := NewPostRepo(database)

		user, err := userRepo.CreateUser("alice", "password")
		if err != nil {
			t.Fatalf("seed user error: %v", err)
		}

		post := &models.Post{
			QID:    "test-qid",
			Title:  "Test Post",
			Body:   "Test Body",
			UserID: user.ID,
		}

		err = post.Create(database)
		if err != nil {
			t.Fatalf("create post error: %v", err)
		}

		// Manually set created_at to simulate expired post
		oldTime := time.Now().AddDate(0, 0, -10)
		err = database.DB().Model(post).Update("created_at", oldTime).Error
		if err != nil {
			t.Fatalf("update created_at error: %v", err)
		}

		count, err := postRepo.CountExpiredPosts(5)
		if err != nil {
			t.Fatalf("GetExpiredPostsCount error: %v", err)
		}

		if count != 1 {
			t.Fatalf("expected 1 expired post, got %d", count)
		}
	})

	t.Run("returns zero when no posts are expired", func(t *testing.T) {
		database := setupPostTestDatabase(t)
		userRepo := NewUserRepo(database)
		postRepo := NewPostRepo(database)

		user, err := userRepo.CreateUser("alice", "password")
		if err != nil {
			t.Fatalf("seed user error: %v", err)
		}

		_, err = postRepo.CreatePost("Recent Post", "Recent Body", user.ID)
		if err != nil {
			t.Fatalf("CreatePost error: %v", err)
		}

		count, err := postRepo.CountExpiredPosts(30)
		if err != nil {
			t.Fatalf("GetExpiredPostsCount error: %v", err)
		}

		if count != 0 {
			t.Fatalf("expected 0 expired posts, got %d", count)
		}
	})
}

func TestPostRepository_PruneExpiredPosts(t *testing.T) {
	t.Run("returns error for invalid retention days", func(t *testing.T) {
		database := setupPostTestDatabase(t)
		postRepo := NewPostRepo(database)

		err := postRepo.PruneExpiredPosts(0, 100)
		if err == nil || err.Error() != "retention days must be positive, got: 0" {
			t.Fatalf("expected 'retention days must be positive' error, got %v", err)
		}
	})

	t.Run("deletes expired posts in batches", func(t *testing.T) {
		database := setupPostTestDatabase(t)
		userRepo := NewUserRepo(database)
		postRepo := NewPostRepo(database)

		user, err := userRepo.CreateUser("alice", "password")
		if err != nil {
			t.Fatalf("seed user error: %v", err)
		}

		// Create 5 expired posts
		for i := range 5 {
			post := &models.Post{
				QID:    fmt.Sprintf("test-qid-%d", i),
				Title:  fmt.Sprintf("Post %d", i),
				Body:   fmt.Sprintf("Body %d", i),
				UserID: user.ID,
			}

			err = post.Create(database)
			if err != nil {
				t.Fatalf("create post error: %v", err)
			}

			// Set created_at to simulate expired posts
			oldTime := time.Now().AddDate(0, 0, -10)
			err = database.DB().Model(post).Update("created_at", oldTime).Error
			if err != nil {
				t.Fatalf("update created_at error: %v", err)
			}
		}

		// Create 2 recent posts that should not be deleted
		_, err = postRepo.CreatePost("Recent Post 1", "Recent Body 1", user.ID)
		if err != nil {
			t.Fatalf("CreatePost error: %v", err)
		}

		_, err = postRepo.CreatePost("Recent Post 2", "Recent Body 2", user.ID)
		if err != nil {
			t.Fatalf("CreatePost error: %v", err)
		}

		err = postRepo.PruneExpiredPosts(5, 2)
		if err != nil {
			t.Fatalf("CleanupExpiredPosts error: %v", err)
		}

		// Check that expired posts are deleted
		count, err := postRepo.CountExpiredPosts(5)
		if err != nil {
			t.Fatalf("GetExpiredPostsCount error: %v", err)
		}

		if count != 0 {
			t.Fatalf("expected 0 expired posts after cleanup, got %d", count)
		}

		// Check that recent posts still exist
		remainingCount, err := postRepo.CountPostsByUserID(user.ID)
		if err != nil {
			t.Fatalf("CountPostsByUserID error: %v", err)
		}

		if remainingCount != 2 {
			t.Fatalf("expected 2 remaining posts, got %d", remainingCount)
		}
	})

	t.Run("handles default batch size when batchSize is zero", func(t *testing.T) {
		database := setupPostTestDatabase(t)
		userRepo := NewUserRepo(database)
		postRepo := NewPostRepo(database)

		user, err := userRepo.CreateUser("alice", "password")
		if err != nil {
			t.Fatalf("seed user error: %v", err)
		}

		// Create expired posts
		for i := range 5 {
			post := &models.Post{
				QID:    fmt.Sprintf("test-qid-%d", i),
				Title:  fmt.Sprintf("Post %d", i),
				Body:   fmt.Sprintf("Body %d", i),
				UserID: user.ID,
			}

			err = post.Create(database)
			if err != nil {
				t.Fatalf("create post error: %v", err)
			}

			// Set created_at to simulate expired posts
			oldTime := time.Now().AddDate(0, 0, -10)
			err = database.DB().Model(post).Update("created_at", oldTime).Error
			if err != nil {
				t.Fatalf("update created_at error: %v", err)
			}
		}

		// Use 0 to trigger default batch size
		err = postRepo.PruneExpiredPosts(5, 0)
		if err != nil {
			t.Fatalf("CleanupExpiredPosts error: %v", err)
		}

		count, err := postRepo.CountExpiredPosts(5)
		if err != nil {
			t.Fatalf("GetExpiredPostsCount error: %v", err)
		}

		if count != 0 {
			t.Fatalf("expected 0 expired posts after cleanup, got %d", count)
		}
	})

	t.Run("does nothing when no expired posts exist", func(t *testing.T) {
		database := setupPostTestDatabase(t)
		userRepo := NewUserRepo(database)
		postRepo := NewPostRepo(database)

		user, err := userRepo.CreateUser("alice", "password")
		if err != nil {
			t.Fatalf("seed user error: %v", err)
		}

		_, err = postRepo.CreatePost("Recent Post", "Recent Body", user.ID)
		if err != nil {
			t.Fatalf("CreatePost error: %v", err)
		}

		err = postRepo.PruneExpiredPosts(30, 100)
		if err != nil {
			t.Fatalf("CleanupExpiredPosts error: %v", err)
		}

		count, err := postRepo.CountPostsByUserID(user.ID)
		if err != nil {
			t.Fatalf("CountPostsByUserID error: %v", err)
		}

		if count != 1 {
			t.Fatalf("expected 1 post to remain, got %d", count)
		}
	})
}
