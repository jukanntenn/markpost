// Package delivery provides delivery channel business logic and services.
package delivery

import (
	"context"
	"log"
	"net"
	"strconv"
	"strings"

	"markpost/internal/config"
	"markpost/internal/domain/delivery"
	"markpost/internal/service/delivery/filter"
	"markpost/internal/service/post"
)

// PostDeliveryService matches delivery jobs against channel keywords and sends notifications.
type PostDeliveryService struct {
	repo   delivery.Repository
	feishu *FeishuClient
}

// NewPostDeliveryService creates a PostDeliveryService with the given channel repository.
func NewPostDeliveryService(repo delivery.Repository) *PostDeliveryService {
	cfg := config.Get()
	return &PostDeliveryService{
		repo:   repo,
		feishu: NewFeishuClient(cfg.Delivery.RequestTimeout),
	}
}

// Deliver finds matching channels for a post and sends notifications.
func (s *PostDeliveryService) Deliver(ctx context.Context, job post.DeliveryJob) {
	channels, err := s.repo.GetByUserID(ctx, job.UserID)
	if err != nil {
		log.Printf("delivery list channels failed user_id=%d err=%v", job.UserID, err)
		return
	}

	cfg := config.Get()
	postURL := buildPostURL(cfg.Server.PublicURL, cfg.Server.Host, cfg.Server.Port, job.PostQID)
	bodyPreview := buildBodyPreview(job.Body, cfg.Delivery.BodyPreviewChars)

	for _, channel := range channels {
		if !channel.Enabled {
			continue
		}

		matches, err := filter.Compile(channel.Keywords)
		if err != nil {
			log.Printf("delivery skip channel with invalid keywords channel_id=%d user_id=%d err=%v", channel.ID, channel.UserID, err)
			continue
		}
		if !matches.Match(job.Title) {
			continue
		}

		switch channel.Kind {
		case delivery.ChannelKindFeishu:
			webhookURL := feishuWebhookFromChannel(channel)
			cardLinkURL := feishuCardLinkURLFromChannel(channel)
			if err := s.feishu.SendCard(ctx, CardDeliveryParams{
				WebhookURL:  webhookURL,
				CardLinkURL: cardLinkURL,
				PostURL:     postURL,
				PostTitle:   job.Title,
				BodyPreview: bodyPreview,
				PostQID:     job.PostQID,
			}); err != nil {
				log.Printf("delivery feishu failed channel_id=%d user_id=%d err=%v", channel.ID, channel.UserID, err)
			}
		default:
			log.Printf("delivery unsupported channel kind=%s channel_id=%d user_id=%d", channel.Kind, channel.ID, channel.UserID)
		}
	}
}

func buildBodyPreview(body string, maxChars int) string {
	preview := strings.TrimSpace(body)
	preview = truncateRunes(preview, maxChars)
	if preview != "" && preview != strings.TrimSpace(body) {
		preview += "…"
	}
	return preview
}

func truncateRunes(s string, maxRunes int) string {
	if maxRunes <= 0 || s == "" {
		return ""
	}

	r := []rune(s)
	if len(r) <= maxRunes {
		return s
	}
	return string(r[:maxRunes])
}

func buildPostURL(publicURL string, host string, port uint16, qid string) string {
	base := strings.TrimRight(strings.TrimSpace(publicURL), "/")
	if base == "" {
		h := strings.TrimSpace(host)
		if h == "" || h == "0.0.0.0" {
			h = "127.0.0.1"
		}
		base = "http://" + net.JoinHostPort(h, strconv.Itoa(int(port)))
	}

	if strings.TrimSpace(qid) == "" {
		return base
	}
	return base + "/" + strings.TrimLeft(qid, "/")
}
