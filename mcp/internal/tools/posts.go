package tools

import (
	"context"
	"encoding/json"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jukanntenn/markpost/mcp/internal/markpost"
)

func jsonMarshal(v any) (json.RawMessage, error) {
	return json.Marshal(v)
}

// registerPosts adds the posts toolset: publish, browse, read, and delete the
// authenticated user's markdown posts.
func registerPosts(s *mcp.Server, c *markpost.Client, readOnly bool) {
	if !readOnly {
		mcp.AddTool(s, &mcp.Tool{
			Name:        "create_post",
			Description: "Publish a new markdown post on the markpost instance. Returns the post id and its public URL. The post is delivered to the user's enabled delivery channels according to their keyword filters.",
			Annotations: &mcp.ToolAnnotations{Title: "Create post"},
		}, createPost(c))

		mcp.AddTool(s, &mcp.Tool{
			Name:        "delete_post",
			Description: "Delete one of the current user's posts by its QID (the short id from create_post / list_posts). The public URL stops resolving immediately. Destructive and irreversible.",
			Annotations: &mcp.ToolAnnotations{Title: "Delete post", DestructiveHint: boolPtr(true)},
		}, deletePost(c))
	}

	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_posts",
		Description: "List the current user's posts, newest first. Optional title/body search substring. Paginated ({items, total, page, limit, total_pages}); each item carries id (QID), title, and created_at.",
		Annotations: &mcp.ToolAnnotations{Title: "List posts", ReadOnlyHint: true},
	}, listPosts(c))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_post",
		Description: "Fetch a post's markdown source by its QID. Returns '# <title>' followed by the body — the authoritative content, not the rendered HTML.",
		Annotations: &mcp.ToolAnnotations{Title: "Get post", ReadOnlyHint: true},
	}, getPost(c))
}

func boolPtr(b bool) *bool { return &b }

type createPostArgs struct {
	Title string `json:"title" jsonschema:"the post title"`
	Body  string `json:"body" jsonschema:"the post body in Markdown"`
}

func createPost(c *markpost.Client) func(context.Context, *mcp.CallToolRequest, createPostArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, args createPostArgs) (*mcp.CallToolResult, any, error) {
		res, err := c.CreatePost(ctx, args.Title, args.Body)
		if err != nil {
			return errorResult("create_post", err)
		}
		raw, err := jsonMarshal(res)
		if err != nil {
			return errorResult("create_post", err)
		}
		return rawResult(raw)
	}
}

type postIDArgs struct {
	QID string `json:"qid" jsonschema:"the post's QID (short id from create_post or list_posts)"`
}

func deletePost(c *markpost.Client) func(context.Context, *mcp.CallToolRequest, postIDArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, args postIDArgs) (*mcp.CallToolResult, any, error) {
		if err := c.DeletePost(ctx, args.QID); err != nil {
			return errorResult("delete_post", err)
		}
		return textResult(`{"deleted": true}`), nil, nil
	}
}

type listPostsArgs struct {
	pageArgs
	Search string `json:"search,omitempty" jsonschema:"substring matched against title and body"`
}

func listPosts(c *markpost.Client) func(context.Context, *mcp.CallToolRequest, listPostsArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, args listPostsArgs) (*mcp.CallToolResult, any, error) {
		raw, err := c.ListPosts(ctx, args.Search, args.Page, args.Limit)
		if err != nil {
			return errorResult("list_posts", err)
		}
		return rawResult(raw)
	}
}

func getPost(c *markpost.Client) func(context.Context, *mcp.CallToolRequest, postIDArgs) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, args postIDArgs) (*mcp.CallToolResult, any, error) {
		raw, err := c.GetPostRaw(ctx, args.QID)
		if err != nil {
			return errorResult("get_post", err)
		}
		return textResult(string(raw)), nil, nil
	}
}
