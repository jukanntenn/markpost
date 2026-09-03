package tools

import (
	"context"
	"encoding/json"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jukanntenn/markpost/mcp/internal/markpost"
)

// registerAdmin adds the admin toolset (opt-in via --toolsets admin): the
// full admin mirror of /api/v1/admin — users, retention policy, posts,
// delivery channels, audit logs, runtime settings, and dashboard stats.
// Every call requires the configured credentials to hold the admin role.
func registerAdmin(s *mcp.Server, c *markpost.Client, readOnly bool) {
	// --- read surface ---
	mcp.AddTool(s, &mcp.Tool{
		Name:        "admin_list_users",
		Description: "List all users (admin), paginated, with username substring search. Items include role, active/VIP flags, retention override, and last login.",
		Annotations: &mcp.ToolAnnotations{Title: "Admin: list users", ReadOnlyHint: true},
	}, adminListUsers(c))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "admin_get_user",
		Description: "Get one user's admin profile (admin) by id, including their post key and retention settings.",
		Annotations: &mcp.ToolAnnotations{Title: "Admin: get user", ReadOnlyHint: true},
	}, adminGetUser(c))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "admin_list_posts",
		Description: "List posts across all users (admin), paginated, with search and username filters.",
		Annotations: &mcp.ToolAnnotations{Title: "Admin: list posts", ReadOnlyHint: true},
	}, adminListPosts(c))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "admin_list_channels",
		Description: "List delivery channels across all users (admin), paginated with owner information.",
		Annotations: &mcp.ToolAnnotations{Title: "Admin: list channels", ReadOnlyHint: true},
	}, adminListChannels(c))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "admin_list_delivery_history",
		Description: "List delivery history across all users (admin), paginated; filter by user_id, channel_id, status (delivered/failed/expired), and error_category.",
		Annotations: &mcp.ToolAnnotations{Title: "Admin: list delivery history", ReadOnlyHint: true},
	}, adminListDeliveryHistory(c))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "admin_get_delivery_stats",
		Description: "Get the site-wide delivery trend (admin): per-day delivered/failed counts over a window (default 7 days, up to 365).",
		Annotations: &mcp.ToolAnnotations{Title: "Admin: site delivery stats", ReadOnlyHint: true},
	}, adminGetSiteDeliveryStats(c))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "admin_list_locked_channels",
		Description: "List channels flagged for persistent delivery failures (admin) — the lockdown queue.",
		Annotations: &mcp.ToolAnnotations{Title: "Admin: list locked channels", ReadOnlyHint: true},
	}, adminListLockedChannels(c))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "admin_list_audit_logs",
		Description: "List audit log entries (admin), paginated, filterable by actor_id, action, target_type, target_id, and RFC3339 since/until. The response carries action-count facets.",
		Annotations: &mcp.ToolAnnotations{Title: "Admin: list audit logs", ReadOnlyHint: true},
	}, adminListAuditLogs(c))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "admin_list_user_sessions",
		Description: "List a user's sessions (admin) by user id.",
		Annotations: &mcp.ToolAnnotations{Title: "Admin: list user sessions", ReadOnlyHint: true},
	}, adminListUserSessions(c))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "admin_get_stats",
		Description: "Get the admin dashboard counters — total users, posts, channels, delivery history — with 7-day deltas.",
		Annotations: &mcp.ToolAnnotations{Title: "Admin: dashboard stats", ReadOnlyHint: true},
	}, adminGetStats(c))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "admin_get_settings",
		Description: "List the instance's runtime settings (admin): the vip feature flag and vip_retention_days override.",
		Annotations: &mcp.ToolAnnotations{Title: "Admin: get settings", ReadOnlyHint: true},
	}, adminGetSettings(c))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "admin_get_retention_defaults",
		Description: "Report the instance's global retention fallback windows (admin): posts and delivery-history defaults applied when a user has no override.",
		Annotations: &mcp.ToolAnnotations{Title: "Admin: retention defaults", ReadOnlyHint: true},
	}, adminRetentionDefaults(c))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "admin_retention_impact",
		Description: "Preview (read-only, no writes) how many posts and delivery-history rows a candidate retention policy would delete, for explicit user ids or every VIP user (admin).",
		Annotations: &mcp.ToolAnnotations{Title: "Admin: retention impact preview", ReadOnlyHint: true},
	}, adminRetentionImpact(c))

	if readOnly {
		return
	}

	// --- write surface ---
	mcp.AddTool(s, &mcp.Tool{
		Name:        "admin_create_user",
		Description: "Create a user (admin) with username and password (email optional). Returns the new user's profile.",
		Annotations: &mcp.ToolAnnotations{Title: "Admin: create user"},
	}, adminCreateUser(c))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "admin_delete_user",
		Description: "Delete a user and their data (admin). Destructive and irreversible.",
		Annotations: &mcp.ToolAnnotations{Title: "Admin: delete user", DestructiveHint: boolPtr(true)},
	}, adminDeleteUser(c))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "admin_set_user_role",
		Description: "Set a user's role (admin): \"admin\" or \"user\".",
		Annotations: &mcp.ToolAnnotations{Title: "Admin: set user role"},
	}, adminSetUserRole(c))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "admin_reset_user_password",
		Description: "Reset a user's password (admin). A temporary password is generated server-side and returned exactly once — surface it to the operator immediately.",
		Annotations: &mcp.ToolAnnotations{Title: "Admin: reset user password"},
	}, adminResetUserPassword(c))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "admin_set_user_active",
		Description: "Activate or deactivate a user (admin). Deactivated users cannot log in.",
		Annotations: &mcp.ToolAnnotations{Title: "Admin: set user active"},
	}, adminSetUserActive(c))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "admin_set_user_vip",
		Description: "Set a user's VIP flag (admin). VIP users follow the vip_retention_days policy when they have no personal override.",
		Annotations: &mcp.ToolAnnotations{Title: "Admin: set user VIP"},
	}, adminSetUserVIP(c))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "admin_set_user_retention",
		Description: "Set one user's retention policy (admin): days = null to inherit the global default, 0 to keep forever, 1-3650 to keep N days.",
		Annotations: &mcp.ToolAnnotations{Title: "Admin: set user retention"},
	}, adminSetUserRetention(c))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "admin_bulk_set_retention",
		Description: "Apply one retention policy to many users at once (admin): explicit user_ids (max 200) or scope=\"vip\" for every VIP user. Run admin_retention_impact first to preview the deletions.",
		Annotations: &mcp.ToolAnnotations{Title: "Admin: bulk set retention", DestructiveHint: boolPtr(true)},
	}, adminBulkSetRetention(c))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "admin_delete_post",
		Description: "Delete any post by QID regardless of owner (admin). Destructive and irreversible.",
		Annotations: &mcp.ToolAnnotations{Title: "Admin: delete post", DestructiveHint: boolPtr(true)},
	}, adminDeletePost(c))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "admin_create_channel",
		Description: "Create a delivery channel owned by an arbitrary user (admin). configuration is a kind-specific JSON object string (see create_channel).",
		Annotations: &mcp.ToolAnnotations{Title: "Admin: create channel"},
	}, adminCreateChannel(c))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "admin_set_channel_enabled",
		Description: "Enable or disable any delivery channel across users (admin).",
		Annotations: &mcp.ToolAnnotations{Title: "Admin: set channel enabled"},
	}, adminSetChannelEnabled(c))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "admin_delete_channel",
		Description: "Delete any delivery channel across users (admin). Destructive.",
		Annotations: &mcp.ToolAnnotations{Title: "Admin: delete channel", DestructiveHint: boolPtr(true)},
	}, adminDeleteChannel(c))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "admin_revoke_user_sessions",
		Description: "Revoke every session of a user (admin), forcing re-login everywhere.",
		Annotations: &mcp.ToolAnnotations{Title: "Admin: revoke user sessions", DestructiveHint: boolPtr(true)},
	}, adminRevokeUserSessions(c))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "admin_revoke_session",
		Description: "Revoke a single session by token id (admin), across all users.",
		Annotations: &mcp.ToolAnnotations{Title: "Admin: revoke session", DestructiveHint: boolPtr(true)},
	}, adminRevokeSession(c))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "admin_set_setting",
		Description: "Upsert one runtime setting (admin). Key \"vip\" takes {\"enabled\": bool}; key \"vip_retention_days\" takes {\"days\": 0-3650}. Returns the full settings list.",
		Annotations: &mcp.ToolAnnotations{Title: "Admin: set setting"},
	}, adminSetSetting(c))
}

