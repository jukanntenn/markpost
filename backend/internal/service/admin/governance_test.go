package admin

import (
	"context"
	"testing"
	"time"

	"markpost/internal/domain/delivery"
	"markpost/internal/domain/post"
	"markpost/internal/domain/user"
	"markpost/internal/infra"
	"markpost/internal/service"
)

type postListerAdapter struct {
	repo post.Repository
}

func (a *postListerAdapter) GetAllPosts(ctx context.Context, search, username string, offset, limit int) ([]post.Post, int64, error) {
	items, err := a.repo.ListAll(ctx, search, username, offset, limit)
	if err != nil {
		return nil, 0, err
	}
	count, err := a.repo.CountAll(ctx, search, username)
	if err != nil {
		return nil, 0, err
	}
	return items, count, nil
}

func (a *postListerAdapter) CountAllPosts(ctx context.Context) (int64, error) {
	return a.repo.CountAll(ctx, "", "")
}

func (a *postListerAdapter) CountSince(ctx context.Context, since time.Time) (int64, error) {
	return a.repo.CountSince(ctx, since)
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

func (a *channelListerAdapter) CountAll(ctx context.Context) (int64, error) {
	return a.repo.CountAll(ctx)
}

func newAdminService(t *testing.T) (*Service, user.Repository, user.TokenRepository) {
	t.Helper()
	db := infra.SetupTestDB(t)
	userRepo := infra.NewUserRepository(db, 16)
	tokenRepo := infra.NewTokenRepository(db)
	postRepo := infra.NewPostRepository(db)
	channelRepo := infra.NewDeliveryChannelRepository(db)
	attemptRepo := infra.NewAttemptRepository(db)
	auditRepo := infra.NewAuditRepository(db)

	concrete, ok := userRepo.(*infra.UserRepository)
	if !ok {
		t.Fatal("userRepo not *infra.UserRepository")
	}
	svc := NewService(
		concrete,
		&postListerAdapter{repo: postRepo},
		&channelListerAdapter{repo: channelRepo},
		attemptRepo,
		tokenRepo,
		auditRepo,
	)
	svc.SetUserMutator(concrete)
	svc.SetChannelMutator(channelRepo)
	return svc, userRepo, tokenRepo
}

// K.7 D3-3 / I.4 权限矩阵：防自降级 + 防最后一个 admin。
func TestService_GovernanceGuards(t *testing.T) {
	t.Run("cannot change own role (self_forbidden)", func(t *testing.T) {
		svc, userRepo, _ := newAdminService(t)
		admin, _ := userRepo.Create(context.Background(), "a@example.com", "alice", "correctpass")
		_ = userRepo.SetRole(context.Background(), admin.ID, user.RoleAdmin)

		err := svc.SetUserRole(context.Background(), admin.ID, admin.ID, user.RoleUser)
		se, ok := service.AsError(err)
		if !ok || se.Code != ErrSelfForbidden {
			t.Fatalf("expected self_forbidden, got %v", err)
		}
	})

	t.Run("cannot disable yourself (self_forbidden)", func(t *testing.T) {
		svc, userRepo, _ := newAdminService(t)
		admin, _ := userRepo.Create(context.Background(), "b@example.com", "bob", "correctpass")
		err := svc.SetUserActive(context.Background(), admin.ID, admin.ID, false)
		se, ok := service.AsError(err)
		if !ok || se.Code != ErrSelfForbidden {
			t.Fatalf("expected self_forbidden, got %v", err)
		}
	})

	t.Run("cannot delete yourself (self_forbidden)", func(t *testing.T) {
		svc, userRepo, _ := newAdminService(t)
		admin, _ := userRepo.Create(context.Background(), "c@example.com", "carol", "correctpass")
		_, err := svc.DeleteUser(context.Background(), admin.ID, admin.ID)
		se, ok := service.AsError(err)
		if !ok || se.Code != ErrSelfForbidden {
			t.Fatalf("expected self_forbidden, got %v", err)
		}
	})

	t.Run("cannot demote the last admin (last_admin)", func(t *testing.T) {
		svc, userRepo, _ := newAdminService(t)
		admin, _ := userRepo.Create(context.Background(), "d@example.com", "dave", "correctpass")
		_ = userRepo.SetRole(context.Background(), admin.ID, user.RoleAdmin)
		// 另一个普通用户（actor 不是 target，但 target 是唯一 admin）。
		user1, _ := userRepo.Create(context.Background(), "e@example.com", "erin", "correctpass")

		err := svc.SetUserRole(context.Background(), user1.ID, admin.ID, user.RoleUser)
		se, ok := service.AsError(err)
		if !ok || se.Code != ErrLastAdmin {
			t.Fatalf("expected last_admin, got %v", err)
		}
	})

	t.Run("demoting a non-last admin succeeds and bumps token_version", func(t *testing.T) {
		svc, userRepo, _ := newAdminService(t)
		admin1, _ := userRepo.Create(context.Background(), "f@example.com", "frank", "correctpass")
		admin2, _ := userRepo.Create(context.Background(), "g@example.com", "grace", "correctpass")
		_ = userRepo.SetRole(context.Background(), admin1.ID, user.RoleAdmin)
		_ = userRepo.SetRole(context.Background(), admin2.ID, user.RoleAdmin)
		actor, _ := userRepo.Create(context.Background(), "h@example.com", "heidi", "correctpass")

		if err := svc.SetUserRole(context.Background(), actor.ID, admin2.ID, user.RoleUser); err != nil {
			t.Fatalf("expected demotion to succeed, got %v", err)
		}
		u, _ := userRepo.GetByID(context.Background(), admin2.ID)
		if u.IsAdmin() {
			t.Error("expected admin2 demoted")
		}
		if u.TokenVersion != 1 {
			t.Errorf("expected token_version bumped to 1, got %d", u.TokenVersion)
		}
	})

	t.Run("cannot delete the last admin (last_admin)", func(t *testing.T) {
		svc, userRepo, _ := newAdminService(t)
		admin, _ := userRepo.Create(context.Background(), "i@example.com", "ivan", "correctpass")
		_ = userRepo.SetRole(context.Background(), admin.ID, user.RoleAdmin)
		actor, _ := userRepo.Create(context.Background(), "j@example.com", "judy", "correctpass")

		_, err := svc.DeleteUser(context.Background(), actor.ID, admin.ID)
		se, ok := service.AsError(err)
		if !ok || se.Code != ErrLastAdmin {
			t.Fatalf("expected last_admin, got %v", err)
		}
	})
}

// C3.3：封禁 → is_active=false + token_version++（会话即时失效）。
func TestService_SetUserActive_BumpsTokenVersionOnBan(t *testing.T) {
	svc, userRepo, _ := newAdminService(t)
	u, _ := userRepo.Create(context.Background(), "k@example.com", "kate", "correctpass")
	actor, _ := userRepo.Create(context.Background(), "l@example.com", "leo", "correctpass")

	if err := svc.SetUserActive(context.Background(), actor.ID, u.ID, false); err != nil {
		t.Fatalf("ban: %v", err)
	}
	banned, _ := userRepo.GetByID(context.Background(), u.ID)
	if banned.IsActive {
		t.Error("expected user inactive")
	}
	if banned.TokenVersion != 1 {
		t.Errorf("expected token_version 1 after ban, got %d", banned.TokenVersion)
	}

	// 解封不 bump（token_version 保持，已有 token 在封禁时已失效）。
	if err := svc.SetUserActive(context.Background(), actor.ID, u.ID, true); err != nil {
		t.Fatalf("unban: %v", err)
	}
	unbanned, _ := userRepo.GetByID(context.Background(), u.ID)
	if !unbanned.IsActive {
		t.Error("expected user active again")
	}
	if unbanned.TokenVersion != 1 {
		t.Errorf("expected token_version still 1 after unban, got %d", unbanned.TokenVersion)
	}
}

// C3.3：强制下线 → token_version++ + 全部 refresh token 吊销。
func TestService_RevokeUserSessions(t *testing.T) {
	svc, userRepo, tokenRepo := newAdminService(t)
	u, _ := userRepo.Create(context.Background(), "m@example.com", "mike", "correctpass")

	if err := svc.RevokeUserSessions(context.Background(), u.ID); err != nil {
		t.Fatalf("revoke sessions: %v", err)
	}
	updated, _ := userRepo.GetByID(context.Background(), u.ID)
	if updated.TokenVersion != 1 {
		t.Errorf("expected token_version 1 after force sign-out, got %d", updated.TokenVersion)
	}
	_ = tokenRepo
}

// D3.3 方案 B：重置密码生成 12 字符临时密码，旧会话吊销。
func TestService_ResetUserPassword_ReturnsPlaintextOnce(t *testing.T) {
	svc, userRepo, _ := newAdminService(t)
	u, _ := userRepo.Create(context.Background(), "n@example.com", "nina", "correctpass")

	password, err := svc.ResetUserPassword(context.Background(), u.ID)
	if err != nil {
		t.Fatalf("reset password: %v", err)
	}
	if len(password) != 12 {
		t.Errorf("expected 12-char password, got %q", password)
	}
	updated, _ := userRepo.GetByID(context.Background(), u.ID)
	if updated.TokenVersion != 1 {
		t.Errorf("expected token_version 1 after reset, got %d", updated.TokenVersion)
	}
}

// C2.3：CreateUser 密码策略预检。
func TestService_CreateUser_PasswordPolicy(t *testing.T) {
	svc, _, _ := newAdminService(t)
	_, err := svc.CreateUser(context.Background(), "o@example.com", "olivia", "short")
	se, ok := service.AsError(err)
	if !ok || se.Code.Value != "password_too_short" {
		t.Fatalf("expected password_too_short, got %v", err)
	}
}

// I.3 SSRF：admin 创建渠道路径同样拒绝私有/保留地址（不能绕过用户侧校验）。
func TestService_CreateChannel_SSRFAdminPath(t *testing.T) {
	svc, userRepo, _ := newAdminService(t)
	u, _ := userRepo.Create(context.Background(), "ssrf@example.com", "ssrfuser", "correctpass")

	cases := []struct {
		name    string
		webhook string
		want    string
	}{
		{"loopback", "http://127.0.0.1:8080/hook", "webhook_url_forbidden"},
		{"link-local metadata", "http://169.254.169.254/latest/meta-data/", "webhook_url_forbidden"},
		{"private 10/8", "http://10.0.0.1/hook", "webhook_url_forbidden"},
		{"private 172.16/12", "http://172.16.5.5/hook", "webhook_url_forbidden"},
		{"private 192.168/16", "http://192.168.1.1/hook", "webhook_url_forbidden"},
		// 非 http(s) scheme 走语法校验（与用户侧 validateConfiguration 同语义）。
		{"non-http scheme", "ftp://example.com/hook", "validation"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ch := &delivery.Channel{
				UserID: u.ID,
				Kind:   delivery.ChannelKindFeishu,
				Name:   "ssrf-admin",
				Configuration: delivery.ChannelConfiguration{
					"webhook_url": tc.webhook,
				},
				Enabled: true,
			}
			err := svc.CreateChannel(context.Background(), ch)
			se, ok := service.AsError(err)
			if !ok || se.Code.Value != tc.want {
				t.Fatalf("expected %s, got %v", tc.want, err)
			}
		})
	}

	t.Run("public webhook allowed", func(t *testing.T) {
		ch := &delivery.Channel{
			UserID: u.ID,
			Kind:   delivery.ChannelKindFeishu,
			Name:   "ssrf-ok",
			Configuration: delivery.ChannelConfiguration{
				"webhook_url": "https://open.feishu.cn/open-apis/bot/v2/hook/test",
			},
			Enabled: true,
		}
		if err := svc.CreateChannel(context.Background(), ch); err != nil {
			t.Fatalf("expected public webhook to pass, got %v", err)
		}
	})
}
