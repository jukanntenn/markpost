// Package delivery provides delivery channel business logic and services.
package delivery

import (
	"context"
	"encoding/json"
	"strings"

	"markpost/internal/domain/delivery"
	"markpost/internal/service"
	"markpost/internal/service/delivery/filter"
	"markpost/pkg/utils"
)

// UpdateChannelParams holds the parameters for creating or updating a delivery channel.
type UpdateChannelParams struct {
	Kind          string
	Name          string
	Configuration json.RawMessage
	Keywords      *string
	Enabled       *bool
}

// Service provides delivery channel business logic.
type Service struct {
	repo        delivery.Repository
	attemptRepo delivery.AttemptRepository
}

// NewService creates a new Service instance.
func NewService(repo delivery.Repository, attemptRepo delivery.AttemptRepository) *Service {
	return &Service{repo: repo, attemptRepo: attemptRepo}
}

func normalizeAndValidateKind(kind string) (delivery.ChannelKind, error) {
	normalized := delivery.ChannelKind(utils.Normalize(kind))
	if !normalized.IsValid() {
		return "", service.New(service.ErrValidation, "unsupported channel kind: "+string(normalized))
	}
	return normalized, nil
}

func validateConfiguration(ctx context.Context, kind delivery.ChannelKind, raw json.RawMessage) (delivery.ChannelConfiguration, error) {
	var config delivery.ChannelConfiguration
	if err := json.Unmarshal(raw, &config); err != nil {
		return nil, service.New(service.ErrValidation, "invalid configuration JSON: "+err.Error())
	}

	switch kind {
	case delivery.ChannelKindFeishu:
		feishu := config.Feishu()
		webhookURL := strings.TrimSpace(feishu.WebhookURL)
		if webhookURL == "" {
			return nil, service.New(service.ErrValidation, "webhook URL is required")
		}
		if !isAllowedWebhookURL(webhookURL) {
			return nil, service.New(service.ErrValidation, "invalid webhook URL: must be a valid HTTP or HTTPS URL")
		}
		// I.3 SSRF 防护：拒绝私有/保留地址。
		if err := validateWebhookURLSSRF(ctx, webhookURL); err != nil {
			return nil, err
		}
		config["webhook_url"] = webhookURL
		if _, ok := config["card_link_url"]; !ok {
			config["card_link_url"] = ""
		}
	default:
		return nil, service.New(service.ErrValidation, "unsupported channel kind: "+string(kind))
	}

	return config, nil
}

// ListByUserID lists all delivery channels for a user.
func (s *Service) ListByUserID(ctx context.Context, userID int) ([]delivery.Channel, error) {
	channels, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, service.Wrap(service.ErrInternal, "list channels failed", err)
	}
	return channels, nil
}

