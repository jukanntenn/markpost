// Package delivery provides delivery channel business logic and services.
package delivery

import (
	"context"
	"encoding/json"
	"net/url"
	"strings"

	"markpost/internal/domain/delivery"
	"markpost/internal/service"
	"markpost/pkg/utils"
)

// UpdateChannelParams holds the parameters for creating or updating a delivery channel.
type UpdateChannelParams struct {
	Kind          string
	Name          string
	Configuration json.RawMessage
	Keywords      string
	Enabled       *bool
}

// Service provides delivery channel business logic.
type Service struct {
	repo delivery.Repository
}

// NewService creates a new Service instance.
func NewService(repo delivery.Repository) *Service {
	return &Service{repo: repo}
}

func normalizeAndValidateKind(kind string) (delivery.ChannelKind, error) {
	normalized := delivery.ChannelKind(utils.Normalize(kind))
	if !normalized.IsValid() {
		return "", service.NewServiceError(service.ErrValidation, "unsupported channel kind: "+string(normalized))
	}
	return normalized, nil
}

func validateConfiguration(kind delivery.ChannelKind, raw json.RawMessage) (delivery.ChannelConfiguration, error) {
	var config delivery.ChannelConfiguration
	if err := json.Unmarshal(raw, &config); err != nil {
		return nil, service.NewServiceError(service.ErrValidation, "invalid configuration JSON: "+err.Error())
	}

	switch kind {
	case delivery.ChannelKindFeishu:
		feishu := config.Feishu()
		if strings.TrimSpace(feishu.WebhookURL) == "" {
			return nil, service.NewServiceError(service.ErrValidation, "webhook URL is required")
		}
		parsedURL, err := url.Parse(strings.TrimSpace(feishu.WebhookURL))
		if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
			return nil, service.NewServiceError(service.ErrValidation, "invalid webhook URL: must be a valid HTTP or HTTPS URL")
		}
		config["webhook_url"] = strings.TrimSpace(feishu.WebhookURL)
		if _, ok := config["card_link_url"]; !ok {
			config["card_link_url"] = ""
		}
	default:
		return nil, service.NewServiceError(service.ErrValidation, "unsupported channel kind: "+string(kind))
	}

	return config, nil
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
func (s *Service) Create(ctx context.Context, userID int, params UpdateChannelParams) (*delivery.Channel, error) {
	cleanedName := strings.TrimSpace(params.Name)
	if cleanedName == "" {
		return nil, service.NewServiceError(service.ErrValidation, "channel name is required")
	}

	kind, err := normalizeAndValidateKind(params.Kind)
	if err != nil {
		return nil, err
	}

	if len(params.Configuration) == 0 {
		return nil, service.NewServiceError(service.ErrValidation, "configuration is required")
	}
	config, err := validateConfiguration(kind, params.Configuration)
	if err != nil {
		return nil, err
	}

	ch := &delivery.Channel{
		UserID:        userID,
		Kind:          kind,
		Name:          cleanedName,
		Enabled:       true,
		Configuration: config,
		Keywords:      strings.TrimSpace(params.Keywords),
	}

	if err := s.repo.Create(ctx, ch); err != nil {
		return nil, service.NewServiceErrorWrap(service.ErrInternal, "create channel failed", err)
	}

	return ch, nil
}

// Update updates an existing delivery channel.
func (s *Service) Update(ctx context.Context, userID int, id int, params UpdateChannelParams) (*delivery.Channel, error) {
	ch, err := s.repo.GetByIDAndUserID(ctx, id, userID)
	if err != nil {
		return nil, service.WrapNotFoundOrInternal(err, "channel not found", "get channel failed")
	}

	if params.Kind != "" {
		kind, err := normalizeAndValidateKind(params.Kind)
		if err != nil {
			return nil, err
		}
		ch.Kind = kind
	}
	utils.ApplyIfNonEmpty(&ch.Name, params.Name)
	if len(params.Configuration) > 0 {
		config, err := validateConfiguration(ch.Kind, params.Configuration)
		if err != nil {
			return nil, err
		}
		ch.Configuration = config
	}
	utils.ApplyIfNonEmpty(&ch.Keywords, params.Keywords)
	if params.Enabled != nil {
		ch.Enabled = *params.Enabled
	}

	if err := s.repo.Update(ctx, ch); err != nil {
		return nil, service.NewServiceErrorWrap(service.ErrInternal, "update channel failed", err)
	}

	return ch, nil
}

// Delete deletes a delivery channel by ID and user ID.
func (s *Service) Delete(ctx context.Context, userID int, id int) error {
	affected, err := s.repo.DeleteByIDAndUserID(ctx, id, userID)
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
	return service.Paginate(
		func() ([]delivery.Channel, error) { return s.repo.ListAll(ctx, offset, limit) },
		func() (int64, error) { return s.repo.CountAll(ctx) },
		"all channels",
	)
}
