package tools

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostsToolset(t *testing.T) {
	t.Run("create_post composes id and url", func(t *testing.T) {
		h := newHarness(t, false, []string{"posts"},
			apiRoute{Method: "GET", Path: "/api/v1/post-key", Body: `{"post_key":"pk9","created_at":"2026-09-03T00:00:00Z"}`},
			apiRoute{Method: "POST", Path: "/pk9", Status: 201, Body: `{"id":"ab12"}`},
		)
		_, text := h.call(t, "create_post", map[string]any{"title": "Hi", "body": "# Hi"})
		requireJSONEq(t, text, `{"id":"ab12","url":"`+h.fake.srv.URL+`/ab12"}`)

		assert.Equal(t, []string{
			"POST /api/v1/auth/login",
			"GET /api/v1/post-key",
			"POST /pk9",
		}, h.fake.recorded())
	})

	t.Run("list_posts forwards search and pagination", func(t *testing.T) {
		h := newHarness(t, false, []string{"posts"},
			apiRoute{Method: "GET", Path: "/api/v1/posts", Body: `{"items":[{"id":1,"qid":"ab12","title":"Hi","created_at":"2026-09-03T00:00:00Z"}],"total":1,"page":2,"limit":10,"total_pages":1}`},
		)
		_, text := h.call(t, "list_posts", map[string]any{"search": "hi", "page": 2, "limit": 10})
		requireJSONEq(t, text, `{"items":[{"id":1,"qid":"ab12","title":"Hi","created_at":"2026-09-03T00:00:00Z"}],"total":1,"page":2,"limit":10,"total_pages":1}`)
		assert.Contains(t, h.fake.recorded(), "GET /api/v1/posts?limit=10&page=2&search=hi")
	})

	t.Run("get_post returns markdown", func(t *testing.T) {
		h := newHarness(t, false, []string{"posts"},
			apiRoute{Method: "GET", Path: "/ab12", Raw: true, Body: "# Hi\n\nbody"},
		)
		_, text := h.call(t, "get_post", map[string]any{"qid": "ab12"})
		assert.Equal(t, "# Hi\n\nbody", text)
		assert.Contains(t, h.fake.recorded(), "GET /ab12?format=raw")
	})

	t.Run("delete_post reports deletion and backend errors", func(t *testing.T) {
		h := newHarness(t, false, []string{"posts"},
			apiRoute{Method: "DELETE", Path: "/api/v1/posts/ab12", Status: 204},
		)
		res, text := h.call(t, "delete_post", map[string]any{"qid": "ab12"})
		assert.False(t, res.IsError)
		requireJSONEq(t, text, `{"deleted": true}`)

		h2 := newHarness(t, false, []string{"posts"},
			apiRoute{Method: "DELETE", Path: "/api/v1/posts/gone", Status: 404, Body: `{"error":{"code":"not_found","message":"post not found"}}`},
		)
		res2, text2 := h2.call(t, "delete_post", map[string]any{"qid": "gone"})
		assert.True(t, res2.IsError)
		assert.Contains(t, text2, "post not found")
		assert.Contains(t, text2, "HTTP 404")
	})

	t.Run("read-only mode drops write tools", func(t *testing.T) {
		h := newHarness(t, true, []string{"posts"})
		res, err := h.session.ListTools(t.Context(), nil)
		require.NoError(t, err)
		var names []string
		for _, tool := range res.Tools {
			names = append(names, tool.Name)
		}
		assert.Equal(t, []string{"get_post", "list_posts"}, names)
	})
}

