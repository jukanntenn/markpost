package markpost

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
)

// Delivery endpoints: the caller's channels and delivery pipeline history.

// ListChannels returns the caller's delivery channels ({items: [...]}).
func (c *Client) ListChannels(ctx context.Context) (json.RawMessage, error) {
	return c.request(ctx, http.MethodGet, apiPrefix+"/delivery/channels", nil, nil, true)
}

// CreateChannelInput mirrors the create-channel body. Configuration is raw
// JSON because its shape depends on the channel kind (e.g. Feishu needs
// {"webhook_url"}, custom webhook needs {"url"}).
type CreateChannelInput struct {
	Kind          string          `json:"kind"`
	Name          string          `json:"name"`
	Configuration json.RawMessage `json:"configuration"`
	Keywords      string          `json:"keywords"`
}

// CreateChannel creates a delivery channel (201, {channel: {...}}).
func (c *Client) CreateChannel(ctx context.Context, in CreateChannelInput) (json.RawMessage, error) {
	return c.request(ctx, http.MethodPost, apiPrefix+"/delivery/channels", nil, in, true)
}

// UpdateChannelInput mirrors the update-channel body: nil fields are omitted
// (PATCH partial update).
type UpdateChannelInput struct {
	Kind          *string          `json:"kind,omitempty"`
	Name          *string          `json:"name,omitempty"`
	Configuration *json.RawMessage `json:"configuration,omitempty"`
	Keywords      *string          `json:"keywords,omitempty"`
	Enabled       *bool            `json:"enabled,omitempty"`
}

// UpdateChannel partially updates one of the caller's channels (route is
// PATCH; the backend returns the updated {channel: {...}}).
func (c *Client) UpdateChannel(ctx context.Context, id int, in UpdateChannelInput) (json.RawMessage, error) {
	return c.request(ctx, http.MethodPatch, apiPrefix+"/delivery/channels/"+url.PathEscape(strconv.Itoa(id)), nil, in, true)
}

// DeleteChannel removes one of the caller's channels (204).
func (c *Client) DeleteChannel(ctx context.Context, id int) error {
	_, err := c.request(ctx, http.MethodDelete, apiPrefix+"/delivery/channels/"+url.PathEscape(strconv.Itoa(id)), nil, nil, true)
	return err
}

// TestChannel sends a test message to one of the caller's channels.
func (c *Client) TestChannel(ctx context.Context, id int) (json.RawMessage, error) {
	return c.request(ctx, http.MethodPost, apiPrefix+"/delivery/channels/"+url.PathEscape(strconv.Itoa(id))+"/test", nil, nil, true)
}

// ListDeliveryHistory returns the caller's delivery history (paginated,
// optional channel_id / status filters where status is delivered|failed|expired).
func (c *Client) ListDeliveryHistory(ctx context.Context, channelID int, status string, page, limit int) (json.RawMessage, error) {
	q := paginationQuery(page, limit)
	if channelID > 0 {
		q.Set("channel_id", strconv.Itoa(channelID))
	}
	if status != "" {
		q.Set("status", status)
	}
	return c.request(ctx, http.MethodGet, apiPrefix+"/delivery/history", q, nil, true)
}

// ListLatestDeliveries returns the most recent delivery per channel.
func (c *Client) ListLatestDeliveries(ctx context.Context) (json.RawMessage, error) {
	return c.request(ctx, http.MethodGet, apiPrefix+"/delivery/latest", nil, nil, true)
}

// GetDeliveryStats returns today counters plus the per-day trend (days ≤ 365).
func (c *Client) GetDeliveryStats(ctx context.Context, days int) (json.RawMessage, error) {
	q := url.Values{}
	if days > 0 {
		q.Set("days", strconv.Itoa(days))
	}
	return c.request(ctx, http.MethodGet, apiPrefix+"/delivery/stats", q, nil, true)
}

// ListPendingDeliveries returns the caller's in-flight delivery attempts.
func (c *Client) ListPendingDeliveries(ctx context.Context) (json.RawMessage, error) {
	return c.request(ctx, http.MethodGet, apiPrefix+"/delivery/pending", nil, nil, true)
}
