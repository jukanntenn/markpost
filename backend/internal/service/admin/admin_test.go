package admin

import (
	"context"
	"errors"
	"testing"

	"markpost/internal/domain/delivery"
	"markpost/internal/domain/post"
	"markpost/internal/domain/user"
	"markpost/internal/service"
)

type mockUserRepo struct {
	users    []user.User
	err      error
	count    int64
	countErr error
}

func (m *mockUserRepo) GetAll(_ context.Context, _, _ int) ([]user.User, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.users, nil
}

func (m *mockUserRepo) Count(_ context.Context) (int64, error) {
	if m.countErr != nil {
		return 0, m.countErr
	}
	return m.count, nil
}

type mockPostRepo struct {
	posts    []post.Post
	err      error
	count    int64
	countErr error
	search   string
}

func (m *mockPostRepo) ListAll(_ context.Context, search string, _, _ int) ([]post.Post, error) {
	if m.err != nil {
		return nil, m.err
	}
	m.search = search
	return m.posts, nil
}

func (m *mockPostRepo) CountAll(_ context.Context, search string) (int64, error) {
	if m.countErr != nil {
		return 0, m.countErr
	}
	return m.count, nil
}

type mockDeliveryRepo struct {
	channels []delivery.Channel
	err      error
	count    int64
	countErr error
}

func (m *mockDeliveryRepo) ListAll(_ context.Context, _, _ int) ([]delivery.Channel, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.channels, nil
}

func (m *mockDeliveryRepo) CountAll(_ context.Context) (int64, error) {
	if m.countErr != nil {
		return 0, m.countErr
	}
	return m.count, nil
}

func TestListAllUsers(t *testing.T) {
	ctx := context.Background()

	t.Run("returns users and total on success", func(t *testing.T) {
		users := []user.User{{ID: 1, Username: "alice"}, {ID: 2, Username: "bob"}}
		repo := &mockUserRepo{users: users, count: 2}
		svc := NewService(repo, nil, nil)

		result, total, err := svc.ListAllUsers(ctx, 0, 10)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if len(result) != 2 {
			t.Errorf("expected 2 users, got %d", len(result))
		}
		if total != 2 {
			t.Errorf("expected total 2, got %d", total)
		}
	})

	t.Run("wraps list error as ErrInternal", func(t *testing.T) {
		repo := &mockUserRepo{err: errors.New("db fail")}
		svc := NewService(repo, nil, nil)

		_, _, err := svc.ListAllUsers(ctx, 0, 10)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		se, ok := service.AsServiceError(err)
		if !ok {
			t.Fatal("expected service error")
		}
		if se.Code != service.ErrInternal {
			t.Errorf("expected code %s, got %s", service.ErrInternal, se.Code)
		}
	})

	t.Run("wraps count error as ErrInternal", func(t *testing.T) {
		repo := &mockUserRepo{
			users:    []user.User{{ID: 1}},
			countErr: errors.New("count fail"),
		}
		svc := NewService(repo, nil, nil)

		_, _, err := svc.ListAllUsers(ctx, 0, 10)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		se, ok := service.AsServiceError(err)
		if !ok {
			t.Fatal("expected service error")
		}
		if se.Code != service.ErrInternal {
			t.Errorf("expected code %s, got %s", service.ErrInternal, se.Code)
		}
	})
}

func TestListAllPosts(t *testing.T) {
	ctx := context.Background()

	t.Run("returns posts and total on success", func(t *testing.T) {
		posts := []post.Post{{ID: 1, Title: "First"}, {ID: 2, Title: "Second"}}
		repo := &mockPostRepo{posts: posts, count: 2}
		svc := NewService(nil, repo, nil)

		result, total, err := svc.ListAllPosts(ctx, "test", 0, 10)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if len(result) != 2 {
			t.Errorf("expected 2 posts, got %d", len(result))
		}
		if total != 2 {
			t.Errorf("expected total 2, got %d", total)
		}
		if repo.search != "test" {
			t.Errorf("expected search 'test', got %q", repo.search)
		}
	})

	t.Run("wraps list error as ErrInternal", func(t *testing.T) {
		repo := &mockPostRepo{err: errors.New("db fail")}
		svc := NewService(nil, repo, nil)

		_, _, err := svc.ListAllPosts(ctx, "", 0, 10)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		se, ok := service.AsServiceError(err)
		if !ok {
			t.Fatal("expected service error")
		}
		if se.Code != service.ErrInternal {
			t.Errorf("expected code %s, got %s", service.ErrInternal, se.Code)
		}
	})

	t.Run("wraps count error as ErrInternal", func(t *testing.T) {
		repo := &mockPostRepo{
			posts:    []post.Post{{ID: 1}},
			countErr: errors.New("count fail"),
		}
		svc := NewService(nil, repo, nil)

		_, _, err := svc.ListAllPosts(ctx, "", 0, 10)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		se, ok := service.AsServiceError(err)
		if !ok {
			t.Fatal("expected service error")
		}
		if se.Code != service.ErrInternal {
			t.Errorf("expected code %s, got %s", service.ErrInternal, se.Code)
		}
	})
}

func TestListAllDeliveryChannels(t *testing.T) {
	ctx := context.Background()

	t.Run("returns channels and total on success", func(t *testing.T) {
		channels := []delivery.Channel{{ID: 1, Name: "ch1"}, {ID: 2, Name: "ch2"}}
		repo := &mockDeliveryRepo{channels: channels, count: 2}
		svc := NewService(nil, nil, repo)

		result, total, err := svc.ListAllDeliveryChannels(ctx, 0, 10)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if len(result) != 2 {
			t.Errorf("expected 2 channels, got %d", len(result))
		}
		if total != 2 {
			t.Errorf("expected total 2, got %d", total)
		}
	})

	t.Run("wraps list error as ErrInternal", func(t *testing.T) {
		repo := &mockDeliveryRepo{err: errors.New("db fail")}
		svc := NewService(nil, nil, repo)

		_, _, err := svc.ListAllDeliveryChannels(ctx, 0, 10)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		se, ok := service.AsServiceError(err)
		if !ok {
			t.Fatal("expected service error")
		}
		if se.Code != service.ErrInternal {
			t.Errorf("expected code %s, got %s", service.ErrInternal, se.Code)
		}
	})

	t.Run("wraps count error as ErrInternal", func(t *testing.T) {
		repo := &mockDeliveryRepo{
			channels: []delivery.Channel{{ID: 1}},
			countErr: errors.New("count fail"),
		}
		svc := NewService(nil, nil, repo)

		_, _, err := svc.ListAllDeliveryChannels(ctx, 0, 10)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		se, ok := service.AsServiceError(err)
		if !ok {
			t.Fatal("expected service error")
		}
		if se.Code != service.ErrInternal {
			t.Errorf("expected code %s, got %s", service.ErrInternal, se.Code)
		}
	})
}
