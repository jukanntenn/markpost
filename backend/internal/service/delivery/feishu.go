// Package delivery provides delivery channel business logic and services.
package delivery

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// FeishuClient sends text messages to Feishu webhook URLs.
type FeishuClient struct {
	httpClient *http.Client
}

// NewFeishuClient creates a FeishuClient with the given request timeout.
func NewFeishuClient(timeout time.Duration) *FeishuClient {
	return &FeishuClient{
		httpClient: &http.Client{Timeout: timeout},
	}
}

// SendText posts a text message to the given Feishu webhook URL.
func (c *FeishuClient) SendText(ctx context.Context, webhookURL string, text string) error {
	payload := map[string]any{
		"msg_type": "text",
		"content": map[string]any{
			"text": text,
		},
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("feishu marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("feishu create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("feishu request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("feishu webhook status=%d body=%s", resp.StatusCode, string(b))
	}

	return nil
}
