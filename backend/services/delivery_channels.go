package services

import (
	"fmt"
	"net/url"
	"strings"

	"markpost/models"
	"markpost/repositories"
)

type DeliveryChannelService struct {
	repo repositories.DeliveryChannelRepoInterface
}

func NewDeliveryChannelService(repo repositories.DeliveryChannelRepoInterface) *DeliveryChannelService {
	return &DeliveryChannelService{repo: repo}
}

func (s *DeliveryChannelService) ListByUserID(userID int) ([]models.DeliveryChannel, error) {
	channels, err := s.repo.ListByUserID(userID)
	if err != nil {
		return nil, NewServiceErrorWrap(ErrInternal, "list delivery channels failed", err)
	}
	return channels, nil
}

func (s *DeliveryChannelService) Create(userID int, kind models.DeliveryChannelKind, name string, webhookURL string, keywords string, enabled bool) (*models.DeliveryChannel, error) {
	if err := validateChannel(kind, webhookURL); err != nil {
		return nil, err
	}

	channel, err := s.repo.Create(userID, kind, name, webhookURL, keywords, enabled)
	if err != nil {
		return nil, NewServiceErrorWrap(ErrInternal, "create delivery channel failed", err)
	}
	return channel, nil
}

func (s *DeliveryChannelService) Update(userID int, id int, name *string, webhookURL *string, keywords *string, enabled *bool) (*models.DeliveryChannel, error) {
	channel, err := s.repo.GetByIDAndUserID(id, userID)
	if err != nil {
		if err == models.ErrNotFound {
			return nil, NewServiceErrorWrap(ErrNotFound, "delivery channel not found", err)
		}
		return nil, NewServiceErrorWrap(ErrInternal, "get delivery channel failed", err)
	}

	if name != nil {
		channel.Name = *name
	}
	if enabled != nil {
		channel.Enabled = *enabled
	}
	if webhookURL != nil {
		if err := validateChannel(channel.Kind, *webhookURL); err != nil {
			return nil, err
		}
		channel.WebhookURL = *webhookURL
	}
	if keywords != nil {
		channel.Keywords = *keywords
	}

	if err := s.repo.Update(channel); err != nil {
		return nil, NewServiceErrorWrap(ErrInternal, "update delivery channel failed", err)
	}

	return channel, nil
}

func (s *DeliveryChannelService) Delete(userID int, id int) error {
	rows, err := s.repo.DeleteByIDAndUserID(id, userID)
	if err != nil {
		return NewServiceErrorWrap(ErrInternal, "delete delivery channel failed", err)
	}
	if rows == 0 {
		return NewServiceError(ErrNotFound, "delivery channel not found")
	}
	return nil
}

func validateChannel(kind models.DeliveryChannelKind, webhookURL string) *ServiceError {
	if strings.TrimSpace(webhookURL) == "" {
		return NewServiceErrorWithDetails(ErrValidation, "request validation failed", []ServiceError{
			{Code: ErrRequired, Description: "webhook_url"},
		})
	}

	u, err := url.Parse(webhookURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return NewServiceErrorWithDetails(ErrValidation, "request validation failed", []ServiceError{
			{Code: ErrFieldViolation, Description: "webhook_url"},
		})
	}
	if u.Scheme != "https" {
		return NewServiceErrorWithDetails(ErrValidation, "request validation failed", []ServiceError{
			{Code: ErrFieldViolation, Description: "webhook_url"},
		})
	}

	switch kind {
	case models.DeliveryChannelKindFeishu:
		host := strings.ToLower(u.Hostname())
		if host == "open.feishu.cn" || strings.HasSuffix(host, ".feishu.cn") || strings.HasSuffix(host, ".larksuite.com") {
			return nil
		}
		return NewServiceErrorWithDetails(ErrValidation, "request validation failed", []ServiceError{
			{Code: ErrFieldViolation, Description: "webhook_url"},
		})
	default:
		return NewServiceErrorWithDetails(ErrValidation, "request validation failed", []ServiceError{
			{Code: ErrFieldViolation, Description: "kind"},
		})
	}
}

func ParseDeliveryChannelKind(raw string) (models.DeliveryChannelKind, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case string(models.DeliveryChannelKindFeishu):
		return models.DeliveryChannelKindFeishu, nil
	default:
		return "", fmt.Errorf("unknown delivery channel kind: %s", raw)
	}
}