func TestDeliveryToolset(t *testing.T) {
	t.Run("list_channels forwards the envelope", func(t *testing.T) {
		h := newHarness(t, false, []string{"delivery"},
			apiRoute{Method: "GET", Path: "/api/v1/delivery/channels", Body: `{"items":[{"id":3,"kind":"feishu","name":"ops","enabled":true}]}`},
		)
		_, text := h.call(t, "list_channels", map[string]any{})
		requireJSONEq(t, text, `{"items":[{"id":3,"kind":"feishu","name":"ops","enabled":true}]}`)
	})

	t.Run("create_channel posts kind-specific configuration", func(t *testing.T) {
		h := newHarness(t, false, []string{"delivery"},
			apiRoute{Method: "POST", Path: "/api/v1/delivery/channels", Status: 201, Body: `{"channel":{"id":9,"kind":"feishu","name":"n"}}`},
		)
		_, text := h.call(t, "create_channel", map[string]any{
			"kind": "feishu", "name": "n", "configuration": `{"webhook_url":"https://example.com/x"}`, "keywords": "ops,alert",
		})
		requireJSONEq(t, text, `{"channel":{"id":9,"kind":"feishu","name":"n"}}`)
		assert.Contains(t, h.fake.recorded(), "POST /api/v1/delivery/channels")
	})

	t.Run("update_channel sends only provided fields", func(t *testing.T) {
		h := newHarness(t, false, []string{"delivery"},
			apiRoute{Method: "PATCH", Path: "/api/v1/delivery/channels/9", Body: `{"channel":{"id":9,"enabled":false}}`},
		)
		_, text := h.call(t, "update_channel", map[string]any{"id": 9, "enabled": false})
		requireJSONEq(t, text, `{"channel":{"id":9,"enabled":false}}`)
		assert.Contains(t, h.fake.recorded(), "PATCH /api/v1/delivery/channels/9")
	})

	t.Run("delete_channel and test_channel", func(t *testing.T) {
		h := newHarness(t, false, []string{"delivery"},
			apiRoute{Method: "DELETE", Path: "/api/v1/delivery/channels/9", Status: 204},
			apiRoute{Method: "POST", Path: "/api/v1/delivery/channels/9/test", Body: `{"message":"test message sent"}`},
		)
		_, text := h.call(t, "delete_channel", map[string]any{"id": 9})
		requireJSONEq(t, text, `{"deleted": true}`)
		_, text2 := h.call(t, "test_channel", map[string]any{"id": 9})
		requireJSONEq(t, text2, `{"message":"test message sent"}`)
	})

	t.Run("history, latest, stats, pending", func(t *testing.T) {
		h := newHarness(t, false, []string{"delivery"},
			apiRoute{Method: "GET", Path: "/api/v1/delivery/history", Body: `{"items":[],"total":0,"page":1,"limit":20,"total_pages":0}`},
			apiRoute{Method: "GET", Path: "/api/v1/delivery/latest", Body: `{"items":[]}`},
			apiRoute{Method: "GET", Path: "/api/v1/delivery/stats", Body: `{"today":{"delivered":2},"trend":[]}`},
			apiRoute{Method: "GET", Path: "/api/v1/delivery/pending", Body: `{"items":[]}`},
		)
		_, text := h.call(t, "list_delivery_history", map[string]any{"channel_id": 9, "status": "failed", "page": 1})
		requireJSONEq(t, text, `{"items":[],"total":0,"page":1,"limit":20,"total_pages":0}`)
		assert.Contains(t, h.fake.recorded(), "GET /api/v1/delivery/history?channel_id=9&page=1&status=failed")

		_, text = h.call(t, "list_latest_deliveries", map[string]any{})
		requireJSONEq(t, text, `{"items":[]}`)

		_, text = h.call(t, "get_delivery_stats", map[string]any{"days": 30})
		requireJSONEq(t, text, `{"today":{"delivered":2},"trend":[]}`)
		assert.Contains(t, h.fake.recorded(), "GET /api/v1/delivery/stats?days=30")

		_, text = h.call(t, "list_pending_deliveries", map[string]any{})
		requireJSONEq(t, text, `{"items":[]}`)
	})
}

