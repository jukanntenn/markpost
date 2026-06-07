// Package delivery provides delivery channel business logic and services.
package delivery

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"text/template"
	"time"

	"markpost/internal/domain/delivery"
)

// FeishuClient sends messages to Feishu webhook URLs.
type FeishuClient struct {
	httpClient *http.Client
}

// NewFeishuClient creates a FeishuClient with the given request timeout.
func NewFeishuClient(timeout time.Duration) *FeishuClient {
	return &FeishuClient{
		httpClient: &http.Client{Timeout: timeout},
	}
}

type feishuCardPayload struct {
	Schema string `json:"schema"`
	Config struct {
		UpdateMulti bool `json:"update_multi"`
	} `json:"config"`
	CardLink *feishuCardLink  `json:"card_link,omitempty"`
	Header   feishuCardHeader `json:"header"`
	Body     feishuCardBody   `json:"body"`
}

type feishuCardLink struct {
	URL string `json:"url"`
}

type feishuCardHeader struct {
	Title feishuCardTitle `json:"title"`
}

type feishuCardTitle struct {
	Tag     string `json:"tag"`
	Content string `json:"content"`
}

type feishuCardBody struct {
	Elements []feishuCardElement `json:"elements"`
}

type feishuCardElement struct {
	Tag        string              `json:"tag"`
	Content    string              `json:"content,omitempty"`
	Text       *feishuCardTextElem `json:"text,omitempty"`
	URL        string              `json:"url,omitempty"`
	ButtonType string              `json:"type,omitempty"`
}

type feishuCardTextElem struct {
	Tag     string `json:"tag"`
	Content string `json:"content"`
}

// SendText posts a text message to the given Feishu webhook URL.
func (c *FeishuClient) SendText(ctx context.Context, webhookURL string, text string) error {
	payload := map[string]any{
		"msg_type": "text",
		"content": map[string]any{
			"text": text,
		},
	}
	return c.sendRequest(ctx, webhookURL, payload)
}

// CardDeliveryParams holds the parameters for sending a Feishu card message.
type CardDeliveryParams struct {
	WebhookURL  string
	CardLinkURL string
	PostURL     string
	PostTitle   string
	BodyPreview string
	PostQID     string
}

// SendCard posts an interactive card message to the given Feishu webhook URL.
func (c *FeishuClient) SendCard(ctx context.Context, params CardDeliveryParams) error {
	resolvedCardLinkURL := resolveCardLinkURL(params)
	showFooter := resolvedCardLinkURL != params.PostURL && params.PostURL != ""

	elements := make([]feishuCardElement, 0)

	if params.BodyPreview != "" {
		elements = append(elements, feishuCardElement{
			Tag:     "markdown",
			Content: params.BodyPreview,
		})
	}

	if showFooter {
		elements = append(elements, feishuCardElement{
			Tag: "button",
			Text: &feishuCardTextElem{
				Tag:     "plain_text",
				Content: "View Post",
			},
			URL:        params.PostURL,
			ButtonType: "primary",
		})
	}

	card := feishuCardPayload{
		Schema: "2.0",
		Config: struct {
			UpdateMulti bool `json:"update_multi"`
		}{UpdateMulti: true},
		Header: feishuCardHeader{
			Title: feishuCardTitle{
				Tag:     "plain_text",
				Content: params.PostTitle,
			},
		},
		Body: feishuCardBody{
			Elements: elements,
		},
	}

	if resolvedCardLinkURL != "" {
		card.CardLink = &feishuCardLink{URL: resolvedCardLinkURL}
	}

	payload := map[string]any{
		"msg_type": "interactive",
		"card":     card,
	}
	return c.sendRequest(ctx, params.WebhookURL, payload)
}

func resolveCardLinkURL(params CardDeliveryParams) string {
	if strings.TrimSpace(params.CardLinkURL) == "" {
		return params.PostURL
	}
	tmpl, err := template.New("card_link").Parse(params.CardLinkURL)
	if err != nil {
		return params.PostURL
	}
	var buf bytes.Buffer
	data := map[string]string{"QID": params.PostQID}
	if err := tmpl.Execute(&buf, data); err != nil {
		return params.PostURL
	}
	return buf.String()
}

func (c *FeishuClient) sendRequest(ctx context.Context, webhookURL string, payload any) error {
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

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return fmt.Errorf("feishu read response: %w", err)
	}

	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	if json.Unmarshal(respBody, &result) == nil && result.Code != 0 {
		return fmt.Errorf("feishu api code=%d msg=%s", result.Code, result.Msg)
	}

	return nil
}

func feishuWebhookFromChannel(ch delivery.Channel) string {
	return ch.Configuration.Feishu().WebhookURL
}

func feishuCardLinkURLFromChannel(ch delivery.Channel) string {
	return ch.Configuration.Feishu().CardLinkURL
}
