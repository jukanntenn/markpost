// Package admin provides admin-level business logic and services.
package admin

import (
	"context"
	"errors"
	"fmt"
	"time"

	"markpost/internal/domain"

	"markpost/internal/domain/audit"
	"markpost/internal/domain/delivery"
	"markpost/internal/domain/post"
	"markpost/internal/domain/settings"
	"markpost/internal/domain/user"
	"markpost/internal/service"
	"markpost/internal/service/auth"
	deliverySvc "markpost/internal/service/delivery"
	"markpost/pkg/utils"
)

// RetentionCounter counts the rows a candidate retention window would delete
// (impact preview), for posts and delivery_history alike.
type RetentionCounter interface {
	CountExpiringForUsers(ctx context.Context, userIDs []int, vipOnly bool, cutoff *time.Time) (int64, error)
}

// UserLister defines the interface for retrieving users.
type UserLister interface {
	GetAll(ctx context.Context, offset, limit int) ([]user.User, error)
	Search(ctx context.Context, search string, offset, limit int) ([]user.User, error)
	CountSearch(ctx context.Context, search string) (int64, error)
	Count(ctx context.Context) (int64, error)
	CountByRole(ctx context.Context, role user.Role) (int64, error)
	CountBanned(ctx context.Context) (int64, error)
	CountSince(ctx context.Context, since time.Time) (int64, error)
	CountVIP(ctx context.Context) (int64, error)
}

// UserMutator defines the interface for modifying users.
type UserMutator interface {
	Create(ctx context.Context, email, username, password string) (*user.User, error)
	GetByID(ctx context.Context, id int) (*user.User, error)
	SetRole(ctx context.Context, userID int, role user.Role) error
	SetPassword(ctx context.Context, userID int, password string) error
	SetActive(ctx context.Context, userID int, active bool) error
	SetUserVIP(ctx context.Context, userID int, vip bool, retentionIfUnset *int) error
	SetUserRetention(ctx context.Context, userID int, days *int) error
	SetUserRetentionBatch(ctx context.Context, userIDs []int, days *int) (int64, error)
	SetVIPUsersRetention(ctx context.Context, days *int) (int64, error)
	DeleteByID(ctx context.Context, userID int) (int64, error)
	// BumpTokenVersion increments token_version — the single primitive behind
	// instant invalidation of all of a user's tokens (C2.6).
	BumpTokenVersion(ctx context.Context, userID int) error
}

// ChannelMutator defines the interface for modifying delivery channels.
type ChannelMutator interface {
	Create(ctx context.Context, channel *delivery.Channel) error
	GetByID(ctx context.Context, id int) (*delivery.Channel, error)
	GetByIDAndUserID(ctx context.Context, id int, userID int) (*delivery.Channel, error)
	Update(ctx context.Context, channel *delivery.Channel) error
	SetEnabled(ctx context.Context, id int, enabled bool) error
	DeleteByIDAndUserID(ctx context.Context, id int, userID int) (int64, error)
	DeleteByID(ctx context.Context, id int) (int64, error)
}

// SessionLister defines the interface for listing/revoking user sessions.
type SessionLister interface {
	ListByUserID(ctx context.Context, userID int) ([]user.RefreshToken, error)
	RevokeAllByUserID(ctx context.Context, userID int) error
	RevokeRefreshTokenByID(ctx context.Context, tokenID, userID int) error
	GetRefreshTokenByID(ctx context.Context, tokenID int) (*user.RefreshToken, error)
}

// PostLister defines the interface for retrieving posts.
type PostLister interface {
	GetAllPosts(ctx context.Context, search, username string, offset, limit int) ([]post.Post, int64, error)
	CountAllPosts(ctx context.Context) (int64, error)
	CountSince(ctx context.Context, since time.Time) (int64, error)
}

// ChannelLister defines the interface for retrieving delivery channels.
type ChannelLister interface {
	ListAll(ctx context.Context, offset, limit int) ([]delivery.Channel, int64, error)
	CountAll(ctx context.Context) (int64, error)
}

// HistoryLister defines the interface for retrieving delivery history.
type HistoryLister interface {
	ListHistory(ctx context.Context, filter delivery.HistoryFilter, offset, limit int) ([]*delivery.HistoryRow, error)
	CountHistory(ctx context.Context, filter delivery.HistoryFilter) (int64, error)
	CountSince(ctx context.Context, since time.Time) (int64, error)
	DailyStatsAll(ctx context.Context, days int) ([]*delivery.DailyStat, error)
	LockedChannels(ctx context.Context) ([]*delivery.LockedChannel, error)
}

