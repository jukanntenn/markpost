package delivery

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"markpost/internal/domain/delivery"
	domainpost "markpost/internal/domain/post"
	"markpost/internal/infra"
)

// recordingMetrics captures the dispatcher's metric calls so the fast-fail test
// can assert the category label was forwarded.
type recordingMetrics struct {
	pendingDelta     int64
	failedCategories []string
	dispatched       int
}

func (r *recordingMetrics) AddDeliveryPending(_ context.Context, delta int64) {
	r.pendingDelta += delta
}
func (r *recordingMetrics) IncDeliveryDispatched(context.Context) { r.dispatched++ }
func (r *recordingMetrics) IncDeliveryFailed(_ context.Context, category string) {
	r.failedCategories = append(r.failedCategories, category)
}

// TestPostDeliveryService_SendStripsImagesFromCard is the end-to-end regression
// for the Feishu "invalid image keys" (ErrCode 200570) failure: a post body
// containing inline image markdown must reach the webhook as a card whose
// markdown element has no image syntax.
func TestPostDeliveryService_SendStripsImagesFromCard(t *testing.T) {
	loadDeliveryTestConfig(t)

	var receivedBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":0}`))
	}))
	defer server.Close()

	channel := &delivery.Channel{
		UserID:        1,
		Kind:          delivery.ChannelKindFeishu,
		Name:          "Test",
		Enabled:       true,
		Configuration: makeFeishuChannelConfig(server.URL),
	}
	p := &domainpost.Post{
		ID: 1, QID: "p-img", Title: "Img Post",
		Body: "intro text\n\n![](http://img.zuanke8.cn/forum/202608/12/223429wjdsgrej3de8lecb.jpg)\n\nmore text",
	}

	svc := &PostDeliveryService{feishu: NewFeishuClient(5 * time.Second)}
	if err := svc.Send(context.Background(), p, channel); err != nil {
		t.Fatalf("Send error: %v", err)
	}

	card := mustAs[map[string]any](t, receivedBody["card"], "card")
	body := mustAs[map[string]any](t, card["body"], "card.body")
	elements := mustAs[[]any](t, body["elements"], "card.body.elements")

	foundMarkdown := false
	for _, elem := range elements {
		el := mustAs[map[string]any](t, elem, "element")
		if el["tag"] != "markdown" {
			continue
		}
		foundMarkdown = true
		content, _ := el["content"].(string)
		if strings.Contains(content, "![") {
			t.Errorf("card markdown still contains image syntax: %q", content)
		}
		if !strings.Contains(content, "intro text") || !strings.Contains(content, "more text") {
			t.Errorf("card markdown lost surrounding text: %q", content)
		}
	}
	if !foundMarkdown {
		t.Fatal("expected a markdown element in the card body")
	}
}

