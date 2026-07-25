// Package delivery provides delivery channel business logic and services.
package delivery

import (
	"context"
	"net"
	"strconv"
	"strings"

	"markpost/internal/config"
	"markpost/internal/domain/delivery"
	"markpost/internal/domain/post"
)

// PostDeliveryService is the Sender implementation for Feishu channels: it
// builds the card payload from the post + channel and issues the webhook call.
// Channel matching/filtering happens at enqueue time (in Dispatcher.Enqueue),
// so Send only handles a single already-resolved channel.
type PostDeliveryService struct {
	feishu *FeishuClient
}

// NewPostDeliveryService creates a PostDeliveryService using the configured
// Feishu request timeout.
func NewPostDeliveryService() *PostDeliveryService {
	cfg := config.Get()
	return &PostDeliveryService{
		feishu: NewFeishuClient(cfg.Delivery.RequestTimeout),
	}
}

// Send dispatches a Feishu card for the given post to the given channel. Any
// non-nil error causes the dispatcher to apply backoff or fail the attempt.
func (s *PostDeliveryService) Send(ctx context.Context, p *post.Post, channel *delivery.Channel) error {
	switch channel.Kind {
	case delivery.ChannelKindFeishu:
		cfg := config.Get()
		postURL := buildPostURL(cfg.Server.PublicURL, cfg.Server.Host, cfg.Server.Port, p.QID)
		bodyPreview := buildBodyPreview(p.Body, cfg.Delivery.BodyPreviewChars)
		return s.feishu.SendCard(ctx, CardDeliveryParams{
			WebhookURL:  feishuWebhookFromChannel(*channel),
			CardLinkURL: feishuCardLinkURLFromChannel(*channel),
			PostURL:     postURL,
			PostTitle:   p.Title,
			BodyPreview: bodyPreview,
			PostQID:     p.QID,
		})
	default:
		return errUnsupportedChannelKind{kind: string(channel.Kind)}
	}
}

// SendTest sends a fixed diagnostic card to the channel so the user can verify
// the webhook configuration is working without publishing a post. It is
// fire-and-forget: it does not enter the retry queue and does not write a
// delivery_history row.
func (s *PostDeliveryService) SendTest(ctx context.Context, channel *delivery.Channel) error {
	switch channel.Kind {
	case delivery.ChannelKindFeishu:
		return s.feishu.SendCard(ctx, CardDeliveryParams{
			WebhookURL:  feishuWebhookFromChannel(*channel),
			CardLinkURL: feishuCardLinkURLFromChannel(*channel),
			PostTitle:   testCardTitle,
			BodyPreview: testCardBody,
		})
	default:
		return errUnsupportedChannelKind{kind: string(channel.Kind)}
	}
}

const (
	testCardTitle = "Markpost test message"
	testCardBody  = "This is a test delivery from Markpost to confirm the channel is configured correctly."
)

type errUnsupportedChannelKind struct{ kind string }

func (e errUnsupportedChannelKind) Error() string { return "unsupported channel kind: " + e.kind }

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