// AuditRecorder defines the interface for recording and reading audit logs.
type AuditRecorder interface {
	Record(ctx context.Context, e audit.Entry) error
	List(ctx context.Context, filter audit.AuditFilter, offset, limit int) ([]audit.LogRow, int64, error)
	ActionCounts(ctx context.Context, filter audit.AuditFilter) (map[string]int64, error)
}

// SettingsStore defines the interface for runtime settings access (MRFC
// 2026-08-23-github-login-vip-grant-strategy).
type SettingsStore interface {
	GetAll(ctx context.Context) ([]settings.Setting, error)
	Set(ctx context.Context, key string, value settings.SettingValue, updatedBy int) error
	// VIPRetentionDays is the class default materialized at grant time.
	VIPRetentionDays(ctx context.Context) (*int, error)
}

// Service provides admin-level business logic.
type Service struct {
	userLister     UserLister
	userMutator    UserMutator
	postLister     PostLister
	channelLister  ChannelLister
	channelMutator ChannelMutator
	historyLister  HistoryLister
	sessionLister  SessionLister
	auditRecorder  AuditRecorder
	settingsStore  SettingsStore

	postExpiryCounter    RetentionCounter
	historyExpiryCounter RetentionCounter
	// Global fallbacks mirrored from config for impact previews and the
	// defaults endpoint: the effective window of an inherit (nil) policy.
	globalPostRetentionDays int
	globalHistoryRetention  time.Duration
}

// NewService creates a new admin Service instance.
func NewService(
	userLister UserLister,
	postLister PostLister,
	channelLister ChannelLister,
	historyLister HistoryLister,
	sessionLister SessionLister,
	auditRecorder AuditRecorder,
) *Service {
	return &Service{
		userLister:    userLister,
		postLister:    postLister,
		channelLister: channelLister,
		historyLister: historyLister,
		sessionLister: sessionLister,
		auditRecorder: auditRecorder,
	}
}

// SetUserMutator sets the user mutator for write operations.
func (s *Service) SetUserMutator(mutator UserMutator) {
	s.userMutator = mutator
}

// SetChannelMutator sets the channel mutator for write operations.
func (s *Service) SetChannelMutator(mutator ChannelMutator) {
	s.channelMutator = mutator
}

// SetSettingsStore sets the runtime settings store.
func (s *Service) SetSettingsStore(store SettingsStore) {
	s.settingsStore = store
}

// SetRetentionCounters wires the posts and delivery_history impact counters
// (retention preview) plus the global fallback windows mirrored from config.
func (s *Service) SetRetentionCounters(posts, history RetentionCounter, globalPostRetentionDays int, globalHistoryRetention time.Duration) {
	s.postExpiryCounter = posts
	s.historyExpiryCounter = history
	s.globalPostRetentionDays = globalPostRetentionDays
	s.globalHistoryRetention = globalHistoryRetention
}

// RecordAudit records an admin write operation for audit purposes.
func (s *Service) RecordAudit(ctx context.Context, e audit.Entry) error {
	return s.auditRecorder.Record(ctx, e)
}

// ListAllUsers retrieves all users with pagination and an optional username
// search (D3.1).
func (s *Service) ListAllUsers(ctx context.Context, search string, offset, limit int) ([]user.User, int64, error) {
	if search == "" {
		return service.Paginate(
			func() ([]user.User, error) { return s.userLister.GetAll(ctx, offset, limit) },
			func() (int64, error) { return s.userLister.Count(ctx) },
			"users",
		)
	}
	return service.Paginate(
		func() ([]user.User, error) { return s.userLister.Search(ctx, search, offset, limit) },
		func() (int64, error) { return s.userLister.CountSearch(ctx, search) },
		"users",
	)
}

// ListAllPosts retrieves all posts with optional search and username filter (F.9).
func (s *Service) ListAllPosts(ctx context.Context, search, username string, offset, limit int) ([]post.Post, int64, error) {
	return s.postLister.GetAllPosts(ctx, search, username, offset, limit)
}

