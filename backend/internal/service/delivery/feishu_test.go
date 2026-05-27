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
			json.Unmarshal(body, &receivedBody)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"code":0}`))
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
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"code":9400}`))
		}))
		defer server.Close()

		client := NewFeishuClient(5 * time.Second)
		err := client.SendText(context.Background(), server.URL, "test")
		if err == nil {
			t.Fatal("expected error for non-2xx status")
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
