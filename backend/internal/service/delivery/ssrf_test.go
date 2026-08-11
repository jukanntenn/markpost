package delivery

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"markpost/internal/service"
)

// I.3 SSRF 防护：拒绝私有/保留地址的 webhook_url。
func TestValidateWebhookURLSSRF(t *testing.T) {
	forbidden := []string{
		"http://127.0.0.1/webhook",
		"http://10.0.0.1/webhook",
		"http://10.255.255.255/webhook",
		"http://172.16.0.1/webhook",
		"http://172.31.255.255/webhook",
		"http://192.168.1.1/webhook",
		"http://169.254.169.254/latest/meta-data", // 云元数据
		"http://[::1]/webhook",
		"http://[fc00::1]/webhook",
		"http://[fd00::1]/webhook",
		"http://localhost/webhook",
		"ftp://example.com/webhook",
		"file:///etc/passwd",
		"http://",      // 无 host
		"http:///path", // 无 host
		"not a url at all",
	}
	for _, u := range forbidden {
		if err := validateWebhookURLSSRF(context.Background(), u); err == nil {
			t.Errorf("expected %q to be rejected", u)
		}
	}

	allowed := []string{
		"http://example.com/webhook",
		"https://open.feishu.cn/open-apis/bot/v2/hook/abc",
		"http://8.8.8.8/webhook",
		"http://[2001:4860:4860::8888]/webhook",
		"https://example.com:8443/path?q=1",
	}
	for _, u := range allowed {
		if err := validateWebhookURLSSRF(context.Background(), u); err != nil {
			t.Errorf("expected %q to be allowed, got %v", u, err)
		}
	}
}

// 通过 Create 集成验证：内网 webhook 返回 webhook_url_forbidden（422）。
func TestCreateChannel_RejectsInternalWebhook(t *testing.T) {
	svc, _, db := setupDeliveryService(t)
	ctx := context.Background()

	config, _ := json.Marshal(map[string]string{
		"webhook_url": "http://169.254.169.254/latest/meta-data",
	})
	_, err := svc.Create(ctx, createTestUser(t, db, 0), UpdateChannelParams{
		Kind:          "feishu",
		Name:          "evil",
		Configuration: config,
	})
	if err == nil {
		t.Fatal("expected error for internal webhook URL")
	}
	se, ok := service.AsError(err)
	if !ok {
		t.Fatalf("expected service error, got %v", err)
	}
	if se.Code != ErrWebhookURLForbidden {
		t.Errorf("expected code %q, got %q", ErrWebhookURLForbidden.Value, se.Code.Value)
	}
}

// 公网 webhook 正常创建。
func TestCreateChannel_AcceptsPublicWebhook(t *testing.T) {
	svc, _, db := setupDeliveryService(t)
	ctx := context.Background()

	config, _ := json.Marshal(map[string]string{
		"webhook_url": "https://open.feishu.cn/open-apis/bot/v2/hook/test",
	})
	userID := createTestUser(t, db, 1)
	ch, err := svc.Create(ctx, userID, UpdateChannelParams{
		Kind:          "feishu",
		Name:          "workgroup",
		Configuration: config,
	})
	if err != nil {
		t.Fatalf("expected public webhook accepted, got %v", err)
	}
	if ch == nil || ch.Name != "workgroup" {
		t.Fatalf("unexpected channel %+v", ch)
	}
}

// 更新路径同样校验 SSRF。
func TestUpdateChannel_RejectsInternalWebhook(t *testing.T) {
	svc, _, db := setupDeliveryService(t)
	ctx := context.Background()

	good, _ := json.Marshal(map[string]string{
		"webhook_url": "https://open.feishu.cn/open-apis/bot/v2/hook/ok",
	})
	userID := createTestUser(t, db, 2)
	ch, err := svc.Create(ctx, userID, UpdateChannelParams{
		Kind:          "feishu",
		Name:          "ok",
		Configuration: good,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	evil, _ := json.Marshal(map[string]string{
		"webhook_url": "http://192.168.0.10/steal",
	})
	_, err = svc.Update(ctx, userID, ch.ID, UpdateChannelParams{Configuration: evil})
	if err == nil {
		t.Fatal("expected SSRF rejection on update")
	}
	se, ok := service.AsError(err)
	if !ok || se.Code != ErrWebhookURLForbidden {
		t.Fatalf("expected webhook_url_forbidden, got %v", err)
	}
}

// 关键词校验仍在（错误表达式被拒）。
func TestCreateChannel_RejectsInvalidKeywords(t *testing.T) {
	svc, _, db := setupDeliveryService(t)
	config, _ := json.Marshal(map[string]string{
		"webhook_url": "https://open.feishu.cn/open-apis/bot/v2/hook/ok",
	})
	kw := `"unterminated`
	_, err := svc.Create(context.Background(), createTestUser(t, db, 3), UpdateChannelParams{
		Kind:          "feishu",
		Name:          "kw",
		Configuration: config,
		Keywords:      &kw,
	})
	if err == nil {
		t.Fatal("expected invalid keywords expression error")
	}
	if !strings.Contains(err.Error(), "keywords") {
		t.Errorf("expected keywords-related error, got %v", err)
	}
}