// Create creates a new delivery channel for a user.
func (s *Service) Create(ctx context.Context, userID int, params UpdateChannelParams) (*delivery.Channel, error) {
	cleanedName := strings.TrimSpace(params.Name)
	if cleanedName == "" {
		return nil, service.New(service.ErrValidation, "channel name is required")
	}

	kind, err := normalizeAndValidateKind(params.Kind)
	if err != nil {
		return nil, err
	}

	if len(params.Configuration) == 0 {
		return nil, service.New(service.ErrValidation, "configuration is required")
	}
	config, err := validateConfiguration(ctx, kind, params.Configuration)
	if err != nil {
		return nil, err
	}

	keywords := ""
	if params.Keywords != nil {
		keywords = strings.TrimSpace(*params.Keywords)
	}
	if _, err := filter.Compile(keywords); err != nil {
		return nil, service.New(service.ErrValidation, "invalid keywords expression: "+err.Error())
	}

	ch := &delivery.Channel{
		UserID:        userID,
		Kind:          kind,
		Name:          cleanedName,
		Enabled:       true,
		Configuration: config,
		Keywords:      keywords,
	}

	if err := s.repo.Create(ctx, ch); err != nil {
		return nil, service.Wrap(service.ErrInternal, "create channel failed", err)
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
	if len(params.Configuration) > 0 {
		config, err := validateConfiguration(ctx, ch.Kind, params.Configuration)
		if err != nil {
			return nil, err
		}
		ch.Configuration = config
	}
	if params.Keywords != nil {
		normalized := strings.TrimSpace(*params.Keywords)
		if _, err := filter.Compile(normalized); err != nil {
			return nil, service.New(service.ErrValidation, "invalid keywords expression: "+err.Error())
		}
		ch.Keywords = normalized
	}
	utils.ApplyIfNonEmpty(&ch.Name, params.Name)
	if params.Enabled != nil {
		ch.Enabled = *params.Enabled
	}

	if err := s.repo.Update(ctx, ch); err != nil {
		return nil, service.Wrap(service.ErrInternal, "update channel failed", err)
	}

	return ch, nil
}

// Delete deletes a delivery channel by ID and user ID.
func (s *Service) Delete(ctx context.Context, userID int, id int) error {
	affected, err := s.repo.DeleteByIDAndUserID(ctx, id, userID)
	if err != nil {
		return service.Wrap(service.ErrInternal, "delete channel failed", err)
	}

	if affected == 0 {
		return service.New(service.ErrNotFound, "channel not found")
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

// CountAll returns the total channel count (admin stats, D2.4).
func (s *Service) CountAll(ctx context.Context) (int64, error) {
	return s.repo.CountAll(ctx)
}

// ListHistory lists a user's own delivery history (newest first) with
// pagination. channelID > 0 further scopes the result to one channel;
// status > 0 filters by terminal status (B3.4 状态 filter).
func (s *Service) ListHistory(ctx context.Context, userID, channelID int, status delivery.Status, offset, limit int) ([]*delivery.HistoryRow, int64, error) {
	filter := delivery.HistoryFilter{OwnerID: userID, ChannelID: channelID, Status: status}
	return service.Paginate(
		func() ([]*delivery.HistoryRow, error) { return s.attemptRepo.ListHistory(ctx, filter, offset, limit) },
		func() (int64, error) { return s.attemptRepo.CountHistory(ctx, filter) },
		"delivery history",
	)
}

// ListHistoryAll is the admin cross-user history listing (F.8): user/channel/
// status filters, OwnerID 0 = all rows including anonymized ones.
func (s *Service) ListHistoryAll(ctx context.Context, userID, channelID int, status delivery.Status, offset, limit int) ([]*delivery.HistoryRow, int64, error) {
	filter := delivery.HistoryFilter{OwnerID: userID, ChannelID: channelID, Status: status}
	return service.Paginate(
		func() ([]*delivery.HistoryRow, error) { return s.attemptRepo.ListHistory(ctx, filter, offset, limit) },
		func() (int64, error) { return s.attemptRepo.CountHistory(ctx, filter) },
		"all delivery history",
	)
}

// PendingAttempts returns the user's in-flight attempts joined to their post
// and channel — the dashboard activity feed's "投递中" data source (K.2).
func (s *Service) PendingAttempts(ctx context.Context, userID int) ([]*delivery.PendingAttemptRow, error) {
	rows, err := s.attemptRepo.ListPending(ctx, userID)
	if err != nil {
		return nil, service.Wrap(service.ErrInternal, "list pending attempts failed", err)
	}
	// I.10 契约：空列表序列化为 [] 而非 null。
	if rows == nil {
		rows = []*delivery.PendingAttemptRow{}
	}
	return rows, nil
}

// DailyStats returns the user's per-day terminal delivery counts (B2.7).
func (s *Service) DailyStats(ctx context.Context, userID, days int) ([]*delivery.DailyStat, error) {
	rows, err := s.attemptRepo.DailyStats(ctx, userID, days)
	if err != nil {
		return nil, service.Wrap(service.ErrInternal, "daily stats failed", err)
	}
	if rows == nil {
		rows = []*delivery.DailyStat{}
	}
	return rows, nil
}

// DailyStatsAll is the admin cross-user per-day counts (D2.5).
func (s *Service) DailyStatsAll(ctx context.Context, days int) ([]*delivery.DailyStat, error) {
	rows, err := s.attemptRepo.DailyStatsAll(ctx, days)
	if err != nil {
		return nil, service.Wrap(service.ErrInternal, "admin daily stats failed", err)
	}
	if rows == nil {
		rows = []*delivery.DailyStat{}
	}
	return rows, nil
}

// TodayCounts returns the user's today counters for the pipeline status bar
// (K.2: delivered/failed/pending).
func (s *Service) TodayCounts(ctx context.Context, userID int) (*delivery.TodayCounts, error) {
	counts, err := s.attemptRepo.TodayCounts(ctx, userID)
	if err != nil {
		return nil, service.Wrap(service.ErrInternal, "today counts failed", err)
	}
	return counts, nil
}

// LockedChannels returns channels flagged by the failing-channel query
// (D2.1/K.7) for the admin "需要关注" card.
func (s *Service) LockedChannels(ctx context.Context) ([]*delivery.LockedChannel, error) {
	rows, err := s.attemptRepo.LockedChannels(ctx)
	if err != nil {
		return nil, service.Wrap(service.ErrInternal, "locked channels query failed", err)
	}
	// I.10 契约：空列表序列化为 [] 而非 null。
	if rows == nil {
		rows = []*delivery.LockedChannel{}
	}
	return rows, nil
}

// LatestPerChannel returns the most recent delivery_history row for each of the
// user's channels, for the per-channel delivery-health overview.
func (s *Service) LatestPerChannel(ctx context.Context, userID int) ([]*delivery.HistoryRow, error) {
	rows, err := s.attemptRepo.LatestPerChannel(ctx, userID)
	if err != nil {
		return nil, service.Wrap(service.ErrInternal, "load latest per channel failed", err)
	}
	return rows, nil
}

// SendTest sends a diagnostic card to the channel to verify its webhook
// configuration. It does not enter the retry queue and writes no history.
func (s *Service) SendTest(ctx context.Context, userID, id int) error {
	ch, err := s.repo.GetByIDAndUserID(ctx, id, userID)
	if err != nil {
		return service.WrapNotFoundOrInternal(err, "channel not found", "get channel failed")
	}
	// The sender holds only an HTTP client keyed by the configured request
	// timeout; constructing it per call is cheap and keeps this method
	// independent of dispatcher wiring.
	sender := NewPostDeliveryService()
	if err := sender.SendTest(ctx, ch); err != nil {
		return service.Wrap(service.ErrInternal, "send test message failed", err)
	}
	return nil
}