type adminSearchArgs struct {
	pageArgs
	Search string `json:"search,omitempty" jsonschema:"username substring search"`
}

func adminListUsers(c *markpost.Client) func(context.Context, *mcp.CallToolRequest, adminSearchArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, args adminSearchArgs) (*mcp.CallToolResult, any, error) {
		raw, err := c.ListUsers(ctx, args.Search, args.Page, args.Limit)
		if err != nil {
			return errorResult("admin_list_users", err)
		}
		return rawResult(raw)
	}
}

type userIDArgs struct {
	UserID int `json:"user_id" jsonschema:"the user's numeric id"`
}

func adminGetUser(c *markpost.Client) func(context.Context, *mcp.CallToolRequest, userIDArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, args userIDArgs) (*mcp.CallToolResult, any, error) {
		raw, err := c.GetUser(ctx, args.UserID)
		if err != nil {
			return errorResult("admin_get_user", err)
		}
		return rawResult(raw)
	}
}

type adminListPostsArgs struct {
	pageArgs
	Search   string `json:"search,omitempty" jsonschema:"title/body search substring"`
	Username string `json:"username,omitempty" jsonschema:"filter by owner username"`
}

func adminListPosts(c *markpost.Client) func(context.Context, *mcp.CallToolRequest, adminListPostsArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, args adminListPostsArgs) (*mcp.CallToolResult, any, error) {
		raw, err := c.ListAllPosts(ctx, args.Search, args.Username, args.Page, args.Limit)
		if err != nil {
			return errorResult("admin_list_posts", err)
		}
		return rawResult(raw)
	}
}

