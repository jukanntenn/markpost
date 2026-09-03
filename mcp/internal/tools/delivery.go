package tools

import (
	"context"
	"encoding/json"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jukanntenn/markpost/mcp/internal/markpost"
)

// registerDelivery adds the delivery toolset: manage the user's delivery
// channels (webhook forwards with keyword filtering) and inspect the delivery
// pipeline (history, latest per channel, stats, in-flight attempts).
func registerDelivery(s *mcp.Server, c *markpost.Client, readOnly bool) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_channels",
		Description: "List the current user's delivery channels. Each channel has a kind (e.g. feishu, slack, custom webhook), a name, enabled flag, kind-specific configuration, and a keyword filter — posts whose body matches the keywords are forwarded to that channel.",
		Annotations: &mcp.ToolAnnotations{Title: "List delivery channels", ReadOnlyHint: true},
	}, listChannels(c))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_delivery_history",
		Description: "List the user's delivery attempts, newest first, with terminal status (delivered/failed/expired), last error, and error category. Filterable by channel and status; paginated.",
		Annotations: &mcp.ToolAnnotations{Title: "List delivery history", ReadOnlyHint: true},
	}, listDeliveryHistory(c))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_latest_deliveries",
		Description: "List the single most recent delivery attempt per channel — the quickest health overview of all channels.",
		Annotations: &mcp.ToolAnnotations{Title: "List latest deliveries", ReadOnlyHint: true},
	}, listLatestDeliveries(c))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_delivery_stats",
		Description: "Get the user's delivery counters for today plus a per-day delivered/failed trend (default last 7 days, up to 365).",
		Annotations: &mcp.ToolAnnotations{Title: "Get delivery stats", ReadOnlyHint: true},
	}, getDeliveryStats(c))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_pending_deliveries",
		Description: "List the user's in-flight delivery attempts (queued or retrying).",
		Annotations: &mcp.ToolAnnotations{Title: "List pending deliveries", ReadOnlyHint: true},
	}, listPendingDeliveries(c))

	if readOnly {
		return
	}

	mcp.AddTool(s, &mcp.Tool{
		Name:        "create_channel",
		Description: "Create a delivery channel for the current user. configuration is a kind-specific JSON object — feishu takes {\"webhook_url\": \"...\", \"card_link_url\": \"...\"}; the exact fields depend on the kind. keywords is a comma-separated filter: only posts matching at least one keyword are delivered (empty = all posts).",
		Annotations: &mcp.ToolAnnotations{Title: "Create delivery channel"},
	}, createChannel(c))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "update_channel",
		Description: "Partially update one of the current user's delivery channels by id — only the provided fields change (kind, name, configuration, keywords, enabled).",
		Annotations: &mcp.ToolAnnotations{Title: "Update delivery channel"},
	}, updateChannel(c))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "delete_channel",
		Description: "Delete one of the current user's delivery channels by id. Delivery history is preserved but the channel stops receiving posts. Destructive.",
		Annotations: &mcp.ToolAnnotations{Title: "Delete delivery channel", DestructiveHint: boolPtr(true)},
	}, deleteChannel(c))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "test_channel",
		Description: "Send a test message to one of the current user's delivery channels to verify its configuration end-to-end. The upstream webhook is actually called.",
		Annotations: &mcp.ToolAnnotations{Title: "Test delivery channel", OpenWorldHint: boolPtr(true)},
	}, testChannel(c))
}

func listChannels(c *markpost.Client) func(context.Context, *mcp.CallToolRequest, struct{}) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
		raw, err := c.ListChannels(ctx)
		if err != nil {
			return errorResult("list_channels", err)
		}
		return rawResult(raw)
	}
}

type listDeliveryHistoryArgs struct {
	pageArgs
	ChannelID int    `json:"channel_id,omitempty" jsonschema:"filter by delivery channel id"`
	Status    string `json:"status,omitempty" jsonschema:"filter by terminal status: delivered, failed, or expired"`
}

func listDeliveryHistory(c *markpost.Client) func(context.Context, *mcp.CallToolRequest, listDeliveryHistoryArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, args listDeliveryHistoryArgs) (*mcp.CallToolResult, any, error) {
		raw, err := c.ListDeliveryHistory(ctx, args.ChannelID, args.Status, args.Page, args.Limit)
		if err != nil {
			return errorResult("list_delivery_history", err)
		}
		return rawResult(raw)
	}
}