// ListAllDeliveryChannels retrieves all delivery channels with pagination.
func (s *Service) ListAllDeliveryChannels(ctx context.Context, offset, limit int) ([]delivery.Channel, int64, error) {
	return s.channelLister.ListAll(ctx, offset, limit)
}

// ListAllDeliveryHistory retrieves delivery history (all users, including
// anonymized rows) with user/channel/status filters (F.8).
func (s *Service) ListAllDeliveryHistory(ctx context.Context, filter delivery.HistoryFilter, offset, limit int) ([]*delivery.HistoryRow, int64, error) {
	return service.Paginate(
		func() ([]*delivery.HistoryRow, error) { return s.historyLister.ListHistory(ctx, filter, offset, limit) },
		func() (int64, error) { return s.historyLister.CountHistory(ctx, filter) },
		"delivery history",
	)
}

// ListAuditLogs retrieves audit logs with optional filters (D4.3).
func (s *Service) ListAuditLogs(ctx context.Context, filter audit.AuditFilter, offset, limit int) ([]audit.LogRow, int64, error) {
	return s.auditRecorder.List(ctx, filter, offset, limit)
}

// AuditActionCounts returns the per-action log counts under the filter
// (D4.3 筛选计数).
func (s *Service) AuditActionCounts(ctx context.Context, filter audit.AuditFilter) (map[string]int64, error) {
	return s.auditRecorder.ActionCounts(ctx, filter)
}

// CreateUser creates a new user (admin operation) with the shared password
// policy pre-check (C2.3).
func (s *Service) CreateUser(ctx context.Context, email, username, password string) (*user.User, error) {
	if err := utils.ValidatePasswordPolicy(password); err != nil {
		if utils.TooShort(password) {
			return nil, service.New(auth.ErrPasswordTooShort, "password too short")
		}
		return nil, service.New(auth.ErrPasswordTooLong, "password too long")
	}
	u, err := s.userMutator.Create(ctx, email, username, password)
	if err != nil {
		// 用户名/邮箱占用 → 409 conflict（I.10 契约），其余 → internal。
		if errors.Is(err, domain.ErrUsernameTaken) || errors.Is(err, domain.ErrEmailTaken) {
			return nil, service.New(service.ErrConflict, "username or email already taken")
		}
		return nil, service.Wrap(service.ErrInternal, "create user failed", err)
	}
	return u, nil
}

// SetUserRole updates a user's role (admin operation) with the
// anti-self-demotion and last-admin guards (K.7 D3-3) plus token_version++
// (D3.3: 角色切换立即踢出已有 admin 会话).
func (s *Service) SetUserRole(ctx context.Context, actorID, userID int, role user.Role) error {
	if actorID == userID {
		return service.New(ErrSelfForbidden, "cannot change own role")
	}
	target, err := s.userMutator.GetByID(ctx, userID)
	if err != nil {
		return service.WrapNotFoundOrInternal(err, "user not found", "get user failed")
	}
	if target.IsAdmin() && role != user.RoleAdmin {
		admins, err := s.userLister.CountByRole(ctx, user.RoleAdmin)
		if err != nil {
			return service.Wrap(service.ErrInternal, "count admins failed", err)
		}
		if admins <= 1 {
			return service.New(ErrLastAdmin, "cannot demote the last admin")
		}
	}
	if err := s.userMutator.SetRole(ctx, userID, role); err != nil {
		return service.Wrap(service.ErrInternal, "set role failed", err)
	}
	return s.bump(ctx, userID)
}

// ResetUserPassword resets a user's password (方案 B, D3.3): the system
// generates a random temporary password (returned in plaintext exactly once),
// then bumps token_version and revokes all sessions — the user must sign in
// again with the new password.
func (s *Service) ResetUserPassword(ctx context.Context, userID int) (string, error) {
	password, err := utils.GenerateRandomPassword(12)
	if err != nil {
		return "", service.Wrap(service.ErrInternal, "generate temporary password failed", err)
	}
	if err := s.userMutator.SetPassword(ctx, userID, password); err != nil {
		return "", service.Wrap(service.ErrInternal, "set password failed", err)
	}
	if err := s.bump(ctx, userID); err != nil {
		return "", err
	}
	if err := s.sessionLister.RevokeAllByUserID(ctx, userID); err != nil {
		return "", service.Wrap(service.ErrInternal, "revoke sessions failed", err)
	}
	return password, nil
}