func adminListChannels(c *markpost.Client) func(context.Context, *mcp.CallToolRequest, pageArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, args pageArgs) (*mcp.CallToolResult, any, error) {
		raw, err := c.ListAllChannels(ctx, args.Page, args.Limit)
		if err != nil {
			return errorResult("admin_list_channels", err)
		}
		return rawResult(raw)
	}
}

type adminHistoryArgs struct {
	pageArgs
	UserID        int    `json:"user_id,omitempty" jsonschema:"filter by owner user id"`
	ChannelID     int    `json:"channel_id,omitempty" jsonschema:"filter by channel id"`
	Status        string `json:"status,omitempty" jsonschema:"delivered, failed, or expired"`
	ErrorCategory string `json:"error_category,omitempty" jsonschema:"card_rejected, upstream_client_error, upstream_server_error, upstream_business_error, network, or internal"`
}

func adminListDeliveryHistory(c *markpost.Client) func(context.Context, *mcp.CallToolRequest, adminHistoryArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, args adminHistoryArgs) (*mcp.CallToolResult, any, error) {
		raw, err := c.ListAllDeliveryHistory(ctx, markpost.AdminHistoryFilter{
			UserID:        args.UserID,
			ChannelID:     args.ChannelID,
			Status:        args.Status,
			ErrorCategory: args.ErrorCategory,
		}, args.Page, args.Limit)
		if err != nil {
			return errorResult("admin_list_delivery_history", err)
		}
		return rawResult(raw)
	}
}

func adminGetSiteDeliveryStats(c *markpost.Client) func(context.Context, *mcp.CallToolRequest, daysArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, args daysArgs) (*mcp.CallToolResult, any, error) {
		raw, err := c.GetSiteDeliveryStats(ctx, args.Days)
		if err != nil {
			return errorResult("admin_get_delivery_stats", err)
		}
		return rawResult(raw)
	}
}

type noArgs struct{}

func adminListLockedChannels(c *markpost.Client) func(context.Context, *mcp.CallToolRequest, noArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, _ noArgs) (*mcp.CallToolResult, any, error) {
		raw, err := c.ListLockedChannels(ctx)
		if err != nil {
			return errorResult("admin_list_locked_channels", err)
		}
		return rawResult(raw)
	}
}

