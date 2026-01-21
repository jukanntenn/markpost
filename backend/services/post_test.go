package services

import (
	"fmt"
	"strings"
	"testing"

	"markpost/models"
	"markpost/repositories"
)

type stubPostRepo struct {
	createPostResult *models.Post
	createPostErr    error

	getPostResult *models.Post
	getPostErr    error

	countResult int64
	countErr    error

	listResult []models.Post
	listErr    error
}

func (s *stubPostRepo) CreatePost(title, body string, userID int) (*models.Post, error) {
	if s.createPostErr != nil {
		return nil, s.createPostErr
	}
	if s.createPostResult != nil {
		return s.createPostResult, nil
	}
	return &models.Post{
		QID:    "qid-stub",
		Title:  title,
		Body:   body,
		UserID: userID,
	}, nil
}

func (s *stubPostRepo) GetPostByQID(qid string) (*models.Post, error) {
	if s.getPostErr != nil {
		return nil, s.getPostErr
	}
	if s.getPostResult != nil {
		return s.getPostResult, nil
	}
	return &models.Post{
		QID:  qid,
		Body: "body",
	}, nil
}

func (s *stubPostRepo) CountPostsByUserID(userID int) (int64, error) {
	if s.countErr != nil {
		return 0, s.countErr
	}
	return s.countResult, nil
}

func (s *stubPostRepo) GetPostsByUserID(userID int, offset int, limit int) ([]models.Post, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	if s.listResult != nil {
		return s.listResult, nil
	}
	return []models.Post{}, nil
}

func (s *stubPostRepo) PruneExpiredPosts(retentionDays int, batchSize int) error {
	return fmt.Errorf("not implemented")
}

func (s *stubPostRepo) CountExpiredPosts(retentionDays int) (int64, error) {
	return 0, fmt.Errorf("not implemented")
}

func (s *stubPostRepo) CreatePosts(posts []models.Post) (int, error) {
	return 0, fmt.Errorf("not implemented")
}

func setupPostTestDatabase(t *testing.T) *models.Database {
	t.Helper()

	database, err := models.NewTestDatabase()
	if err != nil {
		t.Fatalf("NewTestDatabase error: %v", err)
	}

	return database
}

func TestPostService_CreatePost(t *testing.T) {
	t.Run("success with real repository", func(t *testing.T) {
		db := setupPostTestDatabase(t)

		userRepo := repositories.NewUserRepo(db)
		user, err := userRepo.CreateUser("alice", "password")
		if err != nil {
			t.Fatalf("CreateUser error: %v", err)
		}

		postRepo := repositories.NewPostRepo(db)
		svc := NewPostService(postRepo)

		qid, err := svc.CreatePost("Test Title", "Test Body", user.ID)
		if err != nil {
			t.Fatalf("CreatePost error: %v", err)
		}
		if qid == "" {
			t.Fatalf("expected non-empty qid")
		}

		post, err := postRepo.GetPostByQID(qid)
		if err != nil {
			t.Fatalf("GetPostByQID error: %v", err)
		}
		if post.Title != "Test Title" || post.Body != "Test Body" || post.UserID != user.ID {
			t.Fatalf("unexpected post: %+v", post)
		}
	})

	t.Run("nonexistent user -> ErrInternal", func(t *testing.T) {
		db := setupPostTestDatabase(t)

		postRepo := repositories.NewPostRepo(db)
		svc := NewPostService(postRepo)

		_, err := svc.CreatePost("Title", "Body", 999999)
		if err == nil {
			t.Fatalf("expected error")
		}
		se, ok := err.(*ServiceError)
		if !ok {
			t.Fatalf("expected ServiceError, got: %#v", err)
		}
		if se.Code != ErrInternal {
			t.Fatalf("expected ErrInternal, got: %v", se.Code)
		}
	})

	t.Run("repo error -> ErrInternal", func(t *testing.T) {
		stub := &stubPostRepo{
			createPostErr: fmt.Errorf("db error"),
		}
		svc := NewPostService(stub)

		_, err := svc.CreatePost("Title", "Body", 1)
		if err == nil {
			t.Fatalf("expected error")
		}
		se, ok := err.(*ServiceError)
		if !ok {
			t.Fatalf("expected ServiceError, got: %#v", err)
		}
		if se.Code != ErrInternal {
			t.Fatalf("expected ErrInternal, got: %v", se.Code)
		}
	})
}