func TestAccountToolset(t *testing.T) {
	routes := []apiRoute{
		{Method: "GET", Path: "/api/v1/me/retention", Body: `{"posts_days":7,"history_days":14}`},
		{Method: "GET", Path: "/api/v1/auth/sessions", Body: `{"sessions":[{"id":5}]}`},
		{Method: "DELETE", Path: "/api/v1/auth/sessions/5", Body: `{"revoked":true}`},
		{Method: "DELETE", Path: "/api/v1/auth/sessions", Body: `{"revoked":true}`},
		{Method: "POST", Path: "/api/v1/post-key/rotate", Body: `{"post_key":"pk-new"}`},
		{Method: "POST", Path: "/api/v1/auth/change-password", Body: `{"token":"t2","refresh_token":"r2","expires_in":86400}`},
	}

	t.Run("retention and sessions", func(t *testing.T) {
		h := newHarness(t, false, []string{"account"}, routes...)
		_, text := h.call(t, "get_my_retention", map[string]any{})
		requireJSONEq(t, text, `{"posts_days":7,"history_days":14}`)

		_, text = h.call(t, "list_my_sessions", map[string]any{})
		requireJSONEq(t, text, `{"sessions":[{"id":5}]}`)

		_, text = h.call(t, "revoke_my_session", map[string]any{"token_id": 5})
		requireJSONEq(t, text, `{"revoked":true}`)

		_, text = h.call(t, "revoke_my_other_sessions", map[string]any{})
		requireJSONEq(t, text, `{"revoked":true}`)
	})

	t.Run("rotate and change password", func(t *testing.T) {
		h := newHarness(t, false, []string{"account"}, routes...)
		_, text := h.call(t, "rotate_post_key", map[string]any{})
		requireJSONEq(t, text, `{"post_key":"pk-new"}`)

		_, text = h.call(t, "change_my_password", map[string]any{"current_password": "old", "new_password": "newpass123"})
		requireJSONEq(t, text, `{"token":"t2","refresh_token":"r2","expires_in":86400}`)
	})
}

