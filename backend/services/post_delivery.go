package services

import (
	"context"
	"log"
	"net"
	"strconv"
	"strings"

	"markpost/conf"
	"markpost/models"
	"markpost/repositories"
)

type PostDeliveryService struct {
	channelRepo repositories.DeliveryChannelRepoInterface
	feishu      *FeishuClient
}

func NewPostDeliveryService(channelRepo repositories.DeliveryChannelRepoInterface) *PostDeliveryService {
	cfg := conf.Conf()
	return &PostDeliveryService{
		channelRepo: channelRepo,
		feishu:      NewFeishuClient(cfg.Delivery.RequestTimeout),
	}
}

type DeliveryJob struct {
	UserID  int
	PostQID string
	Title   string
	Body    string
}

func (s *PostDeliveryService) Deliver(ctx context.Context, job DeliveryJob) {
	channels, err := s.channelRepo.ListByUserID(job.UserID)
	if err != nil {
		log.Printf("delivery list channels failed user_id=%d err=%v", job.UserID, err)
		return
	}

	cfg := conf.Conf()
	postURL := buildPostURL(cfg.Server.PublicURL, cfg.Server.Host, cfg.Server.Port, job.PostQID)
	message := buildDeliveryMessage(job.Title, job.Body, postURL, cfg.Delivery.BodyPreviewChars)

	for _, channel := range channels {
		if !channel.Enabled {
			continue
		}

		switch channel.Kind {
		case models.DeliveryChannelKindFeishu:
			if err := s.feishu.SendText(ctx, channel.WebhookURL, message); err != nil {
				log.Printf("delivery feishu failed channel_id=%d user_id=%d err=%v", channel.ID, channel.UserID, err)
			}
		default:
			log.Printf("delivery unsupported channel kind=%s channel_id=%d user_id=%d", channel.Kind, channel.ID, channel.UserID)
		}
	}
}

func truncateRunes(s string, max int) string {
	if max <= 0 || s == "" {
		return ""
	}

	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max])
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

func buildDeliveryMessage(title string, body string, postURL string, bodyPreviewChars int) string {
	titleText := strings.TrimSpace(title)
	if titleText == "" {
		titleText = "New post"
	}

	bodyPreview := strings.TrimSpace(body)
	bodyPreview = truncateRunes(bodyPreview, bodyPreviewChars)
	if bodyPreview != "" && bodyPreview != strings.TrimSpace(body) {
		bodyPreview += "…"
	}

	postURL = strings.TrimSpace(postURL)
	if bodyPreview == "" {
		if postURL == "" {
			return titleText
		}
		return titleText + "\n\n" + postURL
	}
	if postURL == "" {
		return titleText + "\n\n" + bodyPreview
	}
	return titleText + "\n\n" + bodyPreview + "\n\n" + postURL
}
