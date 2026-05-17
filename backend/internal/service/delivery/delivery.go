// Package delivery provides delivery channel business logic and services.
package delivery

import (
	"context"
	"errors"
	"net/url"
	"strings"

	"markpost/internal/domain/delivery"
	"markpost/internal/service"
)

// Repository defines the interface for delivery channel data access.
type Repository interface {
	GetByUserID(ctx context.Context, userID int) ([]delivery.Channel, error)
	GetByIDAndUserID(ctx context.Context, id int, userID int) (*delivery.Channel, error)
	Create(ctx context.Context, channel *delivery.Channel) error
	Update(ctx context.Context, channel *delivery.Channel) error
	DeleteByIDAndUserID(ctx context.Context, id int, userID int) (int64, error)
	ListAll(ctx context.Context, offset, limit int) ([]delivery.Channel, error)
	CountAll(ctx context.Context) (int64, error)
}

type CreateChannelParams struct {
	Kind       string
	Name       string
	WebhookURL string
	Keywords   string
}

type UpdateChannelParams struct {
	Kind       string
	Name       string
	WebhookURL string
	Keywords   string
	Enabled    *bool
}

// Service provides delivery channel business logic.
type Service struct {
	repo Repository
}

// NewService creates a new Service instance.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// validateChannel validates a delivery channel configuration.
func validateChannel(kind string, webhookURL string, name string) error {
	if strings.TrimSpace(name) == "" {
		return service.NewServiceError(service.ErrValidation, "channel name is required")
	}

	kind = strings.TrimSpace(strings.ToLower(kind))
	if kind != string(delivery.ChannelKindFeishu) {
		return service.NewServiceError(service.ErrValidation, "unsupported channel kind: "+kind)
	}

	webhookURL = strings.TrimSpace(webhookURL)
	if webhookURL == "" {
		return service.NewServiceError(service.ErrValidation, "webhook URL is required")
	}

	parsedURL, err := url.Parse(webhookURL)
	if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		return service.NewServiceError(service.ErrValidation, "invalid webhook URL: must be a valid HTTP or HTTPS URL")
	}

	return nil
}

// ListByUserID lists all delivery channels for a user.
func (s *Service) ListByUserID(ctx context.Context, userID int) ([]delivery.Channel, error) {
	channels, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, service.NewServiceErrorWrap(service.ErrInternal, "list channels failed", err)
	}
	return channels, nil
}

// Create creates a new delivery channel for a user.
func (s *Service) Create(ctx context.Context, userID int, params CreateChannelParams) (*delivery.Channel, error) {
	if err := validateChannel(params.Kind, params.WebhookURL, params.Name); err != nil {
		return nil, err
	}

	ch := &delivery.Channel{
		UserID:     userID,
		Kind:       delivery.ChannelKind(strings.ToLower(strings.TrimSpace(params.Kind))),
		Name:       strings.TrimSpace(params.Name),
		Enabled:    true,
		WebhookURL: strings.TrimSpace(params.WebhookURL),
		Keywords:   strings.TrimSpace(params.Keywords),
	}

	if err := s.repo.Create(ctx, ch); err != nil {
		return nil, service.NewServiceErrorWrap(service.ErrInternal, "create channel failed", err)
	}

	return ch, nil
}

// Update updates an existing delivery channel.
func (s *Service) Update(ctx context.Context, userID int, id int, params UpdateChannelParams) (*delivery.Channel, error) {
	ch, err := s.repo.GetByIDAndUserID(ctx, userID, id)
	if err != nil {
		if errors.Is(err, delivery.ErrNotFound) {
			return nil, service.NewServiceErrorWrap(service.ErrNotFound, "channel not found", err)
		}
		return nil, service.NewServiceErrorWrap(service.ErrInternal, "get channel failed", err)
	}

	if params.Kind != "" {
		ch.Kind = delivery.ChannelKind(strings.ToLower(strings.TrimSpace(params.Kind)))
	}
	applyIfNonEmpty(&ch.Name, params.Name)
	applyIfNonEmpty(&ch.WebhookURL, params.WebhookURL)
	applyIfNonEmpty(&ch.Keywords, params.Keywords)
	if params.Enabled != nil {
		ch.Enabled = *params.Enabled
	}

	if params.Kind != "" || params.Name != "" || params.WebhookURL != "" {
		if err := validateChannel(string(ch.Kind), ch.WebhookURL, ch.Name); err != nil {
			return nil, err
		}
	}

	if err := s.repo.Update(ctx, ch); err != nil {
		return nil, service.NewServiceErrorWrap(service.ErrInternal, "update channel failed", err)
	}

	return ch, nil
}

// Delete deletes a delivery channel by ID and user ID.
func (s *Service) Delete(ctx context.Context, userID int, id int) error {
	affected, err := s.repo.DeleteByIDAndUserID(ctx, userID, id)
	if err != nil {
		return service.NewServiceErrorWrap(service.ErrInternal, "delete channel failed", err)
	}

	if affected == 0 {
		return service.NewServiceError(service.ErrNotFound, "channel not found")
	}

	return nil
}

// ListAll lists all delivery channels with pagination (admin use).
func (s *Service) ListAll(ctx context.Context, offset, limit int) ([]delivery.Channel, int64, error) {
	channels, err := s.repo.ListAll(ctx, offset, limit)
	if err != nil {
		return nil, 0, service.NewServiceErrorWrap(service.ErrInternal, "list all channels failed", err)
	}

	total, err := s.repo.CountAll(ctx)
	if err != nil {
		return nil, 0, service.NewServiceErrorWrap(service.ErrInternal, "count channels failed", err)
	}

	return channels, total, nil
}

func applyIfNonEmpty(target *string, value string) {
	if value != "" {
		*target = strings.TrimSpace(value)
	}
}