type auditLogArgs struct {
	pageArgs
	ActorID    int    `json:"actor_id,omitempty" jsonschema:"filter by acting user id"`
	Action     string `json:"action,omitempty" jsonschema:"filter by action, e.g. user.create, post.delete"`
	TargetType string `json:"target_type,omitempty" jsonschema:"filter by target type (user, post, channel, setting)"`
	TargetID   string `json:"target_id,omitempty" jsonschema:"filter by target id"`
	Since      string `json:"since,omitempty" jsonschema:"RFC3339 lower bound, e.g. 2026-01-01T00:00:00Z"`
	Until      string `json:"until,omitempty" jsonschema:"RFC3339 upper bound"`
}

func adminListAuditLogs(c *markpost.Client) func(context.Context, *mcp.CallToolRequest, auditLogArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, args auditLogArgs) (*mcp.CallToolResult, any, error) {
		raw, err := c.ListAuditLogs(ctx, markpost.AuditLogQuery{
			ActorID:    args.ActorID,
			Action:     args.Action,
			TargetType: args.TargetType,
			TargetID:   args.TargetID,
			Since:      args.Since,
			Until:      args.Until,
		}, args.Page, args.Limit)
		if err != nil {
			return errorResult("admin_list_audit_logs", err)
		}
		return rawResult(raw)
	}
}

func adminListUserSessions(c *markpost.Client) func(context.Context, *mcp.CallToolRequest, userIDArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, args userIDArgs) (*mcp.CallToolResult, any, error) {
		raw, err := c.ListUserSessions(ctx, args.UserID)
		if err != nil {
			return errorResult("admin_list_user_sessions", err)
		}
		return rawResult(raw)
	}
}

func adminGetStats(c *markpost.Client) func(context.Context, *mcp.CallToolRequest, noArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, _ noArgs) (*mcp.CallToolResult, any, error) {
		raw, err := c.GetStats(ctx)
		if err != nil {
			return errorResult("admin_get_stats", err)
		}
		return rawResult(raw)
	}
}

func adminGetSettings(c *markpost.Client) func(context.Context, *mcp.CallToolRequest, noArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, _ noArgs) (*mcp.CallToolResult, any, error) {
		raw, err := c.GetSettings(ctx)
		if err != nil {
			return errorResult("admin_get_settings", err)
		}
		return rawResult(raw)
	}
}

func adminRetentionDefaults(c *markpost.Client) func(context.Context, *mcp.CallToolRequest, noArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, _ noArgs) (*mcp.CallToolResult, any, error) {
		raw, err := c.RetentionDefaults(ctx)
		if err != nil {
			return errorResult("admin_get_retention_defaults", err)
		}
		return rawResult(raw)
	}
}

type retentionTargetArgs struct {
	UserIDs       []int  `json:"user_ids,omitempty" jsonschema:"explicit user ids (max 200); omit when scope=vip"`
	Scope         string `json:"scope,omitempty" jsonschema:"\"vip\" targets every VIP user"`
	RetentionDays *int   `json:"retention_days,omitempty" jsonschema:"null/omit = inherit global default, 0 = keep forever, 1-3650 = keep N days"`
}

func adminRetentionImpact(c *markpost.Client) func(context.Context, *mcp.CallToolRequest, retentionTargetArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, args retentionTargetArgs) (*mcp.CallToolResult, any, error) {
		raw, err := c.RetentionImpact(ctx, retentionInput(args))
		if err != nil {
			return errorResult("admin_retention_impact", err)
		}
		return rawResult(raw)
	}
}

func retentionInput(args retentionTargetArgs) markpost.RetentionTargetInput {
	return markpost.RetentionTargetInput{
		UserIDs:       args.UserIDs,
		Scope:         args.Scope,
		RetentionDays: args.RetentionDays,
	}
}

type adminCreateUserArgs struct {
	Username string `json:"username" jsonschema:"the new user's login name"`
	Password string `json:"password" jsonschema:"the new user's initial password (min 8, max 72 characters)"`
	Email    string `json:"email,omitempty" jsonschema:"optional email address"`
}