// SetUserActive enables/disables a user (admin operation). Disabling bumps
// token_version so the user's existing sessions die instantly (C3.3); an
// admin cannot disable themselves (I.4).
func (s *Service) SetUserActive(ctx context.Context, actorID, userID int, active bool) error {
	if !active && actorID == userID {
		return service.New(ErrSelfForbidden, "cannot disable yourself")
	}
	if err := s.userMutator.SetActive(ctx, userID, active); err != nil {
		return service.Wrap(service.ErrInternal, "set active failed", err)
	}
	if !active {
		return s.bump(ctx, userID)
	}
	return nil
}

// SetUserVIP writes the per-user VIP honorific (admin operation). No
// self-targeting guard (no invariant breaks) and no token_version bump: vip
// rides no claim and carries no authority — the middleware re-reads the row
// each request, so the toggle is visible immediately (MRFC
// 2026-08-23-vip-badge-and-admin-management).
func (s *Service) SetUserVIP(ctx context.Context, userID int, vip bool) error {
	if _, err := s.userMutator.GetByID(ctx, userID); err != nil {
		return service.WrapNotFoundOrInternal(err, "user not found", "get user failed")
	}
	// Grant-time materialization: a VIP still inheriting takes the class
	// default in the same statement; an explicit policy survives untouched.
	var classDefault *int
	if vip {
		days, err := s.settingsStore.VIPRetentionDays(ctx)
		if err != nil && !errors.Is(err, domain.ErrNotFound) {
			return service.Wrap(service.ErrInternal, "read vip retention default failed", err)
		}
		if errors.Is(err, domain.ErrNotFound) {
			days = nil
		}
		classDefault = days
	}
	if err := s.userMutator.SetUserVIP(ctx, userID, vip, classDefault); err != nil {
		return service.Wrap(service.ErrInternal, "set vip failed", err)
	}
	return nil
}

// MaxRetentionDays bounds an explicit retention policy (10 years); 0 means
// keep forever, mirroring [post] retention_days' encoding (MRFC
// 2026-08-31-per-user-history-retention-policy).
const MaxRetentionDays = 3650

func validateRetentionDays(days *int) error {
	if days != nil && (*days < 0 || *days > MaxRetentionDays) {
		return service.New(ErrRetentionDays, fmt.Sprintf("retention days %d out of range", *days))
	}
	return nil
}

// SetUserRetention writes one user's retention policy (nil = inherit) and
// returns the updated user. The handler records the user.set_retention audit.
func (s *Service) SetUserRetention(ctx context.Context, userID int, days *int) (*user.User, error) {
	if err := validateRetentionDays(days); err != nil {
		return nil, err
	}
	if _, err := s.userMutator.GetByID(ctx, userID); err != nil {
		return nil, service.WrapNotFoundOrInternal(err, "user not found", "get user failed")
	}
	if err := s.userMutator.SetUserRetention(ctx, userID, days); err != nil {
		return nil, service.Wrap(service.ErrInternal, "set retention failed", err)
	}
	return s.userMutator.GetByID(ctx, userID)
}

// BulkSetUserRetention writes the policy onto explicit user ids (scope "") or
// every VIP user (scope "vip"), returning the affected count. The handler
// records the single user.set_retention_bulk audit entry.
func (s *Service) BulkSetUserRetention(ctx context.Context, userIDs []int, scope string, days *int) (int64, error) {
	if err := validateRetentionDays(days); err != nil {
		return 0, err
	}
	if scope == "vip" {
		updated, err := s.userMutator.SetVIPUsersRetention(ctx, days)
		if err != nil {
			return 0, service.Wrap(service.ErrInternal, "bulk set retention failed", err)
		}
		return updated, nil
	}
	if len(userIDs) == 0 {
		return 0, service.New(ErrRetentionDays, "no target users")
	}
	updated, err := s.userMutator.SetUserRetentionBatch(ctx, userIDs, days)
	if err != nil {
		return 0, service.Wrap(service.ErrInternal, "bulk set retention failed", err)
	}
	return updated, nil
}

// RetentionImpactResult is the deletion preview for a candidate policy.
type RetentionImpactResult struct {
	UsersAffected   int64 `json:"users_affected"`
	PostsToDelete   int64 `json:"posts_to_delete"`
	HistoryToDelete int64 `json:"history_to_delete"`
}

