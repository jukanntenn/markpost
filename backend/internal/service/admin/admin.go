// Package admin provides admin-level business logic and services.
package admin

import (
	"context"

	"markpost/internal/domain/audit"
	"markpost/internal/domain/delivery"
	"markpost/internal/domain/post"
	"markpost/internal/domain/user"
	"markpost/internal/service"
)

// UserLister defines the interface for retrieving users.
type UserLister interface {
	GetAll(ctx context.Context, offset, limit int) ([]user.User, error)
	Count(ctx context.Context) (int64, error)
}

// UserMutator defines the interface for modifying users.
type UserMutator interface {
	Create(ctx context.Context, email, username, password string) (*user.User, error)
	GetByID(ctx context.Context, id int) (*user.User, error)
	SetRole(ctx context.Context, userID int, role user.Role) error
	SetPassword(ctx context.Context, userID int, password string) error
	SetActive(ctx context.Context, userID int, active bool) error
	DeleteByID(ctx context.Context, userID int) (int64, error)
}

// ChannelMutator defines the interface for modifying delivery channels.
type ChannelMutator interface {
	Create(ctx context.Context, channel *delivery.Channel) error
	GetByIDAndUserID(ctx context.Context, id int, userID int) (*delivery.Channel, error)
	Update(ctx context.Context, channel *delivery.Channel) error
	DeleteByIDAndUserID(ctx context.Context, id int, userID int) (int64, error)
}

// SessionLister defines the interface for listing user sessions.
type SessionLister interface {
	ListByUserID(ctx context.Context, userID int) ([]user.RefreshToken, error)
	RevokeAllByUserID(ctx context.Context, userID int) error
}

// PostLister defines the interface for retrieving posts.
type PostLister interface {
	GetAllPosts(ctx context.Context, search string, offset, limit int) ([]post.Post, int64, error)
}

// ChannelLister defines the interface for retrieving delivery channels.
type ChannelLister interface {
	ListAll(ctx context.Context, offset, limit int) ([]delivery.Channel, int64, error)
}

// HistoryLister defines the interface for retrieving delivery history.
type HistoryLister interface {
	ListHistory(ctx context.Context, filter delivery.HistoryFilter, offset, limit int) ([]*delivery.HistoryRow, error)
	CountHistory(ctx context.Context, filter delivery.HistoryFilter) (int64, error)
}

// AuditRecorder defines the interface for recording audit logs.
type AuditRecorder interface {
	Record(ctx context.Context, e audit.Entry) error
	List(ctx context.Context, offset, limit int) ([]audit.Log, int64, error)
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

// RecordAudit records an admin write operation for audit purposes.
func (s *Service) RecordAudit(ctx context.Context, e audit.Entry) error {
	return s.auditRecorder.Record(ctx, e)
}

// ListAllUsers retrieves all users with pagination.
func (s *Service) ListAllUsers(ctx context.Context, offset, limit int) ([]user.User, int64, error) {
	return service.Paginate(
		func() ([]user.User, error) { return s.userLister.GetAll(ctx, offset, limit) },
		func() (int64, error) { return s.userLister.Count(ctx) },
		"users",
	)
}

// ListAllPosts retrieves all posts with optional search and pagination.
func (s *Service) ListAllPosts(ctx context.Context, search string, offset, limit int) ([]post.Post, int64, error) {
	return s.postLister.GetAllPosts(ctx, search, offset, limit)
}

// ListAllDeliveryChannels retrieves all delivery channels with pagination.
func (s *Service) ListAllDeliveryChannels(ctx context.Context, offset, limit int) ([]delivery.Channel, int64, error) {
	return s.channelLister.ListAll(ctx, offset, limit)
}

// ListAllDeliveryHistory retrieves all delivery history (all users, including
// anonymized rows) with pagination.
func (s *Service) ListAllDeliveryHistory(ctx context.Context, offset, limit int) ([]*delivery.HistoryRow, int64, error) {
	filter := delivery.HistoryFilter{}
	return service.Paginate(
		func() ([]*delivery.HistoryRow, error) { return s.historyLister.ListHistory(ctx, filter, offset, limit) },
		func() (int64, error) { return s.historyLister.CountHistory(ctx, filter) },
		"delivery history",
	)
}

// ListAuditLogs retrieves audit logs with pagination.
func (s *Service) ListAuditLogs(ctx context.Context, offset, limit int) ([]audit.Log, int64, error) {
	return s.auditRecorder.List(ctx, offset, limit)
}

// CreateUser creates a new user (admin operation).
func (s *Service) CreateUser(ctx context.Context, email, username, password string) (*user.User, error) {
	return s.userMutator.Create(ctx, email, username, password)
}

// SetUserRole updates a user's role (admin operation).
func (s *Service) SetUserRole(ctx context.Context, userID int, role user.Role) error {
	return s.userMutator.SetRole(ctx, userID, role)
}

// ResetUserPassword resets a user's password (admin operation).
func (s *Service) ResetUserPassword(ctx context.Context, userID int, password string) error {
	return s.userMutator.SetPassword(ctx, userID, password)
}

// SetUserActive updates a user's active status (admin operation).
func (s *Service) SetUserActive(ctx context.Context, userID int, active bool) error {
	return s.userMutator.SetActive(ctx, userID, active)
}

// DeleteUser deletes a user (admin operation).
func (s *Service) DeleteUser(ctx context.Context, userID int) (int64, error) {
	return s.userMutator.DeleteByID(ctx, userID)
}

// GetUserByID retrieves a user by ID (admin operation).
func (s *Service) GetUserByID(ctx context.Context, userID int) (*user.User, error) {
	return s.userMutator.GetByID(ctx, userID)
}

// CreateChannel creates a new delivery channel (admin operation).
func (s *Service) CreateChannel(ctx context.Context, channel *delivery.Channel) error {
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

// DeleteChannel deletes a delivery channel (admin operation).
func (s *Service) DeleteChannel(ctx context.Context, id int, userID int) (int64, error) {
	return s.channelMutator.DeleteByIDAndUserID(ctx, id, userID)
}

// ListUserSessions lists all refresh tokens for a user (admin operation).
func (s *Service) ListUserSessions(ctx context.Context, userID int) ([]user.RefreshToken, error) {
	return s.sessionLister.ListByUserID(ctx, userID)
}

// RevokeUserSessions revokes all refresh tokens for a user (admin operation).
func (s *Service) RevokeUserSessions(ctx context.Context, userID int) error {
	return s.sessionLister.RevokeAllByUserID(ctx, userID)
}
