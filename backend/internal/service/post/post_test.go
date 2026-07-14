package post

import (
	"context"
	"strings"
	"testing"

	"markpost/internal/domain/post"
	"markpost/internal/infra"
	"markpost/internal/service"
)

func setupPostService(t *testing.T) (*Service, *infra.PostRepository) {
	t.Helper()
	db := infra.SetupTestDB(t)
	repo := infra.NewPostRepository(db)
	svc := NewService(repo, nil)
	return svc, repo.(*infra.PostRepository)
}

func TestService_CreatePost(t *testing.T) {
	t.Run("creates post successfully", func(t *testing.T) {
		svc, _ := setupPostService(t)
		ctx := context.Background()

		qid, err := svc.CreatePost(ctx, "Test Title", "Test Body", 1)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if qid == "" {
			t.Error("expected qid, got empty")
		}
	})

	t.Run("creates post with delivery enqueuer", func(t *testing.T) {
		db := infra.SetupTestDB(t)
		repo := infra.NewPostRepository(db)
		enqueuer := &mockEnqueuer{}
		svc := NewService(repo, enqueuer)
		ctx := context.Background()

		qid, err := svc.CreatePost(ctx, "Title", "Body", 1)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if qid == "" {
			t.Error("expected qid")
		}
		if len(enqueuer.jobs) != 1 {
			t.Errorf("expected 1 enqueued job, got %d", len(enqueuer.jobs))
		}
		if enqueuer.jobs[0].Title != "Title" {
			t.Errorf("job title = %q, want %q", enqueuer.jobs[0].Title, "Title")
		}
	})
}

type mockEnqueuer struct {
	jobs []post.DeliveryJob
}

func (m *mockEnqueuer) Enqueue(job post.DeliveryJob) {
	m.jobs = append(m.jobs, job)
}

func TestService_GetPostMarkdown(t *testing.T) {
	svc, repo := setupPostService(t)
	ctx := context.Background()

	created, _ := repo.Create(ctx, "Test Title", "Test Body", 1)

	t.Run("returns markdown for valid post", func(t *testing.T) {
		title, body, _, _, err := svc.GetPostMarkdown(ctx, created.QID)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if title != "Test Title" {
			t.Errorf("expected title 'Test Title', got: %s", title)
		}
		if body != "Test Body" {
			t.Errorf("expected body 'Test Body', got: %s", body)
		}
	})

	t.Run("returns error for non-existent post", func(t *testing.T) {
		_, _, _, _, err := svc.GetPostMarkdown(ctx, "nonexistent")
		if err == nil {
			t.Fatal("expected error for non-existent post")
		}
	})
}

func TestService_RenderPostHTML(t *testing.T) {
	svc, repo := setupPostService(t)
	ctx := context.Background()

	created, _ := repo.Create(ctx, "Test Title", "# Heading\n\nParagraph", 1)

	t.Run("renders HTML for valid post", func(t *testing.T) {
		title, html, _, _, err := svc.RenderPostHTML(ctx, created.QID)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if title != "Test Title" {
			t.Errorf("expected title 'Test Title', got: %s", title)
		}
		if html == "" {
			t.Error("expected HTML content, got empty")
		}
	})

	t.Run("returns error for non-existent post", func(t *testing.T) {
		_, _, _, _, err := svc.RenderPostHTML(ctx, "nonexistent")
		if err == nil {
			t.Fatal("expected error for non-existent post")
		}
	})
}

func TestService_RenderPostHTML_GFM(t *testing.T) {
	svc, repo := setupPostService(t)
	ctx := context.Background()

	body := "| h1 | h2 |\n|----|----|\n| a  | b  |\n\n" +
		"<details>\n<summary>more</summary>\n\n" +
		"- [x] done\n\n" +
		"~~strike~~\n\n" +
		"https://example.com\n"

	created, err := repo.Create(ctx, "T", body, 1)
	if err != nil {
		t.Fatalf("create post: %v", err)
	}

	_, html, _, _, err := svc.RenderPostHTML(ctx, created.QID)
	if err != nil {
		t.Fatalf("render post: %v", err)
	}

	for _, want := range []string{
		"<table>",
		"<thead>",
		"<details",
		"<summary>more</summary>",
		"type=checkbox",
		"<del>strike</del>",
		"href=https://example.com",
	} {
		if !strings.Contains(html, want) {
			t.Errorf("expected %q in rendered HTML\nhtml: %s", want, html)
		}
	}
}

func TestService_RenderPostHTML_HardWraps(t *testing.T) {
	svc, repo := setupPostService(t)
	ctx := context.Background()

	body := "line one\nline two\nline three\n"

	created, err := repo.Create(ctx, "T", body, 1)
	if err != nil {
		t.Fatalf("create post: %v", err)
	}

	_, html, _, _, err := svc.RenderPostHTML(ctx, created.QID)
	if err != nil {
		t.Fatalf("render post: %v", err)
	}

	if !strings.Contains(html, "line one<br") {
		t.Errorf("expected soft line break to render as <br>\nhtml: %s", html)
	}
	if !strings.Contains(html, "line two<br") {
		t.Errorf("expected soft line break to render as <br>\nhtml: %s", html)
	}
}