func adminCreateUser(c *markpost.Client) func(context.Context, *mcp.CallToolRequest, adminCreateUserArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, args adminCreateUserArgs) (*mcp.CallToolResult, any, error) {
		raw, err := c.CreateUser(ctx, markpost.CreateUserInput{
			Email:    args.Email,
			Username: args.Username,
			Password: args.Password,
		})
		if err != nil {
			return errorResult("admin_create_user", err)
		}
		return rawResult(raw)
	}
}

func adminDeleteUser(c *markpost.Client) func(context.Context, *mcp.CallToolRequest, userIDArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, args userIDArgs) (*mcp.CallToolResult, any, error) {
		raw, err := c.DeleteUser(ctx, args.UserID)
		if err != nil {
			return errorResult("admin_delete_user", err)
		}
		return rawResult(raw)
	}
}

type setRoleArgs struct {
	UserID int    `json:"user_id" jsonschema:"the user's numeric id"`
	Role   string `json:"role" jsonschema:"\"admin\" or \"user\""`
}

func adminSetUserRole(c *markpost.Client) func(context.Context, *mcp.CallToolRequest, setRoleArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, args setRoleArgs) (*mcp.CallToolResult, any, error) {
		raw, err := c.SetUserRole(ctx, args.UserID, args.Role)
		if err != nil {
			return errorResult("admin_set_user_role", err)
		}
		return rawResult(raw)
	}
}

func adminResetUserPassword(c *markpost.Client) func(context.Context, *mcp.CallToolRequest, userIDArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, args userIDArgs) (*mcp.CallToolResult, any, error) {
		raw, err := c.ResetUserPassword(ctx, args.UserID)
		if err != nil {
			return errorResult("admin_reset_user_password", err)
		}
		return rawResult(raw)
	}
}

type setFlagArgs struct {
	UserID int  `json:"user_id" jsonschema:"the user's numeric id"`
	Value  bool `json:"value" jsonschema:"the new flag value"`
}

func adminSetUserActive(c *markpost.Client) func(context.Context, *mcp.CallToolRequest, setFlagArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, args setFlagArgs) (*mcp.CallToolResult, any, error) {
		raw, err := c.SetUserActive(ctx, args.UserID, args.Value)
		if err != nil {
			return errorResult("admin_set_user_active", err)
		}
		return rawResult(raw)
	}
}

func adminSetUserVIP(c *markpost.Client) func(context.Context, *mcp.CallToolRequest, setFlagArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, args setFlagArgs) (*mcp.CallToolResult, any, error) {
		raw, err := c.SetUserVIP(ctx, args.UserID, args.Value)
		if err != nil {
			return errorResult("admin_set_user_vip", err)
		}
		return rawResult(raw)
	}
}

type setRetentionArgs struct {
	UserID        int  `json:"user_id" jsonschema:"the user's numeric id"`
	RetentionDays *int `json:"retention_days" jsonschema:"null/omit = inherit global default, 0 = keep forever, 1-3650 = keep N days"`
}

func adminSetUserRetention(c *markpost.Client) func(context.Context, *mcp.CallToolRequest, setRetentionArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, args setRetentionArgs) (*mcp.CallToolResult, any, error) {
		raw, err := c.SetUserRetention(ctx, args.UserID, args.RetentionDays)
		if err != nil {
			return errorResult("admin_set_user_retention", err)
		}
		return rawResult(raw)
	}
}

func adminBulkSetRetention(c *markpost.Client) func(context.Context, *mcp.CallToolRequest, retentionTargetArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, args retentionTargetArgs) (*mcp.CallToolResult, any, error) {
		raw, err := c.BulkSetRetention(ctx, retentionInput(args))
		if err != nil {
			return errorResult("admin_bulk_set_retention", err)
		}
		return rawResult(raw)
	}
}

func adminDeletePost(c *markpost.Client) func(context.Context, *mcp.CallToolRequest, postIDArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, args postIDArgs) (*mcp.CallToolResult, any, error) {
		if err := c.DeleteAnyPost(ctx, args.QID); err != nil {
			return errorResult("admin_delete_post", err)
		}
		return textResult(`{"deleted": true}`), nil, nil
	}
}

