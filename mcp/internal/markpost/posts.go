package markpost

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// CreatePost publishes a post: it resolves the caller's post key via the
// authenticated /post-key endpoint, then submits {title, body} to the public
// POST /{post_key} endpoint (markpost's only creation path). Returns the
// backend's id plus the public render URL.
func (c *Client) CreatePost(ctx context.Context, title, body string) (*CreatePostResult, error) {
	var pk postKeyResponse
	if err := c.exchangeAuth(ctx, http.MethodGet, apiPrefix+"/post-key", nil, &pk); err != nil {
		return nil, fmt.Errorf("resolve post key: %w", err)
	}
	var created createPostResponse
	if err := c.exchangePlain(ctx, http.MethodPost, "/"+url.PathEscape(pk.PostKey), createPostRequest{Title: title, Body: body}, &created); err != nil {
		return nil, err
	}
	return &CreatePostResult{ID: created.ID, URL: c.baseURL + "/" + created.ID}, nil
}

// ListPosts returns the caller's posts (paginated, optional title/body
// search) as the raw {items, total, page, limit, total_pages} envelope.
func (c *Client) ListPosts(ctx context.Context, search string, page, limit int) (json.RawMessage, error) {
	q := paginationQuery(page, limit)
	if search != "" {
		q.Set("search", search)
	}
	return c.request(ctx, http.MethodGet, apiPrefix+"/posts", q, nil, true)
}

// GetPostRaw fetches the post's markdown: GET /{qid}?format=raw, which the
// backend serves as "# title\n\nbody" text/markdown.
func (c *Client) GetPostRaw(ctx context.Context, qid string) ([]byte, error) {
	q := url.Values{"format": {"raw"}}
	return c.request(ctx, http.MethodGet, "/"+url.PathEscape(qid), q, nil, false)
}

// DeletePost removes one of the caller's posts by QID (204 on success).
func (c *Client) DeletePost(ctx context.Context, qid string) error {
	_, err := c.request(ctx, http.MethodDelete, apiPrefix+"/posts/"+url.PathEscape(qid), nil, nil, true)
	return err
}

// exchangeAuth is exchange() through the bearer/recovery loop of request().
func (c *Client) exchangeAuth(ctx context.Context, method, path string, body, out any) error {
	raw, err := c.request(ctx, method, path, nil, body, true)
	if err != nil {
		return err
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(raw, out)
}

// exchangePlain is exchange() without authentication (public endpoints).
func (c *Client) exchangePlain(ctx context.Context, method, path string, body, out any) error {
	raw, err := c.request(ctx, method, path, nil, body, false)
	if err != nil {
		return err
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(raw, out)
}