func TestPostService_RenderPostHTML(t *testing.T) {
	t.Run("success with real repository", func(t *testing.T) {
		db := setupPostTestDatabase(t)

		userRepo := repositories.NewUserRepo(db)
		user, err := userRepo.CreateUser("alice", "password")
		if err != nil {
			t.Fatalf("CreateUser error: %v", err)
		}

		postRepo := repositories.NewPostRepo(db)
		created, err := postRepo.CreatePost("Markdown Title", "Hello **World**", user.ID)
		if err != nil {
			t.Fatalf("CreatePost error: %v", err)
		}

		svc := NewPostService(postRepo)
		title, html, err := svc.RenderPostHTML(created.QID)
		if err != nil {
			t.Fatalf("RenderPostHTML error: %v", err)
		}
		if title != "Markdown Title" {
			t.Fatalf("unexpected title: %s", title)
		}
		if html == "" {
			t.Fatalf("expected non-empty html")
		}
		if !strings.Contains(html, "<strong>World</strong>") {
			t.Fatalf("expected rendered HTML to contain <strong>World</strong>, got: %s", html)
		}
	})

	t.Run("not found -> ErrNotFound", func(t *testing.T) {
		stub := &stubPostRepo{
			getPostErr: models.ErrNotFound,
		}
		svc := NewPostService(stub)

		_, _, err := svc.RenderPostHTML("not-exist")
		if err == nil {
			t.Fatalf("expected error")
		}
		se, ok := err.(*ServiceError)
		if !ok {
			t.Fatalf("expected ServiceError, got: %#v", err)
		}
		if se.Code != ErrNotFound {
			t.Fatalf("expected ErrNotFound, got: %v", se.Code)
		}
	})

	t.Run("other error -> ErrInternal", func(t *testing.T) {
		stub := &stubPostRepo{
			getPostErr: fmt.Errorf("db error"),
		}
		svc := NewPostService(stub)

		_, _, err := svc.RenderPostHTML("qid")
		if err == nil {
			t.Fatalf("expected error")
		}
		se, ok := err.(*ServiceError)
		if !ok {
			t.Fatalf("expected ServiceError, got: %#v", err)
		}
		if se.Code != ErrInternal {
			t.Fatalf("expected ErrInternal, got: %v", se.Code)
		}
	})
}

func TestPostService_GetUserPosts(t *testing.T) {
	t.Run("default page and limit with real repository", func(t *testing.T) {
		db := setupPostTestDatabase(t)

		userRepo := repositories.NewUserRepo(db)
		user, err := userRepo.CreateUser("alice", "password")
		if err != nil {
			t.Fatalf("CreateUser error: %v", err)
		}

		postRepo := repositories.NewPostRepo(db)
		for i := 1; i <= 3; i++ {
			_, err := postRepo.CreatePost(fmt.Sprintf("Post %d", i), "Body", user.ID)
			if err != nil {
				t.Fatalf("CreatePost error: %v", err)
			}
		}

		svc := NewPostService(postRepo)
		posts, total, err := svc.GetUserPosts(user.ID, 0, 0)
		if err != nil {
			t.Fatalf("GetUserPosts error: %v", err)
		}
		if total != 3 {
			t.Fatalf("expected total 3, got %d", total)
		}
		if len(posts) != 3 {
			t.Fatalf("expected 3 posts, got %d", len(posts))
		}
		for _, p := range posts {
			if p.UserID != user.ID {
				t.Fatalf("unexpected user id: %d", p.UserID)
			}
		}
	})

	t.Run("pagination offset works", func(t *testing.T) {
		db := setupPostTestDatabase(t)

		userRepo := repositories.NewUserRepo(db)
		user, err := userRepo.CreateUser("alice", "password")
		if err != nil {
			t.Fatalf("CreateUser error: %v", err)
		}

		postRepo := repositories.NewPostRepo(db)
		for i := 1; i <= 3; i++ {
			_, err := postRepo.CreatePost(fmt.Sprintf("Post %d", i), "Body", user.ID)
			if err != nil {
				t.Fatalf("CreatePost error: %v", err)
			}
		}

		svc := NewPostService(postRepo)
		posts, total, err := svc.GetUserPosts(user.ID, 2, 1)
		if err != nil {
			t.Fatalf("GetUserPosts error: %v", err)
		}
		if total != 3 {
			t.Fatalf("expected total 3, got %d", total)
		}
		if len(posts) != 1 {
			t.Fatalf("expected 1 post, got %d", len(posts))
		}
		if posts[0].Title != "Post 2" {
			t.Fatalf("expected Post 2, got %s", posts[0].Title)
		}
	})

	t.Run("count error -> ErrInternal", func(t *testing.T) {
		stub := &stubPostRepo{
			countErr: fmt.Errorf("db error"),
		}
		svc := NewPostService(stub)

		_, _, err := svc.GetUserPosts(1, 1, 10)
		if err == nil {
			t.Fatalf("expected error")
		}
		se, ok := err.(*ServiceError)
		if !ok {
			t.Fatalf("expected ServiceError, got: %#v", err)
		}
		if se.Code != ErrInternal {
			t.Fatalf("expected ErrInternal, got: %v", se.Code)
		}
	})

	t.Run("list error -> ErrInternal", func(t *testing.T) {
		stub := &stubPostRepo{
			countResult: 5,
			listErr:     fmt.Errorf("db error"),
		}
		svc := NewPostService(stub)

		_, _, err := svc.GetUserPosts(1, 1, 10)
		if err == nil {
			t.Fatalf("expected error")
		}
		se, ok := err.(*ServiceError)
		if !ok {
			t.Fatalf("expected ServiceError, got: %#v", err)
		}
		if se.Code != ErrInternal {
			t.Fatalf("expected ErrInternal, got: %v", se.Code)
		}
	})
}
