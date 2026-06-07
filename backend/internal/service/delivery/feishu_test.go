package delivery

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestFeishuClient_SendText(t *testing.T) {
	t.Run("sends message successfully", func(t *testing.T) {
		var receivedBody map[string]any
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(body, &receivedBody)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"code":0}`))
		}))
		defer server.Close()

		client := NewFeishuClient(5 * time.Second)
		err := client.SendText(context.Background(), server.URL, "Hello World")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if receivedBody["msg_type"] != "text" {
			t.Errorf("msg_type = %v, want %q", receivedBody["msg_type"], "text")
		}
		content, ok := receivedBody["content"].(map[string]any)
		if !ok {
			t.Fatal("expected content to be a map")
		}
		if content["text"] != "Hello World" {
			t.Errorf("text = %v, want %q", content["text"], "Hello World")
		}
	})

	t.Run("returns error for non-2xx status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"code":9400}`))
		}))
		defer server.Close()

		client := NewFeishuClient(5 * time.Second)
		err := client.SendText(context.Background(), server.URL, "test")
		if err == nil {
			t.Fatal("expected error for non-2xx status")
		}
	})

	t.Run("returns error for non-zero API code", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"code":11246,"msg":"card json is invalid"}`))
		}))
		defer server.Close()

		client := NewFeishuClient(5 * time.Second)
		err := client.SendText(context.Background(), server.URL, "test")
		if err == nil {
			t.Fatal("expected error for non-zero API code")
		}
	})

	t.Run("returns error for unreachable URL", func(t *testing.T) {
		client := NewFeishuClient(100 * time.Millisecond)
		err := client.SendText(context.Background(), "http://192.0.2.1:12345/webhook", "test")
		if err == nil {
			t.Fatal("expected error for unreachable URL")
		}
	})

	t.Run("sets correct content type header", func(t *testing.T) {
		var contentType string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			contentType = r.Header.Get("Content-Type")
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := NewFeishuClient(5 * time.Second)
		_ = client.SendText(context.Background(), server.URL, "test")

		if contentType != "application/json; charset=utf-8" {
			t.Errorf("content-type = %q, want %q", contentType, "application/json; charset=utf-8")
		}
	})
}

func TestFeishuClient_SendCard(t *testing.T) {
	t.Run("sends card with post URL as card_link", func(t *testing.T) {
		var receivedBody map[string]any
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(body, &receivedBody)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"code":0}`))
		}))
		defer server.Close()

		client := NewFeishuClient(5 * time.Second)
		err := client.SendCard(context.Background(), CardDeliveryParams{
			WebhookURL:  server.URL,
			PostURL:     "https://example.com/p-abc",
			PostTitle:   "Test Title",
			BodyPreview: "Some preview text",
			PostQID:     "p-abc",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if receivedBody["msg_type"] != "interactive" {
			t.Errorf("msg_type = %v, want %q", receivedBody["msg_type"], "interactive")
		}

		card, ok := receivedBody["card"].(map[string]any)
		if !ok {
			t.Fatal("expected card to be a map")
		}
		if card["schema"] != "2.0" {
			t.Errorf("schema = %v, want %q", card["schema"], "2.0")
		}
		if card["card_link"] == nil {
			t.Error("expected card_link to be set")
		}
	})

	t.Run("uses custom card_link_url with template variable", func(t *testing.T) {
		var receivedBody map[string]any
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(body, &receivedBody)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"code":0}`))
		}))
		defer server.Close()

		client := NewFeishuClient(5 * time.Second)
		err := client.SendCard(context.Background(), CardDeliveryParams{
			WebhookURL:  server.URL,
			CardLinkURL: "https://custom.example.com/post/{{.QID}}",
			PostURL:     "https://example.com/p-abc",
			PostTitle:   "Test Title",
			BodyPreview: "Preview",
			PostQID:     "p-abc",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		card := receivedBody["card"].(map[string]any)
		cardLink := card["card_link"].(map[string]any)
		if cardLink["url"] != "https://custom.example.com/post/p-abc" {
			t.Errorf("card_link url = %v, want %q", cardLink["url"], "https://custom.example.com/post/p-abc")
		}
	})

	t.Run("shows footer when card_link_url differs from post URL", func(t *testing.T) {
		var receivedBody map[string]any
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(body, &receivedBody)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"code":0}`))
		}))
		defer server.Close()

		client := NewFeishuClient(5 * time.Second)
		err := client.SendCard(context.Background(), CardDeliveryParams{
			WebhookURL:  server.URL,
			CardLinkURL: "https://custom.example.com/{{.QID}}",
			PostURL:     "https://example.com/p-abc",
			PostTitle:   "Test Title",
			BodyPreview: "Preview",
			PostQID:     "p-abc",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		card := receivedBody["card"].(map[string]any)
		body := card["body"].(map[string]any)
		elements := body["elements"].([]any)

		hasButton := false
		for _, elem := range elements {
			el := elem.(map[string]any)
			if el["tag"] == "button" {
				hasButton = true
				if el["url"] != "https://example.com/p-abc" {
					t.Errorf("button url = %v, want %q", el["url"], "https://example.com/p-abc")
				}
			}
		}
		if !hasButton {
			t.Error("expected button element in card body when card_link_url differs from post URL")
		}
	})

	t.Run("hides footer when card_link_url equals post URL", func(t *testing.T) {
		var receivedBody map[string]any
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(body, &receivedBody)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"code":0}`))
		}))
		defer server.Close()

		client := NewFeishuClient(5 * time.Second)
		err := client.SendCard(context.Background(), CardDeliveryParams{
			WebhookURL:  server.URL,
			CardLinkURL: "https://example.com/p-abc",
			PostURL:     "https://example.com/p-abc",
			PostTitle:   "Test Title",
			BodyPreview: "Preview",
			PostQID:     "p-abc",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		card := receivedBody["card"].(map[string]any)
		body := card["body"].(map[string]any)
		elements := body["elements"].([]any)

		for _, elem := range elements {
			el := elem.(map[string]any)
			if el["tag"] == "button" {
				t.Error("expected no button element when card_link_url equals post URL")
			}
		}
	})

	t.Run("sends card without body preview", func(t *testing.T) {
		var receivedBody map[string]any
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(body, &receivedBody)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"code":0}`))
		}))
		defer server.Close()

		client := NewFeishuClient(5 * time.Second)
		err := client.SendCard(context.Background(), CardDeliveryParams{
			WebhookURL: server.URL,
			PostURL:    "https://example.com/p-abc",
			PostTitle:  "Test Title",
			PostQID:    "p-abc",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		card := receivedBody["card"].(map[string]any)
		body := card["body"].(map[string]any)
		elements := body["elements"].([]any)
		if len(elements) != 0 {
			t.Errorf("expected 0 elements, got %d", len(elements))
		}
	})
}

