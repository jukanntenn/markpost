package tools

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jukanntenn/markpost/mcp/internal/markpost"
)

// registerAccount adds the account toolset: the caller's own retention
// policy, sessions, post key, and password.
func registerAccount(s *mcp.Server, c *markpost.Client, readOnly bool) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_my_retention",
		Description: "Report the caller's effective data retention: how many days posts and delivery history are kept (0 = kept forever). Reflects per-user overrides and the instance's global defaults.",
		Annotations: &mcp.ToolAnnotations{Title: "Get my retention", ReadOnlyHint: true},
	}, getRetention(c))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_my_sessions",
		Description: "List the caller's active sessions (refresh tokens), including the one this MCP server holds.",
		Annotations: &mcp.ToolAnnotations{Title: "List my sessions", ReadOnlyHint: true},
	}, listSessions(c))

	if readOnly {
		return
	}

	mcp.AddTool(s, &mcp.Tool{
		Name:        "revoke_my_session",
		Description: "Revoke one of the caller's sessions by its token id (from list_my_sessions). If it is this server's own session, the next tool call transparently re-authenticates. Destructive to that session.",
		Annotations: &mcp.ToolAnnotations{Title: "Revoke my session", DestructiveHint: boolPtr(true)},
	}, revokeSession(c))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "revoke_my_other_sessions",
		Description: "Revoke all of the caller's sessions except the one this MCP server currently holds — a logout-everywhere-else operation.",
		Annotations: &mcp.ToolAnnotations{Title: "Revoke my other sessions", DestructiveHint: boolPtr(true)},
	}, revokeOtherSessions(c))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "rotate_post_key",
		Description: "Rotate the caller's post key (the credential used by create_post). The previous key stops working immediately; the new one is returned. Existing publish integrations must be updated. Destructive.",
		Annotations: &mcp.ToolAnnotations{Title: "Rotate post key", DestructiveHint: boolPtr(true)},
	}, rotatePostKey(c))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "change_my_password",
		Description: "Change the caller's password. This server's session continues seamlessly (the response contains a fresh token pair, which the server adopts automatically). Destructive to other password-based logins.",
		Annotations: &mcp.ToolAnnotations{Title: "Change my password", DestructiveHint: boolPtr(true)},
	}, changePassword(c))
}

func getRetention(c *markpost.Client) func(context.Context, *mcp.CallToolRequest, struct{}) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
		raw, err := c.GetRetention(ctx)
		if err != nil {
			return errorResult("get_my_retention", err)
		}
		return rawResult(raw)
	}
}

func listSessions(c *markpost.Client) func(context.Context, *mcp.CallToolRequest, struct{}) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
		raw, err := c.ListSessions(ctx)
		if err != nil {
			return errorResult("list_my_sessions", err)
		}
		return rawResult(raw)
	}
}

type sessionIDArgs struct {
	TokenID int `json:"token_id" jsonschema:"session (refresh token) id from list_my_sessions"`
}

func revokeSession(c *markpost.Client) func(context.Context, *mcp.CallToolRequest, sessionIDArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, args sessionIDArgs) (*mcp.CallToolResult, any, error) {
		raw, err := c.RevokeSession(ctx, args.TokenID)
		if err != nil {
			return errorResult("revoke_my_session", err)
		}
		return rawResult(raw)
	}
}

func revokeOtherSessions(c *markpost.Client) func(context.Context, *mcp.CallToolRequest, struct{}) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
		raw, err := c.RevokeOtherSessions(ctx)
		if err != nil {
			return errorResult("revoke_my_other_sessions", err)
		}
		return rawResult(raw)
	}
}

func rotatePostKey(c *markpost.Client) func(context.Context, *mcp.CallToolRequest, struct{}) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
		raw, err := c.RotatePostKey(ctx)
		if err != nil {
			return errorResult("rotate_post_key", err)
		}
		return rawResult(raw)
	}
}

type changePasswordArgs struct {
	CurrentPassword string `json:"current_password" jsonschema:"the current password (may be empty for OAuth-only accounts without a local password)"`
	NewPassword     string `json:"new_password" jsonschema:"the new password (min 8, max 72 characters)"`
}

func changePassword(c *markpost.Client) func(context.Context, *mcp.CallToolRequest, changePasswordArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, args changePasswordArgs) (*mcp.CallToolResult, any, error) {
		raw, err := c.ChangePassword(ctx, args.CurrentPassword, args.NewPassword)
		if err != nil {
			return errorResult("change_my_password", err)
		}
		return rawResult(raw)
	}
}
