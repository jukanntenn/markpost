package post

import (
	"context"
	"testing"

	"markpost/internal/domain/post"
)

type mockPostRepository struct {
	posts    map[string]*post.Post
	idPosts  map[int]*post.Post
	nextID   int
}

func newMockPostRepository() *mockPostRepository {
	return &mockPostRepository{
		posts:   make(map[string]*post.Post),
		idPosts: make(map[int]*post.Post),
		nextID:  1,
	}
}

func (m *mockPostRepository) Create(ctx context.Context, title, body string, userID int) (*post.Post, error) {
	p := &post.Post{
		ID:     m.nextID,
		QID:    "test-qid-" + string(rune(m.nextID+'0')),
		Title:  title,
		Body:   body,
		UserID: userID,
	}
	m.posts[p.QID] = p
	m.idPosts[p.ID] = p
	m.nextID++
	return p, nil
}

func (m *mockPostRepository) GetByQID(ctx context.Context, qid string) (*post.Post, error) {
	p, ok := m.posts[qid]
	if !ok {
		return nil, post.ErrNotFound
	}
	return p, nil
}

func (m *mockPostRepository) GetByUserID(ctx context.Context, userID, offset, limit int) ([]post.Post, error) {
	var result []post.Post
	for _, p := range m.posts {
		if p.UserID == userID {
			result = append(result, *p)
		}
	}
	if offset >= len(result) {
		return []post.Post{}, nil
	}
	end := offset + limit
	if end > len(result) {
		end = len(result)
	}
	return result[offset:end], nil
}

func (m *mockPostRepository) CountByUserID(ctx context.Context, userID int) (int64, error) {
	var count int64
	for _, p := range m.posts {
		if p.UserID == userID {
			count++
		}
	}
	return count, nil
}

func (m *mockPostRepository) CountAll(ctx context.Context, search string) (int64, error) {
	return int64(len(m.posts)), nil
}

func (m *mockPostRepository) CreateBatch(ctx context.Context, posts []post.Post) (int, error) {
	return len(posts), nil
}

func (m *mockPostRepository) GetByID(ctx context.Context, id int) (*post.Post, error) {
	p, ok := m.idPosts[id]
	if !ok {
		return nil, post.ErrNotFound
	}
	return p, nil
}

func (m *mockPostRepository) ListAll(ctx context.Context, search string, offset, limit int) ([]post.Post, error) {
	return nil, nil
}

func (m *mockPostRepository) UpdateByID(ctx context.Context, id int, title, body string) (*post.Post, error) {
	return nil, nil
}

func (m *mockPostRepository) DeleteByID(ctx context.Context, id int) (int64, error) {
	return 0, nil
}

func (m *mockPostRepository) PruneExpired(ctx context.Context, retentionDays, batchSize int) error {
	return nil
}

func (m *mockPostRepository) CountExpired(ctx context.Context, retentionDays int) (int64, error) {
	return 0, nil
}

func TestPostService_CreatePost(t *testing.T) {
	mockRepo := newMockPostRepository()
	svc := NewPostService(mockRepo, nil)
	ctx := context.Background()

	t.Run("creates post successfully", func(t *testing.T) {
		qid, err := svc.CreatePost(ctx, "Test Title", "Test Body", 1)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if qid == "" {
			t.Error("expected qid, got empty")
		}
	})

	t.Run("creates post with valid data", func(t *testing.T) {
		qid, _ := svc.CreatePost(ctx, "Another Title", "Another Body", 1)

		p, err := mockRepo.GetByQID(ctx, qid)
		if err != nil {
			t.Fatalf("expected to find post, got: %v", err)
		}
		if p.Title != "Another Title" {
			t.Errorf("expected title 'Another Title', got: %s", p.Title)
		}
		if p.Body != "Another Body" {
			t.Errorf("expected body 'Another Body', got: %s", p.Body)
		}
	})
}

func TestPostService_GetPostMarkdown(t *testing.T) {
	mockRepo := newMockPostRepository()
	svc := NewPostService(mockRepo, nil)
	ctx := context.Background()

	created, _ := mockRepo.Create(ctx, "Test Title", "Test Body", 1)

	t.Run("returns markdown for valid post", func(t *testing.T) {
		title, body, err := svc.GetPostMarkdown(ctx, created.QID)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if title != "Test Title" {
			t.Errorf("expected title 'Test Title', got: %s", title)
		}
		if body != "Test Body" {
			t.Errorf("expected body 'Test Body', got: %s", body)
		}
	})

	t.Run("returns error for non-existent post", func(t *testing.T) {
		_, _, err := svc.GetPostMarkdown(ctx, "nonexistent")
		if err == nil {
			t.Fatal("expected error for non-existent post")
		}
	})
}

func TestPostService_RenderPostHTML(t *testing.T) {
	mockRepo := newMockPostRepository()
	svc := NewPostService(mockRepo, nil)
	ctx := context.Background()

	created, _ := mockRepo.Create(ctx, "Test Title", "# Heading\n\nParagraph", 1)

	t.Run("renders HTML for valid post", func(t *testing.T) {
		title, html, err := svc.RenderPostHTML(ctx, created.QID)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if title != "Test Title" {
			t.Errorf("expected title 'Test Title', got: %s", title)
		}
		if html == "" {
			t.Error("expected HTML content, got empty")
		}
	})

	t.Run("returns error for non-existent post", func(t *testing.T) {
		_, _, err := svc.RenderPostHTML(ctx, "nonexistent")
		if err == nil {
			t.Fatal("expected error for non-existent post")
		}
	})
}

func TestPostService_GetUserPosts(t *testing.T) {
	mockRepo := newMockPostRepository()
	svc := NewPostService(mockRepo, nil)
	ctx := context.Background()

	mockRepo.Create(ctx, "Title 1", "Body 1", 1)
	mockRepo.Create(ctx, "Title 2", "Body 2", 1)
	mockRepo.Create(ctx, "Title 3", "Body 3", 2)

	t.Run("returns posts for user", func(t *testing.T) {
		posts, total, err := svc.GetUserPosts(ctx, 1, 1, 10)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if len(posts) != 2 {
			t.Errorf("expected 2 posts, got: %d", len(posts))
		}
		if total != 2 {
			t.Errorf("expected total 2, got: %d", total)
		}
	})

	t.Run("returns empty for user with no posts", func(t *testing.T) {
		posts, total, err := svc.GetUserPosts(ctx, 999, 1, 10)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if len(posts) != 0 {
			t.Errorf("expected 0 posts, got: %d", len(posts))
		}
		if total != 0 {
			t.Errorf("expected total 0, got: %d", total)
		}
	})

	t.Run("handles pagination correctly", func(t *testing.T) {
		mockRepo.Create(ctx, "Title 4", "Body 4", 3)
		mockRepo.Create(ctx, "Title 5", "Body 5", 3)
		mockRepo.Create(ctx, "Title 6", "Body 6", 3)

		posts, total, err := svc.GetUserPosts(ctx, 3, 1, 2)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if len(posts) != 2 {
			t.Errorf("expected 2 posts on first page, got: %d", len(posts))
		}
		if total != 3 {
			t.Errorf("expected total 3, got: %d", total)
		}

		posts2, _, err := svc.GetUserPosts(ctx, 3, 2, 2)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if len(posts2) != 1 {
			t.Errorf("expected 1 post on second page, got: %d", len(posts2))
		}
	})
}
