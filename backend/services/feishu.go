package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type FeishuClient struct {
	httpClient *http.Client
}

func NewFeishuClient(timeout time.Duration) *FeishuClient {
	return &FeishuClient{
		httpClient: &http.Client{Timeout: timeout},
	}
}

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
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("feishu webhook status=%d body=%s", resp.StatusCode, string(b))
	}

	return nil
}