// TestDispatcher_HandleSendError asserts the classification-driven retry gate:
// a non-retryable error (card_rejected) archives as failed immediately with the
// category recorded, while a retryable error (network) schedules a retry. Each
// subtest takes its own SetupTestDB so the hardcoded seed user does not collide.
func TestDispatcher_HandleSendError(t *testing.T) {
	loadDeliveryTestConfig(t)
	ctx := context.Background()

	t.Run("non_retryable_archives_failed_with_category", func(t *testing.T) {
		db := infra.SetupTestDB(t)
		attemptRepo := infra.NewAttemptRepository(db)
		met := &recordingMetrics{}
		d := NewDispatcher(
			attemptRepo,
			infra.NewDeliveryChannelRepository(db),
			infra.NewPostRepository(db),
			recordingSender{},
			WithMetrics(met),
		)

		uid, pid, cid := seedUserPostChannel(t, db)
		now := time.Now()
		attempt := &delivery.Attempt{
			UserID: uid, PostID: pid, ChannelID: cid, Status: delivery.StatusPending,
			NextAt: now.UnixMilli(), CreatedAt: now, UpdatedAt: now,
		}
		if err := attemptRepo.Create(ctx, []*delivery.Attempt{attempt}); err != nil {
			t.Fatalf("create attempt: %v", err)
		}

		d.handleSendError(ctx, attempt, newDeliveryError(
			CategoryCardRejected, false,
			errors.New("feishu api code=11246 msg=ErrCode: 200570; invalid image keys"),
		))

		// Attempt row is gone (archived + deleted).
		var n int64
		db.Model(&delivery.Attempt{}).Where("id = ?", attempt.ID).Count(&n)
		if n != 0 {
			t.Errorf("expected attempt archived, %d still present", n)
		}

		// History row carries the category.
		var hist delivery.History
		if err := db.Where("user_id = ?", uid).Order("id DESC").First(&hist).Error; err != nil {
			t.Fatalf("expected history row: %v", err)
		}
		if hist.Status != delivery.StatusFailed {
			t.Errorf("history status=%d want failed(%d)", hist.Status, delivery.StatusFailed)
		}
		if hist.ErrorCategory != string(CategoryCardRejected) {
			t.Errorf("history error_category=%q want %q", hist.ErrorCategory, CategoryCardRejected)
		}

		// Metric got the category label and pending decremented.
		if len(met.failedCategories) != 1 || met.failedCategories[0] != string(CategoryCardRejected) {
			t.Errorf("failed categories=%v want [card_rejected]", met.failedCategories)
		}
		if met.pendingDelta != -1 {
			t.Errorf("pending delta=%d want -1", met.pendingDelta)
		}
	})

	t.Run("retryable_schedules_retry", func(t *testing.T) {
		db := infra.SetupTestDB(t)
		attemptRepo := infra.NewAttemptRepository(db)
		met := &recordingMetrics{}
		d := NewDispatcher(
			attemptRepo,
			infra.NewDeliveryChannelRepository(db),
			infra.NewPostRepository(db),
			recordingSender{},
			WithMetrics(met),
		)

		uid, pid, cid := seedUserPostChannel(t, db)
		now := time.Now()
		attempt := &delivery.Attempt{
			UserID: uid, PostID: pid, ChannelID: cid, Status: delivery.StatusPending,
			NextAt: now.UnixMilli(), CreatedAt: now, UpdatedAt: now,
		}
		if err := attemptRepo.Create(ctx, []*delivery.Attempt{attempt}); err != nil {
			t.Fatalf("create attempt: %v", err)
		}

		d.handleSendError(ctx, attempt, newDeliveryError(CategoryNetwork, true, errors.New("dial timeout")))

		// Attempt still present, attempts bumped, next_at advanced into the future.
		var got delivery.Attempt
		if err := db.Where("id = ?", attempt.ID).First(&got).Error; err != nil {
			t.Fatalf("expected attempt to remain after retryable error: %v", err)
		}
		if got.Attempts != 1 {
			t.Errorf("attempts=%d want 1", got.Attempts)
		}
		if got.NextAt <= now.UnixMilli() {
			t.Errorf("next_at not advanced into the future")
		}
		// No terminal failure recorded yet.
		if len(met.failedCategories) != 0 {
			t.Errorf("retryable error should not record failure yet: %v", met.failedCategories)
		}
	})
}

// TestListHistoryErrorCategoryFilter verifies the admin filter dimension.
func TestListHistoryErrorCategoryFilter(t *testing.T) {
	loadDeliveryTestConfig(t)
	db := infra.SetupTestDB(t)
	ctx := context.Background()
	repo := infra.NewAttemptRepository(db)

	uid, pid, cid := seedUserPostChannel(t, db)
	now := time.Now()
	a := &delivery.Attempt{
		UserID: uid, PostID: pid, ChannelID: cid, Status: delivery.StatusPending,
		NextAt: now.UnixMilli(), CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.Create(ctx, []*delivery.Attempt{a}); err != nil {
		t.Fatalf("create attempt: %v", err)
	}
	if err := repo.ArchiveAndDelete(ctx, a, delivery.StatusFailed, "boom", "card_rejected"); err != nil {
		t.Fatalf("archive: %v", err)
	}

	rows, err := repo.ListHistory(ctx, delivery.HistoryFilter{ErrorCategory: "card_rejected"}, 0, 10)
	if err != nil {
		t.Fatalf("ListHistory card_rejected: %v", err)
	}
	if len(rows) != 1 {
		t.Errorf("card_rejected filter rows=%d want 1", len(rows))
	}

	rows, err = repo.ListHistory(ctx, delivery.HistoryFilter{ErrorCategory: "network"}, 0, 10)
	if err != nil {
		t.Fatalf("ListHistory network: %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("network filter rows=%d want 0", len(rows))
	}

	total, err := repo.CountHistory(ctx, delivery.HistoryFilter{})
	if err != nil {
		t.Fatalf("CountHistory: %v", err)
	}
	if total != 1 {
		t.Errorf("total=%d want 1", total)
	}
}