func listLatestDeliveries(c *markpost.Client) func(context.Context, *mcp.CallToolRequest, struct{}) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
		raw, err := c.ListLatestDeliveries(ctx)
		if err != nil {
			return errorResult("list_latest_deliveries", err)
		}
		return rawResult(raw)
	}
}

type daysArgs struct {
	Days int `json:"days,omitempty" jsonschema:"trend window in days, 1-365 (default 7)"`
}

func getDeliveryStats(c *markpost.Client) func(context.Context, *mcp.CallToolRequest, daysArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, args daysArgs) (*mcp.CallToolResult, any, error) {
		raw, err := c.GetDeliveryStats(ctx, args.Days)
		if err != nil {
			return errorResult("get_delivery_stats", err)
		}
		return rawResult(raw)
	}
}

func listPendingDeliveries(c *markpost.Client) func(context.Context, *mcp.CallToolRequest, struct{}) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
		raw, err := c.ListPendingDeliveries(ctx)
		if err != nil {
			return errorResult("list_pending_deliveries", err)
		}
		return rawResult(raw)
	}
}

type createChannelArgs struct {
	Kind          string `json:"kind" jsonschema:"channel kind, e.g. feishu, slack, or custom webhook"`
	Name          string `json:"name" jsonschema:"human-readable channel name"`
	Configuration string `json:"configuration" jsonschema:"kind-specific JSON object as a string, e.g. {\"webhook_url\": \"https://...\"}"`
	Keywords      string `json:"keywords,omitempty" jsonschema:"comma-separated keywords; only posts matching at least one are delivered (empty = all posts)"`
}

func createChannel(c *markpost.Client) func(context.Context, *mcp.CallToolRequest, createChannelArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, args createChannelArgs) (*mcp.CallToolResult, any, error) {
		raw, err := c.CreateChannel(ctx, markpost.CreateChannelInput{
			Kind:          args.Kind,
			Name:          args.Name,
			Configuration: json.RawMessage(args.Configuration),
			Keywords:      args.Keywords,
		})
		if err != nil {
			return errorResult("create_channel", err)
		}
		return rawResult(raw)
	}
}

type updateChannelArgs struct {
	ID            int    `json:"id" jsonschema:"channel id from list_channels"`
	Kind          string `json:"kind,omitempty" jsonschema:"new channel kind"`
	Name          string `json:"name,omitempty" jsonschema:"new channel name"`
	Configuration string `json:"configuration,omitempty" jsonschema:"new kind-specific JSON object as a string"`
	Keywords      string `json:"keywords,omitempty" jsonschema:"new comma-separated keyword filter"`
	Enabled       *bool  `json:"enabled,omitempty" jsonschema:"enable or disable the channel"`
}

func updateChannel(c *markpost.Client) func(context.Context, *mcp.CallToolRequest, updateChannelArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, args updateChannelArgs) (*mcp.CallToolResult, any, error) {
		in := markpost.UpdateChannelInput{
			Name:     optStr(args.Name),
			Keywords: optStr(args.Keywords),
			Enabled:  args.Enabled,
		}
		if args.Kind != "" {
			in.Kind = &args.Kind
		}
		if args.Configuration != "" {
			cfg := json.RawMessage(args.Configuration)
			in.Configuration = &cfg
		}
		raw, err := c.UpdateChannel(ctx, args.ID, in)
		if err != nil {
			return errorResult("update_channel", err)
		}
		return rawResult(raw)
	}
}

type channelIDArgs struct {
	ID int `json:"id" jsonschema:"channel id from list_channels"`
}

func deleteChannel(c *markpost.Client) func(context.Context, *mcp.CallToolRequest, channelIDArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, args channelIDArgs) (*mcp.CallToolResult, any, error) {
		if err := c.DeleteChannel(ctx, args.ID); err != nil {
			return errorResult("delete_channel", err)
		}
		return textResult(`{"deleted": true}`), nil, nil
	}
}

func testChannel(c *markpost.Client) func(context.Context, *mcp.CallToolRequest, channelIDArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, args channelIDArgs) (*mcp.CallToolResult, any, error) {
		raw, err := c.TestChannel(ctx, args.ID)
		if err != nil {
			return errorResult("test_channel", err)
		}
		return rawResult(raw)
	}
}

func optStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