// RetentionImpact previews what a candidate policy would delete at the next
// sweep for explicit user ids or every VIP user. A candidate of nil resolves
// against each table's global fallback; 0 (forever) matches nothing.
func (s *Service) RetentionImpact(ctx context.Context, userIDs []int, scope string, days *int) (*RetentionImpactResult, error) {
	if err := validateRetentionDays(days); err != nil {
		return nil, err
	}
	vipOnly := scope == "vip"
	if !vipOnly && len(userIDs) == 0 {
		return nil, service.New(ErrRetentionDays, "no target users")
	}

	postCutoff := s.candidateCutoff(days, s.globalPostRetentionDays)
	historyCutoff := s.candidateCutoffDuration(days, s.globalHistoryRetention)

	out := &RetentionImpactResult{}
	if vipOnly {
		n, err := s.userLister.CountVIP(ctx)
		if err != nil {
			return nil, service.Wrap(service.ErrInternal, "count vip users failed", err)
		}
		out.UsersAffected = n
	} else {
		out.UsersAffected = int64(len(userIDs))
	}
	if s.postExpiryCounter != nil {
		n, err := s.postExpiryCounter.CountExpiringForUsers(ctx, userIDs, vipOnly, postCutoff)
		if err != nil {
			return nil, service.Wrap(service.ErrInternal, "count expiring posts failed", err)
		}
		out.PostsToDelete = n
	}
	if s.historyExpiryCounter != nil {
		n, err := s.historyExpiryCounter.CountExpiringForUsers(ctx, userIDs, vipOnly, historyCutoff)
		if err != nil {
			return nil, service.Wrap(service.ErrInternal, "count expiring history failed", err)
		}
		out.HistoryToDelete = n
	}
	return out, nil
}

// candidateCutoff maps a candidate policy to a posts cutoff: nil inherits the
// global fallback, 0 (forever) becomes a nil cutoff that matches nothing.
func (s *Service) candidateCutoff(days *int, globalDays int) *time.Time {
	effective := globalDays
	if days != nil {
		effective = *days
	}
	if effective == 0 {
		return nil
	}
	cutoff := time.Now().AddDate(0, 0, -effective)
	return &cutoff
}

// candidateCutoffDuration is candidateCutoff for the duration-shaped history
// global fallback.
func (s *Service) candidateCutoffDuration(days *int, globalRetention time.Duration) *time.Time {
	if days != nil {
		if *days == 0 {
			return nil
		}
		cutoff := time.Now().AddDate(0, 0, -*days)
		return &cutoff
	}
	cutoff := time.Now().Add(-globalRetention)
	return &cutoff
}

// RetentionDefaults reports the global fallback windows so the admin UI can
// render the effective value of inherit policies.
func (s *Service) RetentionDefaults(ctx context.Context) (*RetentionDefaultsResult, error) {
	return &RetentionDefaultsResult{PostRetentionDays: s.globalPostRetentionDays}, nil
}

// RetentionDefaultsResult is the global retention configuration surface.
type RetentionDefaultsResult struct {
	PostRetentionDays int `json:"post_retention_days"`
}

// DeleteUser deletes a user (admin operation) with the self-delete and
// last-admin guards (K.7 D3-3).
func (s *Service) DeleteUser(ctx context.Context, actorID, userID int) (int64, error) {
	if actorID == userID {
		return 0, service.New(ErrSelfForbidden, "cannot delete yourself")
	}
	target, err := s.userMutator.GetByID(ctx, userID)
	if err != nil {
		return 0, service.WrapNotFoundOrInternal(err, "user not found", "get user failed")
	}
	if target.IsAdmin() {
		admins, err := s.userLister.CountByRole(ctx, user.RoleAdmin)
		if err != nil {
			return 0, service.Wrap(service.ErrInternal, "count admins failed", err)
		}
		if admins <= 1 {
			return 0, service.New(ErrLastAdmin, "cannot delete the last admin")
		}
	}
	affected, err := s.userMutator.DeleteByID(ctx, userID)
	if err != nil {
		return 0, service.Wrap(service.ErrInternal, "delete user failed", err)
	}
	return affected, nil
}

// GetUserByID retrieves a user by ID (admin operation; D3.2 资料区).
func (s *Service) GetUserByID(ctx context.Context, userID int) (*user.User, error) {
	u, err := s.userMutator.GetByID(ctx, userID)
	if err != nil {
		return nil, service.WrapNotFoundOrInternal(err, "user not found", "get user failed")
	}
	return u, nil
}

