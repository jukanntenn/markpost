package admin

import (
	"context"
	"fmt"
	"testing"

	"markpost/internal/domain/delivery"
	"markpost/internal/domain/post"
	"markpost/internal/domain/user"
	"markpost/internal/infra"
)

func createAdminTestUser(t *testing.T, userRepo user.Repository, idx int) int {
	t.Helper()
	u, err := userRepo.Create(context.Background(), fmt.Sprintf("user%d@example.com", idx), fmt.Sprintf("user%d", idx), "password")
	if err != nil {
		t.Fatalf("createAdminTestUser(%d): %v", idx, err)
	}
	return u.ID
}

func setupAdminService(t *testing.T) (*Service, user.Repository, post.Repository, delivery.Repository) {
	t.Helper()
	db := infra.SetupTestDB(t)
	userRepo := infra.NewUserRepository(db, 16)
	postRepo := infra.NewPostRepository(db)
	channelRepo := infra.NewDeliveryChannelRepository(db)
	attemptRepo := infra.NewAttemptRepository(db)

	svc := NewService(
		userRepo.(*infra.UserRepository),
		&postListerAdapter{repo: postRepo},
		&channelListerAdapter{repo: channelRepo},
		attemptRepo,
	)
	return svc, userRepo, postRepo, channelRepo
}

type postListerAdapter struct {
	repo post.Repository
}

func (a *postListerAdapter) GetAllPosts(ctx context.Context, search string, offset, limit int) ([]post.Post, int64, error) {
	items, err := a.repo.ListAll(ctx, search, offset, limit)
	if err != nil {
		return nil, 0, err
	}
	count, err := a.repo.CountAll(ctx, search)
	if err != nil {
		return nil, 0, err
	}
	return items, count, nil
}

type channelListerAdapter struct {
	repo delivery.Repository
}

func (a *channelListerAdapter) ListAll(ctx context.Context, offset, limit int) ([]delivery.Channel, int64, error) {
	items, err := a.repo.ListAll(ctx, offset, limit)
	if err != nil {
		return nil, 0, err
	}
	count, err := a.repo.CountAll(ctx)
	if err != nil {
		return nil, 0, err
	}
	return items, count, nil
}

func TestListAllUsers(t *testing.T) {
	svc, userRepo, _, _ := setupAdminService(t)
	ctx := context.Background()

	_, _ = userRepo.Create(ctx, "a@example.com", "alice", "pass")
	_, _ = userRepo.Create(ctx, "b@example.com", "bob", "pass")

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
}

func TestListAllPosts(t *testing.T) {
	svc, userRepo, postRepo, _ := setupAdminService(t)
	uid1 := createAdminTestUser(t, userRepo, 0)
	uid2 := createAdminTestUser(t, userRepo, 1)
	ctx := context.Background()

	_, _ = postRepo.Create(ctx, "First", "Body", uid1)
	_, _ = postRepo.Create(ctx, "Second", "Body", uid2)

	result, total, err := svc.ListAllPosts(ctx, "", 0, 10)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 posts, got %d", len(result))
	}
	if total != 2 {
		t.Errorf("expected total 2, got %d", total)
	}
}

func TestListAllDeliveryChannels(t *testing.T) {
	svc, userRepo, _, channelRepo := setupAdminService(t)
	uid1 := createAdminTestUser(t, userRepo, 0)
	uid2 := createAdminTestUser(t, userRepo, 1)
	ctx := context.Background()

	_ = channelRepo.Create(ctx, &delivery.Channel{UserID: uid1, Kind: delivery.ChannelKindFeishu, Name: "Ch1", Configuration: delivery.ChannelConfiguration{"webhook_url": "https://a.com", "card_link_url": ""}})
	_ = channelRepo.Create(ctx, &delivery.Channel{UserID: uid2, Kind: delivery.ChannelKindFeishu, Name: "Ch2", Configuration: delivery.ChannelConfiguration{"webhook_url": "https://b.com", "card_link_url": ""}})

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
}
