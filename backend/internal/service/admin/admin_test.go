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

type mockUserLister struct {
	users    []user.User
	err      error
	count    int64
	countErr error
}

func (m *mockUserLister) GetAll(_ context.Context, _, _ int) ([]user.User, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.users, nil
}

func (m *mockUserLister) Count(_ context.Context) (int64, error) {
	if m.countErr != nil {
		return 0, m.countErr
	}
	return m.count, nil
}

type mockPostLister struct {
	posts []post.Post
	total int64
	err   error
}

func (m *mockPostLister) GetAllPosts(_ context.Context, _ string, _, _ int) ([]post.Post, int64, error) {
	return m.posts, m.total, m.err
}

type mockChannelLister struct {
	channels []delivery.Channel
	total    int64
	err      error
}

func (m *mockChannelLister) ListAll(_ context.Context, _, _ int) ([]delivery.Channel, int64, error) {
	return m.channels, m.total, m.err
}

func TestListAllUsers(t *testing.T) {
	ctx := context.Background()

	t.Run("returns users and total on success", func(t *testing.T) {
		users := []user.User{{ID: 1, Username: "alice"}, {ID: 2, Username: "bob"}}
		lister := &mockUserLister{users: users, count: 2}
		svc := NewService(lister, &mockPostLister{}, &mockChannelLister{})

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
		lister := &mockUserLister{err: errors.New("db fail")}
		svc := NewService(lister, &mockPostLister{}, &mockChannelLister{})

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
		lister := &mockUserLister{
			users:    []user.User{{ID: 1}},
			countErr: errors.New("count fail"),
		}
		svc := NewService(lister, &mockPostLister{}, &mockChannelLister{})

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
		lister := &mockPostLister{posts: posts, total: 2}
		svc := NewService(nil, lister, &mockChannelLister{})

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
	})

	t.Run("propagates error from lister", func(t *testing.T) {
		lister := &mockPostLister{err: service.NewServiceErrorWrap(service.ErrInternal, "list failed", errors.New("db fail"))}
		svc := NewService(nil, lister, &mockChannelLister{})

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
		lister := &mockChannelLister{channels: channels, total: 2}
		svc := NewService(nil, &mockPostLister{}, lister)

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

	t.Run("propagates error from lister", func(t *testing.T) {
		lister := &mockChannelLister{err: service.NewServiceErrorWrap(service.ErrInternal, "list failed", errors.New("db fail"))}
		svc := NewService(nil, &mockPostLister{}, lister)

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