func TestResolveCardLinkURL(t *testing.T) {
	t.Run("returns post URL when card_link_url is empty", func(t *testing.T) {
		result := resolveCardLinkURL(CardDeliveryParams{
			CardLinkURL: "",
			PostURL:     "https://example.com/p-abc",
			PostQID:     "p-abc",
		})
		if result != "https://example.com/p-abc" {
			t.Errorf("got %q, want %q", result, "https://example.com/p-abc")
		}
	})

	t.Run("resolves template variable", func(t *testing.T) {
		result := resolveCardLinkURL(CardDeliveryParams{
			CardLinkURL: "https://custom.com/{{.QID}}",
			PostURL:     "https://example.com/p-abc",
			PostQID:     "p-abc",
		})
		if result != "https://custom.com/p-abc" {
			t.Errorf("got %q, want %q", result, "https://custom.com/p-abc")
		}
	})

	t.Run("falls back to post URL on invalid template", func(t *testing.T) {
		result := resolveCardLinkURL(CardDeliveryParams{
			CardLinkURL: "{{.Invalid",
			PostURL:     "https://example.com/p-abc",
			PostQID:     "p-abc",
		})
		if result != "https://example.com/p-abc" {
			t.Errorf("got %q, want %q", result, "https://example.com/p-abc")
		}
	})
}
