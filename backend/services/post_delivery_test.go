package services

import (
	"testing"

	"markpost/models"
)

type stubDeliveryEnqueuer struct {
	called int
	jobs   []DeliveryJob
}

func (s *stubDeliveryEnqueuer) Enqueue(job DeliveryJob) {
	s.called++
	s.jobs = append(s.jobs, job)
}

func TestTruncateRunes(t *testing.T) {
	if got := truncateRunes("hello", 0); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}

	if got := truncateRunes("hello", 5); got != "hello" {
		t.Fatalf("expected no truncation, got %q", got)
	}

	if got := truncateRunes("hello", 3); got != "hel" {
		t.Fatalf("expected hel, got %q", got)
	}

	if got := truncateRunes("你好世界", 3); got != "你好世" {
		t.Fatalf("expected 你好世, got %q", got)
	}
}

func TestPostService_CreatePostEnqueuesDeliveryJob(t *testing.T) {
	enq := &stubDeliveryEnqueuer{}
	postRepo := &stubPostRepo{
		createPostResult: &models.Post{QID: "qid-1", Title: "t", Body: "b", UserID: 9},
	}

	svc := NewPostService(postRepo, enq)
	qid, err := svc.CreatePost("t", "b", 9)
	if err != nil {
		t.Fatalf("CreatePost error: %v", err)
	}
	if qid != "qid-1" {
		t.Fatalf("unexpected qid: %s", qid)
	}
	if enq.called != 1 {
		t.Fatalf("expected enqueue called once, got %d", enq.called)
	}
	if len(enq.jobs) != 1 || enq.jobs[0].PostQID != "qid-1" || enq.jobs[0].UserID != 9 {
		t.Fatalf("unexpected enqueue jobs: %+v", enq.jobs)
	}
}

func TestBuildPostURL(t *testing.T) {
	t.Run("uses public url when set", func(t *testing.T) {
		got := buildPostURL("https://example.com", "127.0.0.1", 7330, "abc")
		if got != "https://example.com/abc" {
			t.Fatalf("unexpected url: %s", got)
		}
	})

	t.Run("trims trailing slash from public url", func(t *testing.T) {
		got := buildPostURL("https://example.com/", "127.0.0.1", 7330, "abc")
		if got != "https://example.com/abc" {
			t.Fatalf("unexpected url: %s", got)
		}
	})

	t.Run("falls back to host and port when public url empty", func(t *testing.T) {
		got := buildPostURL("", "localhost", 7330, "abc")
		if got != "http://localhost:7330/abc" {
			t.Fatalf("unexpected url: %s", got)
		}
	})

	t.Run("replaces 0.0.0.0 with 127.0.0.1", func(t *testing.T) {
		got := buildPostURL("", "0.0.0.0", 7330, "abc")
		if got != "http://127.0.0.1:7330/abc" {
			t.Fatalf("unexpected url: %s", got)
		}
	})
}

func TestBuildDeliveryMessage_IncludesLink(t *testing.T) {
	t.Run("includes link even when body empty", func(t *testing.T) {
		got := buildDeliveryMessage("Title", "", "https://example.com/abc", 200)
		if got != "Title\n\nhttps://example.com/abc" {
			t.Fatalf("unexpected message: %q", got)
		}
	})

	t.Run("includes link and preview", func(t *testing.T) {
		got := buildDeliveryMessage("Title", "hello world", "https://example.com/abc", 5)
		want := "Title\n\nhello…\n\nhttps://example.com/abc"
		if got != want {
			t.Fatalf("unexpected message: %q", got)
		}
	})
}