func TestAdminToolset(t *testing.T) {
	t.Run("default off", func(t *testing.T) {
		h := newHarness(t, false, DefaultToolsetNames())
		res, err := h.session.ListTools(t.Context(), nil)
		require.NoError(t, err)
		for _, tool := range res.Tools {
			assert.NotContains(t, tool.Name, "admin_", "admin tools must not be registered by default")
		}
	})

	t.Run("read surface", func(t *testing.T) {
		h := newHarness(t, false, []string{"admin"},
			apiRoute{Method: "GET", Path: "/api/v1/admin/users", Body: `{"items":[{"id":2,"username":"bob"}],"total":1,"page":1,"limit":20,"total_pages":1}`},
			apiRoute{Method: "GET", Path: "/api/v1/admin/users/2", Body: `{"id":2,"username":"bob","post_key":"pk-b"}`},
			apiRoute{Method: "GET", Path: "/api/v1/admin/posts", Body: `{"items":[],"total":0,"page":1,"limit":20,"total_pages":0}`},
			apiRoute{Method: "GET", Path: "/api/v1/admin/delivery/channels", Body: `{"items":[],"total":0,"page":1,"limit":20,"total_pages":0}`},
			apiRoute{Method: "GET", Path: "/api/v1/admin/delivery/history", Body: `{"items":[],"total":0,"page":1,"limit":20,"total_pages":0}`},
			apiRoute{Method: "GET", Path: "/api/v1/admin/delivery/stats", Body: `{"trend":[]}`},
			apiRoute{Method: "GET", Path: "/api/v1/admin/locked-channels", Body: `{"items":[]}`},
			apiRoute{Method: "GET", Path: "/api/v1/admin/audit-logs", Body: `{"items":[],"facets":{},"total":0,"page":1,"limit":20,"total_pages":0}`},
			apiRoute{Method: "GET", Path: "/api/v1/admin/users/2/sessions", Body: `{"sessions":[]}`},
			apiRoute{Method: "GET", Path: "/api/v1/admin/stats", Body: `{"counts":{"users":2,"posts":5}}`},
			apiRoute{Method: "GET", Path: "/api/v1/admin/settings", Body: `{"items":[]}`},
			apiRoute{Method: "GET", Path: "/api/v1/admin/retention/defaults", Body: `{"posts_days":7,"history_days":30}`},
			apiRoute{Method: "POST", Path: "/api/v1/admin/retention/impact", Body: `{"users":1,"posts":3,"history":10}`},
		)

		checks := []struct {
			tool string
			args map[string]any
			want string
		}{
			{"admin_list_users", map[string]any{"search": "bob"}, `{"items":[{"id":2,"username":"bob"}],"total":1,"page":1,"limit":20,"total_pages":1}`},
			{"admin_get_user", map[string]any{"user_id": 2}, `{"id":2,"username":"bob","post_key":"pk-b"}`},
			{"admin_list_posts", map[string]any{}, `{"items":[],"total":0,"page":1,"limit":20,"total_pages":0}`},
			{"admin_list_channels", map[string]any{}, `{"items":[],"total":0,"page":1,"limit":20,"total_pages":0}`},
			{"admin_list_delivery_history", map[string]any{"status": "failed"}, `{"items":[],"total":0,"page":1,"limit":20,"total_pages":0}`},
			{"admin_get_delivery_stats", map[string]any{"days": 14}, `{"trend":[]}`},
			{"admin_list_locked_channels", map[string]any{}, `{"items":[]}`},
			{"admin_list_audit_logs", map[string]any{"action": "user.create"}, `{"items":[],"facets":{},"total":0,"page":1,"limit":20,"total_pages":0}`},
			{"admin_list_user_sessions", map[string]any{"user_id": 2}, `{"sessions":[]}`},
			{"admin_get_stats", map[string]any{}, `{"counts":{"users":2,"posts":5}}`},
			{"admin_get_settings", map[string]any{}, `{"items":[]}`},
			{"admin_get_retention_defaults", map[string]any{}, `{"posts_days":7,"history_days":30}`},
			{"admin_retention_impact", map[string]any{"user_ids": []int{2}}, `{"users":1,"posts":3,"history":10}`},
		}
		for _, ck := range checks {
			t.Run(ck.tool, func(t *testing.T) {
				res, text := h.call(t, ck.tool, ck.args)
				assert.False(t, res.IsError)
				requireJSONEq(t, text, ck.want)
			})
		}
		assert.Contains(t, h.fake.recorded(), "GET /api/v1/admin/users?search=bob")
		assert.Contains(t, h.fake.recorded(), "GET /api/v1/admin/delivery/stats?days=14")
		assert.Contains(t, h.fake.recorded(), "GET /api/v1/admin/audit-logs?action=user.create")
	})

	t.Run("write surface", func(t *testing.T) {
		h := newHarness(t, false, []string{"admin"},
			apiRoute{Method: "POST", Path: "/api/v1/admin/users", Status: 201, Body: `{"id":7,"username":"carol"}`},
			apiRoute{Method: "DELETE", Path: "/api/v1/admin/users/7", Body: `{"deleted":4}`},
			apiRoute{Method: "PATCH", Path: "/api/v1/admin/users/7/role", Body: `{"id":7,"role":"admin"}`},
			apiRoute{Method: "POST", Path: "/api/v1/admin/users/7/password", Body: `{"password":"tmp-pass-123"}`},
			apiRoute{Method: "PATCH", Path: "/api/v1/admin/users/7/active", Body: `{"id":7,"is_active":false}`},
			apiRoute{Method: "PATCH", Path: "/api/v1/admin/users/7/vip", Body: `{"id":7,"vip":true}`},
			apiRoute{Method: "PATCH", Path: "/api/v1/admin/users/7/retention", Body: `{"id":7,"retention_days":30}`},
			apiRoute{Method: "POST", Path: "/api/v1/admin/users/retention/bulk", Body: `{"updated":3}`},
			apiRoute{Method: "DELETE", Path: "/api/v1/admin/posts/ab12", Status: 204},
			apiRoute{Method: "POST", Path: "/api/v1/admin/delivery/channels", Status: 201, Body: `{"id":11,"name":"svc"}`},
			apiRoute{Method: "PATCH", Path: "/api/v1/admin/delivery/channels/11/enabled", Body: `{"message":"Channel updated"}`},
			apiRoute{Method: "DELETE", Path: "/api/v1/admin/delivery/channels/11", Body: `{"deleted":1}`},
			apiRoute{Method: "DELETE", Path: "/api/v1/admin/users/7/sessions", Body: `{"revoked":true}`},
			apiRoute{Method: "DELETE", Path: "/api/v1/admin/sessions/55", Body: `{"revoked":true}`},
			apiRoute{Method: "PUT", Path: "/api/v1/admin/settings/vip", Body: `{"items":[]}`},
		)

		checks := []struct {
			tool string
			args map[string]any
			want string
		}{
			{"admin_create_user", map[string]any{"username": "carol", "password": "carols-pass"}, `{"id":7,"username":"carol"}`},
			{"admin_delete_user", map[string]any{"user_id": 7}, `{"deleted":4}`},
			{"admin_set_user_role", map[string]any{"user_id": 7, "role": "admin"}, `{"id":7,"role":"admin"}`},
			{"admin_reset_user_password", map[string]any{"user_id": 7}, `{"password":"tmp-pass-123"}`},
			{"admin_set_user_active", map[string]any{"user_id": 7, "value": false}, `{"id":7,"is_active":false}`},
			{"admin_set_user_vip", map[string]any{"user_id": 7, "value": true}, `{"id":7,"vip":true}`},
			{"admin_set_user_retention", map[string]any{"user_id": 7, "retention_days": 30}, `{"id":7,"retention_days":30}`},
			{"admin_bulk_set_retention", map[string]any{"user_ids": []int{1, 2, 3}, "retention_days": 0}, `{"updated":3}`},
			{"admin_delete_post", map[string]any{"qid": "ab12"}, `{"deleted": true}`},
			{"admin_create_channel", map[string]any{"user_id": 7, "kind": "feishu", "name": "svc", "configuration": `{"webhook_url":"https://x"}`}, `{"id":11,"name":"svc"}`},
			{"admin_set_channel_enabled", map[string]any{"channel_id": 11, "enabled": false}, `{"message":"Channel updated"}`},
			{"admin_delete_channel", map[string]any{"channel_id": 11}, `{"deleted":1}`},
			{"admin_revoke_user_sessions", map[string]any{"user_id": 7}, `{"revoked":true}`},
			{"admin_revoke_session", map[string]any{"token_id": 55}, `{"revoked":true}`},
			{"admin_set_setting", map[string]any{"key": "vip", "enabled": true}, `{"items":[]}`},
		}
		for _, ck := range checks {
			t.Run(ck.tool, func(t *testing.T) {
				res, text := h.call(t, ck.tool, ck.args)
				assert.False(t, res.IsError)
				requireJSONEq(t, text, ck.want)
			})
		}
	})

	t.Run("403 surfaces as tool error", func(t *testing.T) {
		h := newHarness(t, false, []string{"admin"},
			apiRoute{Method: "GET", Path: "/api/v1/admin/stats", Status: 403, Body: `{"error":{"code":"forbidden","message":"admin role required"}}`},
		)
		res, text := h.call(t, "admin_get_stats", map[string]any{})
		assert.True(t, res.IsError)
		assert.Contains(t, text, "admin role required")
		assert.Contains(t, text, "HTTP 403")
	})

	t.Run("read-only mode keeps admin reads", func(t *testing.T) {
		h := newHarness(t, true, []string{"admin"})
		res, err := h.session.ListTools(t.Context(), nil)
		require.NoError(t, err)
		for _, tool := range res.Tools {
			assert.NotContains(t, []string{
				"admin_create_user", "admin_delete_user", "admin_set_user_role", "admin_reset_user_password",
				"admin_set_user_active", "admin_set_user_vip", "admin_set_user_retention", "admin_bulk_set_retention",
				"admin_delete_post", "admin_create_channel", "admin_set_channel_enabled", "admin_delete_channel",
				"admin_revoke_user_sessions", "admin_revoke_session", "admin_set_setting",
			}, tool.Name)
		}
	})
}

func TestRegisterEnabledRejectsUnknownToolset(t *testing.T) {
	err := RegisterEnabled(nil, nil, []string{"posts", "nope"}, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `unknown toolset "nope"`)
}