type adminCreateChannelArgs struct {
	UserID        int    `json:"user_id" jsonschema:"the owning user's numeric id"`
	Kind          string `json:"kind" jsonschema:"channel kind, e.g. feishu, slack, or custom webhook"`
	Name          string `json:"name" jsonschema:"human-readable channel name"`
	Configuration string `json:"configuration" jsonschema:"kind-specific JSON object as a string, e.g. {\"webhook_url\": \"https://...\"}"`
	Keywords      string `json:"keywords,omitempty" jsonschema:"comma-separated keyword filter (empty = all posts)"`
}

func adminCreateChannel(c *markpost.Client) func(context.Context, *mcp.CallToolRequest, adminCreateChannelArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, args adminCreateChannelArgs) (*mcp.CallToolResult, any, error) {
		raw, err := c.CreateChannelForUser(ctx, markpost.AdminCreateChannelInput{
			UserID:        args.UserID,
			Kind:          args.Kind,
			Name:          args.Name,
			Configuration: json.RawMessage(args.Configuration),
			Keywords:      args.Keywords,
		})
		if err != nil {
			return errorResult("admin_create_channel", err)
		}
		return rawResult(raw)
	}
}

type setChannelEnabledArgs struct {
	ChannelID int  `json:"channel_id" jsonschema:"the channel's numeric id"`
	Enabled   bool `json:"enabled" jsonschema:"true to enable, false to disable"`
}

func adminSetChannelEnabled(c *markpost.Client) func(context.Context, *mcp.CallToolRequest, setChannelEnabledArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, args setChannelEnabledArgs) (*mcp.CallToolResult, any, error) {
		raw, err := c.SetChannelEnabled(ctx, args.ChannelID, args.Enabled)
		if err != nil {
			return errorResult("admin_set_channel_enabled", err)
		}
		return rawResult(raw)
	}
}

type channelIDAdminArgs struct {
	ChannelID int `json:"channel_id" jsonschema:"the channel's numeric id"`
}

func adminDeleteChannel(c *markpost.Client) func(context.Context, *mcp.CallToolRequest, channelIDAdminArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, args channelIDAdminArgs) (*mcp.CallToolResult, any, error) {
		raw, err := c.DeleteChannelByID(ctx, args.ChannelID)
		if err != nil {
			return errorResult("admin_delete_channel", err)
		}
		return rawResult(raw)
	}
}

func adminRevokeUserSessions(c *markpost.Client) func(context.Context, *mcp.CallToolRequest, userIDArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, args userIDArgs) (*mcp.CallToolResult, any, error) {
		raw, err := c.RevokeUserSessions(ctx, args.UserID)
		if err != nil {
			return errorResult("admin_revoke_user_sessions", err)
		}
		return rawResult(raw)
	}
}

func adminRevokeSession(c *markpost.Client) func(context.Context, *mcp.CallToolRequest, sessionIDArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, args sessionIDArgs) (*mcp.CallToolResult, any, error) {
		raw, err := c.RevokeSessionByID(ctx, args.TokenID)
		if err != nil {
			return errorResult("admin_revoke_session", err)
		}
		return rawResult(raw)
	}
}

type setSettingArgs struct {
	Key     string `json:"key" jsonschema:"setting key: vip or vip_retention_days"`
	Enabled *bool  `json:"enabled,omitempty" jsonschema:"for key vip: the feature flag value"`
	Days    *int   `json:"days,omitempty" jsonschema:"for key vip_retention_days: 0-3650 (null = follow global config)"`
}

func adminSetSetting(c *markpost.Client) func(context.Context, *mcp.CallToolRequest, setSettingArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, args setSettingArgs) (*mcp.CallToolResult, any, error) {
		raw, err := c.SetSetting(ctx, args.Key, markpost.SetSettingInput{Enabled: args.Enabled, Days: args.Days})
		if err != nil {
			return errorResult("admin_set_setting", err)
		}
		return rawResult(raw)
	}
}
