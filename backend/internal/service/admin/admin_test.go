package admin

import (
	"context"
	"fmt"
	"testing"

	"markpost/internal/domain/audit"
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
	sessionLister := &mockSessionLister{}
	auditRecorder := &mockAuditRecorder{}

	svc := NewService(
		userRepo.(*infra.UserRepository),
		&postListerAdapter{repo: postRepo},
		&channelListerAdapter{repo: channelRepo},
		attemptRepo,
		sessionLister,
		auditRecorder,
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

type mockAuditRecorder struct {
	logs []audit.Log
}

type mockSessionLister struct{}

func (m *mockSessionLister) ListByUserID(ctx context.Context, userID int) ([]user.RefreshToken, error) {
	return []user.RefreshToken{}, nil
}

func (m *mockSessionLister) RevokeAllByUserID(ctx context.Context, userID int) error {
	return nil
}

func (m *mockAuditRecorder) Record(ctx context.Context, e audit.Entry) error {
	m.logs = append(m.logs, audit.Log{
		ActorID:    e.ActorID,
		Action:     e.Action,
		TargetType: e.TargetType,
		TargetID:   e.TargetID,
	})
	return nil
}

func (m *mockAuditRecorder) List(ctx context.Context, offset, limit int) ([]audit.Log, int64, error) {
	total := int64(len(m.logs))
	if offset >= len(m.logs) {
		return nil, total, nil
	}
	end := offset + limit
	if end > len(m.logs) {
		end = len(m.logs)
	}
	return m.logs[offset:end], total, nil
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

func setupAdminServiceWithMutators(t *testing.T) (*Service, user.Repository, delivery.Repository) {
	t.Helper()
	db := infra.SetupTestDB(t)
	userRepo := infra.NewUserRepository(db, 16)
	postRepo := infra.NewPostRepository(db)
	channelRepo := infra.NewDeliveryChannelRepository(db)
	attemptRepo := infra.NewAttemptRepository(db)
	sessionLister := &mockSessionLister{}
	auditRecorder := &mockAuditRecorder{}

	svc := NewService(
		userRepo.(*infra.UserRepository),
		&postListerAdapter{repo: postRepo},
		&channelListerAdapter{repo: channelRepo},
		attemptRepo,
		sessionLister,
		auditRecorder,
	)
	svc.SetUserMutator(userRepo.(*infra.UserRepository))
	svc.SetChannelMutator(channelRepo)
	return svc, userRepo, channelRepo
}

func TestCreateUser(t *testing.T) {
	svc, _, _ := setupAdminServiceWithMutators(t)
	ctx := context.Background()

	t.Run("creates user successfully", func(t *testing.T) {
		u, err := svc.CreateUser(ctx, "new@example.com", "newuser", "password123")
		if err != nil {
			t.Fatalf("CreateUser: %v", err)
		}
		if u.Email != "new@example.com" {
			t.Errorf("email = %q, want %q", u.Email, "new@example.com")
		}
		if u.Username != "newuser" {
			t.Errorf("username = %q, want %q", u.Username, "newuser")
		}
	})

	t.Run("rejects duplicate email", func(t *testing.T) {
		_, _ = svc.CreateUser(ctx, "dup@example.com", "user1", "pass123")
		_, err := svc.CreateUser(ctx, "dup@example.com", "user2", "pass123")
		if err == nil {
			t.Fatal("expected error for duplicate email")
		}
	})
}

func TestSetUserRole(t *testing.T) {
	svc, userRepo, _ := setupAdminServiceWithMutators(t)
	ctx := context.Background()
	u, _ := userRepo.Create(ctx, "role@example.com", "roleuser", "pass")

	t.Run("sets role to admin", func(t *testing.T) {
		if err := svc.SetUserRole(ctx, u.ID, user.RoleAdmin); err != nil {
			t.Fatalf("SetUserRole: %v", err)
		}
		got, _ := svc.GetUserByID(ctx, u.ID)
		if got.Role != user.RoleAdmin {
			t.Errorf("role = %q, want %q", got.Role, user.RoleAdmin)
		}
	})

	t.Run("sets role to user", func(t *testing.T) {
		if err := svc.SetUserRole(ctx, u.ID, user.RoleUser); err != nil {
			t.Fatalf("SetUserRole: %v", err)
		}
		got, _ := svc.GetUserByID(ctx, u.ID)
		if got.Role != user.RoleUser {
			t.Errorf("role = %q, want %q", got.Role, user.RoleUser)
		}
	})
}

func TestResetUserPassword(t *testing.T) {
	svc, userRepo, _ := setupAdminServiceWithMutators(t)
	ctx := context.Background()
	u, _ := userRepo.Create(ctx, "pw@example.com", "pwuser", "oldpass")

	t.Run("resets password successfully", func(t *testing.T) {
		if err := svc.ResetUserPassword(ctx, u.ID, "newpass123"); err != nil {
			t.Fatalf("ResetUserPassword: %v", err)
		}
	})

	t.Run("returns not found for nonexistent user", func(t *testing.T) {
		err := svc.ResetUserPassword(ctx, 99999, "newpass")
		if err == nil {
			t.Fatal("expected error for nonexistent user")
		}
	})
}

func TestSetUserActive(t *testing.T) {
	svc, userRepo, _ := setupAdminServiceWithMutators(t)
	ctx := context.Background()
	u, _ := userRepo.Create(ctx, "active@example.com", "activeuser", "pass")

	t.Run("deactivates user", func(t *testing.T) {
		if err := svc.SetUserActive(ctx, u.ID, false); err != nil {
			t.Fatalf("SetUserActive: %v", err)
		}
		got, _ := svc.GetUserByID(ctx, u.ID)
		if got.IsActive {
			t.Error("expected user to be inactive")
		}
	})

	t.Run("reactivates user", func(t *testing.T) {
		if err := svc.SetUserActive(ctx, u.ID, true); err != nil {
			t.Fatalf("SetUserActive: %v", err)
		}
		got, _ := svc.GetUserByID(ctx, u.ID)
		if !got.IsActive {
			t.Error("expected user to be active")
		}
	})
}

func TestDeleteUser(t *testing.T) {
	svc, userRepo, _ := setupAdminServiceWithMutators(t)
	ctx := context.Background()
	u, _ := userRepo.Create(ctx, "delete@example.com", "deleteuser", "pass")

	t.Run("deletes existing user", func(t *testing.T) {
		deleted, err := svc.DeleteUser(ctx, u.ID)
		if err != nil {
			t.Fatalf("DeleteUser: %v", err)
		}
		if deleted != 1 {
			t.Errorf("expected 1 deleted, got %d", deleted)
		}
	})

	t.Run("returns 0 for nonexistent user", func(t *testing.T) {
		deleted, err := svc.DeleteUser(ctx, 99999)
		if err != nil {
			t.Fatalf("DeleteUser: %v", err)
		}
		if deleted != 0 {
			t.Errorf("expected 0 deleted, got %d", deleted)
		}
	})
}

func TestGetUserByID(t *testing.T) {
	svc, userRepo, _ := setupAdminServiceWithMutators(t)
	ctx := context.Background()
	u, _ := userRepo.Create(ctx, "get@example.com", "getuser", "pass")

	t.Run("returns existing user", func(t *testing.T) {
		got, err := svc.GetUserByID(ctx, u.ID)
		if err != nil {
			t.Fatalf("GetUserByID: %v", err)
		}
		if got.Email != "get@example.com" {
			t.Errorf("email = %q, want %q", got.Email, "get@example.com")
		}
	})

	t.Run("returns error for nonexistent user", func(t *testing.T) {
		_, err := svc.GetUserByID(ctx, 99999)
		if err == nil {
			t.Fatal("expected error for nonexistent user")
		}
	})
}

func TestCreateChannel(t *testing.T) {
	svc, userRepo, _ := setupAdminServiceWithMutators(t)
	ctx := context.Background()
	u, _ := userRepo.Create(ctx, "ch@example.com", "chuser", "pass")

	t.Run("creates channel successfully", func(t *testing.T) {
		ch := &delivery.Channel{
			UserID:        u.ID,
			Kind:          delivery.ChannelKindFeishu,
			Name:          "Test Channel",
			Configuration: delivery.ChannelConfiguration{"webhook_url": "https://example.com", "card_link_url": ""},
			Enabled:       true,
		}
		if err := svc.CreateChannel(ctx, ch); err != nil {
			t.Fatalf("CreateChannel: %v", err)
		}
		if ch.ID == 0 {
			t.Error("expected channel ID to be set")
		}
	})
}

func TestGetChannelByID(t *testing.T) {
	svc, userRepo, channelRepo := setupAdminServiceWithMutators(t)
	ctx := context.Background()
	u, _ := userRepo.Create(ctx, "chget@example.com", "chgetuser", "pass")
	_ = channelRepo.Create(ctx, &delivery.Channel{UserID: u.ID, Kind: delivery.ChannelKindFeishu, Name: "Ch", Configuration: delivery.ChannelConfiguration{"webhook_url": "https://a.com", "card_link_url": ""}})

	channels, _, _ := svc.ListAllDeliveryChannels(ctx, 0, 10)
	if len(channels) == 0 {
		t.Fatal("expected at least 1 channel")
	}
	chID := channels[0].ID

	t.Run("returns existing channel", func(t *testing.T) {
		got, err := svc.GetChannelByID(ctx, chID, u.ID)
		if err != nil {
			t.Fatalf("GetChannelByID: %v", err)
		}
		if got.Name != "Ch" {
			t.Errorf("name = %q, want %q", got.Name, "Ch")
		}
	})

	t.Run("returns error for nonexistent channel", func(t *testing.T) {
		_, err := svc.GetChannelByID(ctx, 99999, u.ID)
		if err == nil {
			t.Fatal("expected error for nonexistent channel")
		}
	})
}

func TestUpdateChannel(t *testing.T) {
	svc, userRepo, channelRepo := setupAdminServiceWithMutators(t)
	ctx := context.Background()
	u, _ := userRepo.Create(ctx, "chupd@example.com", "chupduser", "pass")
	_ = channelRepo.Create(ctx, &delivery.Channel{UserID: u.ID, Kind: delivery.ChannelKindFeishu, Name: "Old Name", Configuration: delivery.ChannelConfiguration{"webhook_url": "https://a.com", "card_link_url": ""}})

	channels, _, _ := svc.ListAllDeliveryChannels(ctx, 0, 10)
	ch := channels[0]
	ch.Name = "New Name"

	t.Run("updates channel successfully", func(t *testing.T) {
		if err := svc.UpdateChannel(ctx, &ch); err != nil {
			t.Fatalf("UpdateChannel: %v", err)
		}
		got, _ := svc.GetChannelByID(ctx, ch.ID, u.ID)
		if got.Name != "New Name" {
			t.Errorf("name = %q, want %q", got.Name, "New Name")
		}
	})
}

func TestDeleteChannel(t *testing.T) {
	svc, userRepo, channelRepo := setupAdminServiceWithMutators(t)
	ctx := context.Background()
	u, _ := userRepo.Create(ctx, "chdel@example.com", "chdeluser", "pass")
	_ = channelRepo.Create(ctx, &delivery.Channel{UserID: u.ID, Kind: delivery.ChannelKindFeishu, Name: "To Delete", Configuration: delivery.ChannelConfiguration{"webhook_url": "https://a.com", "card_link_url": ""}})

	channels, _, _ := svc.ListAllDeliveryChannels(ctx, 0, 10)
	chID := channels[0].ID

	t.Run("deletes existing channel", func(t *testing.T) {
		deleted, err := svc.DeleteChannel(ctx, chID, u.ID)
		if err != nil {
			t.Fatalf("DeleteChannel: %v", err)
		}
		if deleted != 1 {
			t.Errorf("expected 1 deleted, got %d", deleted)
		}
	})

	t.Run("returns 0 for nonexistent channel", func(t *testing.T) {
		deleted, err := svc.DeleteChannel(ctx, 99999, u.ID)
		if err != nil {
			t.Fatalf("DeleteChannel: %v", err)
		}
		if deleted != 0 {
			t.Errorf("expected 0 deleted, got %d", deleted)
		}
	})
}

func TestListUserSessions(t *testing.T) {
	svc, _, _ := setupAdminServiceWithMutators(t)
	ctx := context.Background()

	t.Run("returns sessions for user", func(t *testing.T) {
		tokens, err := svc.ListUserSessions(ctx, 1)
		if err != nil {
			t.Fatalf("ListUserSessions: %v", err)
		}
		if tokens == nil {
			t.Error("expected non-nil slice")
		}
	})
}

func TestRevokeUserSessions(t *testing.T) {
	svc, _, _ := setupAdminServiceWithMutators(t)
	ctx := context.Background()

	t.Run("revokes sessions successfully", func(t *testing.T) {
		if err := svc.RevokeUserSessions(ctx, 1); err != nil {
			t.Fatalf("RevokeUserSessions: %v", err)
		}
	})
}

func TestRecordAudit(t *testing.T) {
	svc, _, _ := setupAdminServiceWithMutators(t)
	ctx := context.Background()

	t.Run("records audit entry", func(t *testing.T) {
		err := svc.RecordAudit(ctx, audit.Entry{
			ActorID:    1,
			Action:     "user.create",
			TargetType: "user",
			TargetID:   "42",
		})
		if err != nil {
			t.Fatalf("RecordAudit: %v", err)
		}
	})

	t.Run("records audit entry with metadata", func(t *testing.T) {
		err := svc.RecordAudit(ctx, audit.Entry{
			ActorID:    1,
			Action:     "user.set_role",
			TargetType: "user",
			TargetID:   "42",
			Metadata:   map[string]any{"role": "admin"},
		})
		if err != nil {
			t.Fatalf("RecordAudit: %v", err)
		}
	})
}

func TestListAuditLogs(t *testing.T) {
	svc, _, _ := setupAdminServiceWithMutators(t)
	ctx := context.Background()

	_ = svc.RecordAudit(ctx, audit.Entry{ActorID: 1, Action: "a1", TargetType: "user", TargetID: "1"})
	_ = svc.RecordAudit(ctx, audit.Entry{ActorID: 1, Action: "a2", TargetType: "user", TargetID: "2"})

	t.Run("returns audit logs", func(t *testing.T) {
		logs, total, err := svc.ListAuditLogs(ctx, 0, 10)
		if err != nil {
			t.Fatalf("ListAuditLogs: %v", err)
		}
		if total != 2 {
			t.Errorf("expected total 2, got %d", total)
		}
		if len(logs) != 2 {
			t.Errorf("expected 2 logs, got %d", len(logs))
		}
	})

	t.Run("paginates correctly", func(t *testing.T) {
		logs, total, err := svc.ListAuditLogs(ctx, 0, 1)
		if err != nil {
			t.Fatalf("ListAuditLogs: %v", err)
		}
		if total != 2 {
			t.Errorf("expected total 2, got %d", total)
		}
		if len(logs) != 1 {
			t.Errorf("expected 1 log, got %d", len(logs))
		}
	})

	t.Run("returns empty when offset exceeds total", func(t *testing.T) {
		logs, total, err := svc.ListAuditLogs(ctx, 100, 10)
		if err != nil {
			t.Fatalf("ListAuditLogs: %v", err)
		}
		if total != 2 {
			t.Errorf("expected total 2, got %d", total)
		}
		if len(logs) != 0 {
			t.Errorf("expected 0 logs, got %d", len(logs))
		}
	})
}

func TestListAllDeliveryHistory(t *testing.T) {
	svc, _, _ := setupAdminServiceWithMutators(t)
	ctx := context.Background()

	t.Run("returns empty when no history", func(t *testing.T) {
		result, total, err := svc.ListAllDeliveryHistory(ctx, 0, 10)
		if err != nil {
			t.Fatalf("ListAllDeliveryHistory: %v", err)
		}
		if total != 0 {
			t.Errorf("expected total 0, got %d", total)
		}
		if len(result) != 0 {
			t.Errorf("expected 0 results, got %d", len(result))
		}
	})
}