func TestService_RenderPostHTML_SanitizesDangerousHTML(t *testing.T) {
	svc, repo := setupPostService(t)
	ctx := context.Background()

	body := "https://example.com\n\n" +
		"<script>alert(1)</script>\n\n" +
		"<img src=x onerror=alert(1)>\n\n" +
		"<a href=\"javascript:alert(1)\">x</a>\n"

	created, err := repo.Create(ctx, "T", body, 1)
	if err != nil {
		t.Fatalf("create post: %v", err)
	}

	_, html, _, _, err := svc.RenderPostHTML(ctx, created.QID)
	if err != nil {
		t.Fatalf("render post: %v", err)
	}

	for _, unsafe := range []string{"<script", "onerror", "javascript:"} {
		if strings.Contains(html, unsafe) {
			t.Errorf("sanitized HTML must not contain %q\nhtml: %s", unsafe, html)
		}
	}
	for _, want := range []string{
		"href=https://example.com",
		"target=_blank",
		"noopener",
		"noreferrer",
		"nofollow",
	} {
		if !strings.Contains(html, want) {
			t.Errorf("expected %q in rendered HTML\nhtml: %s", want, html)
		}
	}
}

func TestService_RenderPostHTML_UnclosedScriptKeepsContent(t *testing.T) {
	svc, repo := setupPostService(t)
	ctx := context.Background()

	body := "before <script> after\n\n## survived\n\ntail text"
	created, err := repo.Create(ctx, "T", body, 1)
	if err != nil {
		t.Fatalf("create post: %v", err)
	}

	_, html, _, _, err := svc.RenderPostHTML(ctx, created.QID)
	if err != nil {
		t.Fatalf("render post: %v", err)
	}

	if strings.Contains(html, "<script") {
		t.Errorf("rendered HTML must not contain a real <script tag\nhtml: %s", html)
	}
	for _, want := range []string{"before", "after", "survived", "tail text", "&lt;script"} {
		if !strings.Contains(html, want) {
			t.Errorf("content after an unclosed <script> must survive; missing %q\nhtml: %s", want, html)
		}
	}
}

func TestService_GetUserPosts(t *testing.T) {
	svc, repo := setupPostService(t)
	ctx := context.Background()

	_, _ = repo.Create(ctx, "Title 1", "Body 1", 1)
	_, _ = repo.Create(ctx, "Title 2", "Body 2", 1)
	_, _ = repo.Create(ctx, "Title 3", "Body 3", 2)

	t.Run("returns posts for user", func(t *testing.T) {
		posts, total, err := svc.GetUserPosts(ctx, 1, 0, 10)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if len(posts) != 2 {
			t.Errorf("expected 2 posts, got: %d", len(posts))
		}
		if total != 2 {
			t.Errorf("expected total 2, got: %d", total)
		}
	})

	t.Run("returns empty for user with no posts", func(t *testing.T) {
		posts, total, err := svc.GetUserPosts(ctx, 999, 0, 10)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if len(posts) != 0 {
			t.Errorf("expected 0 posts, got: %d", len(posts))
		}
		if total != 0 {
			t.Errorf("expected total 0, got: %d", total)
		}
	})
}

func TestService_GetAllPosts(t *testing.T) {
	svc, repo := setupPostService(t)
	ctx := context.Background()

	_, _ = repo.Create(ctx, "Alpha", "Body", 1)
	_, _ = repo.Create(ctx, "Beta", "Body", 2)

	posts, total, err := svc.GetAllPosts(ctx, "", 0, 10)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(posts) != 2 {
		t.Errorf("expected 2 posts, got: %d", len(posts))
	}
	if total != 2 {
		t.Errorf("expected total 2, got: %d", total)
	}
}

func TestService_DeletePost(t *testing.T) {
	svc, repo := setupPostService(t)
	ctx := context.Background()

	created, _ := repo.Create(ctx, "Title", "Body", 1)

	t.Run("deletes post successfully", func(t *testing.T) {
		err := svc.DeletePost(ctx, created.ID)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
	})

	t.Run("returns no error for non-existent post", func(t *testing.T) {
		err := svc.DeletePost(ctx, 999)
		if err != nil {
			t.Fatalf("expected no error (idempotent delete), got: %v", err)
		}
	})
}

func TestService_PruneExpired(t *testing.T) {
	svc, _ := setupPostService(t)
	ctx := context.Background()

	t.Run("returns error for non-positive retention days", func(t *testing.T) {
		err := svc.PruneExpired(ctx, 0, 100)
		if err == nil {
			t.Fatal("expected error for zero retention days")
		}
		se, ok := service.AsError(err)
		if !ok {
			t.Fatal("expected service error")
		}
		if se.Code != service.ErrValidation {
			t.Errorf("expected code %q, got %q", service.ErrValidation.Value, se.Code.Value)
		}
	})

	t.Run("uses default batch size when zero", func(t *testing.T) {
		err := svc.PruneExpired(ctx, 7, 0)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
	})
}

func TestService_CountExpired(t *testing.T) {
	svc, _ := setupPostService(t)
	ctx := context.Background()

	t.Run("returns error for non-positive retention days", func(t *testing.T) {
		_, err := svc.CountExpired(ctx, 0)
		if err == nil {
			t.Fatal("expected error for zero retention days")
		}
	})

	t.Run("returns count for valid retention days", func(t *testing.T) {
		count, err := svc.CountExpired(ctx, 7)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if count != 0 {
			t.Errorf("expected count 0, got: %d", count)
		}
	})
}