// RevokeUserSessions forces a user's sessions to die instantly (admin
// operation, C3.3): revoke all refresh tokens + bump token_version.
func (s *Service) RevokeUserSessions(ctx context.Context, userID int) error {
	if err := s.sessionLister.RevokeAllByUserID(ctx, userID); err != nil {
		return service.Wrap(service.ErrInternal, "revoke sessions failed", err)
	}
	return s.bump(ctx, userID)
}

// RevokeSessionByID revokes a single refresh token (admin single-session
// revoke, D3.2): the token's owner is resolved from the row itself.
func (s *Service) RevokeSessionByID(ctx context.Context, tokenID int) error {
	token, err := s.sessionLister.GetRefreshTokenByID(ctx, tokenID)
	if err != nil {
		return service.WrapNotFoundOrInternal(err, "session not found", "get session failed")
	}
	if err := s.sessionLister.RevokeRefreshTokenByID(ctx, tokenID, token.UserID); err != nil {
		return service.WrapNotFoundOrInternal(err, "session not found", "revoke session failed")
	}
	return nil
}

// bump increments the target user's token_version.
func (s *Service) bump(ctx context.Context, userID int) error {
	if err := s.userMutator.BumpTokenVersion(ctx, userID); err != nil {
		return service.Wrap(service.ErrInternal, "bump token version failed", err)
	}
	return nil
}

// CreateChannel creates a new delivery channel (admin operation).
func (s *Service) CreateChannel(ctx context.Context, channel *delivery.Channel) error {
	// I.3 SSRF 防护：admin 创建渠道路径同样必须通过 webhook 校验
	// （规范无条件要求"创建/更新渠道时校验 webhook_url"）。
	if err := deliverySvc.ValidateChannelSSRF(ctx, channel.Kind, channel.Configuration); err != nil {
		return err
	}
	return s.channelMutator.Create(ctx, channel)
}

// GetChannelByID retrieves a channel by ID (admin operation).
func (s *Service) GetChannelByID(ctx context.Context, id int, userID int) (*delivery.Channel, error) {
	return s.channelMutator.GetByIDAndUserID(ctx, id, userID)
}

// UpdateChannel updates a delivery channel (admin operation).
func (s *Service) UpdateChannel(ctx context.Context, channel *delivery.Channel) error {
	return s.channelMutator.Update(ctx, channel)
}

// SetChannelEnabled enables or disables a delivery channel (admin operation).
func (s *Service) SetChannelEnabled(ctx context.Context, id int, enabled bool) error {
	return s.channelMutator.SetEnabled(ctx, id, enabled)
}

// DeleteChannel deletes a delivery channel (admin operation).
func (s *Service) DeleteChannel(ctx context.Context, id int, userID int) (int64, error) {
	return s.channelMutator.DeleteByIDAndUserID(ctx, id, userID)
}

// DeleteChannelByID deletes a delivery channel by ID (admin operation).
func (s *Service) DeleteChannelByID(ctx context.Context, id int) (int64, error) {
	return s.channelMutator.DeleteByID(ctx, id)
}

// ListUserSessions lists all refresh tokens for a user (admin operation).
func (s *Service) ListUserSessions(ctx context.Context, userID int) ([]user.RefreshToken, error) {
	tokens, err := s.sessionLister.ListByUserID(ctx, userID)
	if err != nil {
		return nil, service.Wrap(service.ErrInternal, "list user sessions failed", err)
	}
	// I.10 契约：空列表序列化为 [] 而非 null。
	if tokens == nil {
		tokens = []user.RefreshToken{}
	}
	return tokens, nil
}

// Stats holds the admin overview stock metrics plus week-over-week deltas
// (D2.4).
type Stats struct {
	Users            int64 `json:"users"`
	Posts            int64 `json:"posts"`
	Channels         int64 `json:"channels"`
	History          int64 `json:"history"`
	BannedUsers      int64 `json:"banned_users"`
	UsersWeekDelta   int64 `json:"users_week_delta"`
	PostsWeekDelta   int64 `json:"posts_week_delta"`
	HistoryWeekDelta int64 `json:"history_week_delta"`
}

// GetStats returns the stock metrics and their 7-day deltas (D2.4).
func (s *Service) GetStats(ctx context.Context) (*Stats, error) {
	since := time.Now().AddDate(0, 0, -7)

	banned, err := s.userLister.CountBanned(ctx)
	if err != nil {
		return nil, service.Wrap(service.ErrInternal, "count banned users failed", err)
	}
	users, err := s.userLister.Count(ctx)
	if err != nil {
		return nil, service.Wrap(service.ErrInternal, "count users failed", err)
	}
	posts, err := s.postLister.CountAllPosts(ctx)
	if err != nil {
		return nil, service.Wrap(service.ErrInternal, "count posts failed", err)
	}
	channels, err := s.channelLister.CountAll(ctx)
	if err != nil {
		return nil, service.Wrap(service.ErrInternal, "count channels failed", err)
	}
	history, err := s.historyLister.CountHistory(ctx, delivery.HistoryFilter{})
	if err != nil {
		return nil, service.Wrap(service.ErrInternal, "count history failed", err)
	}
	usersDelta, err := s.userLister.CountSince(ctx, since)
	if err != nil {
		return nil, service.Wrap(service.ErrInternal, "count users delta failed", err)
	}
	postsDelta, err := s.postLister.CountSince(ctx, since)
	if err != nil {
		return nil, service.Wrap(service.ErrInternal, "count posts delta failed", err)
	}
	historyDelta, err := s.historyLister.CountSince(ctx, since)
	if err != nil {
		return nil, service.Wrap(service.ErrInternal, "count history delta failed", err)
	}
	return &Stats{
		Users:            users,
		BannedUsers:      banned,
		Posts:            posts,
		Channels:         channels,
		History:          history,
		UsersWeekDelta:   usersDelta,
		PostsWeekDelta:   postsDelta,
		HistoryWeekDelta: historyDelta,
	}, nil
}

// GetSettings returns every runtime setting row (admin operation).
func (s *Service) GetSettings(ctx context.Context) ([]settings.Setting, error) {
	rows, err := s.settingsStore.GetAll(ctx)
	if err != nil {
		return nil, service.Wrap(service.ErrInternal, "list settings failed", err)
	}
	// I.10 契约：空列表序列化为 [] 而非 null。
	if rows == nil {
		rows = []settings.Setting{}
	}
	return rows, nil
}

// SetSetting upserts one runtime setting (admin operation). v1 admits only
// the seeded keys — an unknown key is a client error, not a silent new row.
func (s *Service) SetSetting(ctx context.Context, actorID int, key string, value settings.SettingValue) error {
	switch key {
	case settings.KeyVIP:
		// The switch key owns the {"enabled"} shape; a stray days payload
		// would silently no-op, so it is rejected instead.
		if value.Days != nil {
			return service.New(ErrSettingValueShape, "vip takes {\"enabled\"}, not days")
		}
	case settings.KeyVIPRetention:
		// The class default owns the {"days"} shape; days nil = follow the
		// global config (clearing the class default). A stray enabled payload
		// would silently no-op, so it is rejected instead.
		if value.Enabled {
			return service.New(ErrSettingValueShape, "vip_retention_days takes {\"days\"}, not enabled")
		}
		if value.Days != nil && (*value.Days < 0 || *value.Days > MaxRetentionDays) {
			return service.New(ErrRetentionDays, fmt.Sprintf("days %d out of range", *value.Days))
		}
	default:
		return service.New(ErrUnknownSetting, fmt.Sprintf("unknown setting key %q", key))
	}
	if err := s.settingsStore.Set(ctx, key, value, actorID); err != nil {
		return service.Wrap(service.ErrInternal, "set setting failed", err)
	}
	return nil
}

// DailyStatsAll returns the site-wide per-day delivery counts (D2.5).
func (s *Service) DailyStatsAll(ctx context.Context, days int) ([]*delivery.DailyStat, error) {
	return s.historyLister.DailyStatsAll(ctx, days)
}

// LockedChannels returns channels flagged by the failing-channel query
// (D2.1/K.7) for the admin "需要关注" card.
func (s *Service) LockedChannels(ctx context.Context) ([]*delivery.LockedChannel, error) {
	rows, err := s.historyLister.LockedChannels(ctx)
	if err != nil {
		return nil, service.Wrap(service.ErrInternal, "locked channels query failed", err)
	}
	// I.10 契约：空列表序列化为 [] 而非 null。
	if rows == nil {
		rows = []*delivery.LockedChannel{}
	}
	return rows, nil
}
