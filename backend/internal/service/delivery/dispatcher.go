// Package delivery provides delivery channel business logic and services.
package delivery

import (
	"context"
	"fmt"
	"log"
	"time"

	"markpost/internal/config"
	"markpost/internal/domain/delivery"
	domainpost "markpost/internal/domain/post"
	"markpost/internal/service/delivery/filter"
	"markpost/internal/service/post"

	"github.com/alitto/pond/v2"
)

const (
	claimBatchSize     = 64
	expireBatchSize    = 64
	claimReserveBuffer = 500 * time.Millisecond
)

// AttemptRepo is the subset of delivery.AttemptRepository the dispatcher uses.
// Declared locally so tests can substitute a fake without a database.
type AttemptRepo interface {
	Create(ctx context.Context, attempts []*delivery.Attempt) error
	ClaimDue(ctx context.Context, now, reserveUntilMs int64, limit int) ([]*delivery.Attempt, error)
	MarkRetry(ctx context.Context, id int64, attempts int, lastError string, nextAtMs int64) error
	MarkExpired(ctx context.Context, wallBeforeMs int64, batchSize int) ([]*delivery.Attempt, error)
	ArchiveAndDelete(ctx context.Context, attempt *delivery.Attempt, status delivery.Status, lastError string) error
}

// ChannelRepo fetches delivery channels (for enqueue-time filtering) and
// individual channels (for the worker's send path).
type ChannelRepo interface {
	GetByUserID(ctx context.Context, userID int) ([]delivery.Channel, error)
	GetByIDAndUserID(ctx context.Context, id int, userID int) (*delivery.Channel, error)
}

// PostRepo fetches posts by ID for the worker's send path. A delivery attempt
// lives at most the expiry wall (< 7d retention), so the post is guaranteed to
// exist at delivery time.
type PostRepo interface {
	GetByID(ctx context.Context, id int) (*domainpost.Post, error)
}

// Sender issues the channel notification for a claimed attempt. It returns nil
// on success (the attempt is archived as delivered) or a non-nil error on
// failure (the dispatcher applies backoff or fails the attempt).
type Sender interface {
	Send(ctx context.Context, p *domainpost.Post, channel *delivery.Channel) error
}

// Dispatcher implements post.DeliveryEnqueuer on top of the persistent
// best-effort delivery queue: a Postgres/MySQL/SQLite-backed attempt table,
// drained by a ScanInterval-tick scheduler that claims due rows and dispatches
// them to a bounded pond v2 worker pool. Delivery is at-least-once and
// survives process restarts — all pending state lives in the database.
type Dispatcher struct {
	attemptRepo AttemptRepo
	channelRepo ChannelRepo
	postRepo    PostRepo
	sender      Sender
	cfg         config.DeliveryConfig

	pool   pond.Pool
	ticker *time.Ticker
	stop   chan struct{}
	done   chan struct{}

	now func() time.Time
}

// NewDispatcher constructs a dispatcher over the given repos and sender. The
// worker pool size and queue depth come from [delivery] config. It implements
// post.DeliveryEnqueuer: Enqueue is synchronous, best-effort, and never
// returns an error. Start must be called to launch the scheduler.
func NewDispatcher(attemptRepo AttemptRepo, channelRepo ChannelRepo, postRepo PostRepo, sender Sender) *Dispatcher {
	cfg := config.Get().Delivery
	workers := cfg.Workers
	if workers < 1 {
		workers = 32
	}
	queueSize := cfg.QueueSize
	if queueSize < 0 {
		queueSize = 1024
	}

	pool := pond.NewPool(
		workers,
		pond.WithQueueSize(queueSize),
		pond.WithNonBlocking(true),
	)

	return &Dispatcher{
		attemptRepo: attemptRepo,
		channelRepo: channelRepo,
		postRepo:    postRepo,
		sender:      sender,
		cfg:         cfg,
		pool:        pool,
		stop:        make(chan struct{}),
		done:        make(chan struct{}),
		now:         time.Now,
	}
}

// Enqueue matches a freshly-created post against the author's enabled delivery
// channels and inserts a pending attempt row per matching channel. The keyword
// filter runs before persistence, so only channels that actually match produce
// attempt rows. Enqueue is best-effort: any error is logged and swallowed so a
// delivery failure can never fail post creation.
func (d *Dispatcher) Enqueue(job post.DeliveryJob) {
	ctx := context.Background()
	channels, err := d.channelRepo.GetByUserID(ctx, job.UserID)
	if err != nil {
		log.Printf("delivery enqueue: list channels user_id=%d err=%v", job.UserID, err)
		return
	}

	now := d.now()
	attempts := make([]*delivery.Attempt, 0, len(channels))
	for _, channel := range channels {
		if !channel.Enabled {
			continue
		}
		matcher, err := filter.Compile(channel.Keywords)
		if err != nil {
			log.Printf("delivery enqueue: skip channel invalid keywords channel_id=%d user_id=%d err=%v", channel.ID, channel.UserID, err)
			continue
		}
		if !matcher.Match(job.Title) {
			continue
		}
		attempts = append(attempts, &delivery.Attempt{
			UserID:    job.UserID,
			PostID:    job.PostID,
			ChannelID: channel.ID,
			Status:    delivery.StatusPending,
			NextAt:    now.UnixMilli(),
			CreatedAt: now,
			UpdatedAt: now,
		})
	}

	if len(attempts) == 0 {
		return
	}

	if err := d.attemptRepo.Create(ctx, attempts); err != nil {
		log.Printf("delivery enqueue: create attempts user_id=%d post_qid=%s err=%v", job.UserID, job.PostQID, err)
	}
}

// Start launches the scheduler goroutine. It is safe to call once. The
// scheduler ticks every ScanInterval, sweeping the expiry wall and claiming
// due attempts into the worker pool.
func (d *Dispatcher) Start(ctx context.Context) {
	interval := d.cfg.ScanInterval
	if interval <= 0 {
		interval = time.Second
	}
	d.ticker = time.NewTicker(interval)
	go d.run(ctx)
}

func (d *Dispatcher) run(ctx context.Context) {
	defer close(d.done)
	for {
		select {
		case <-d.stop:
			return
		case <-ctx.Done():
			return
		case <-d.ticker.C:
			d.tick(ctx)
		}
	}
}

func (d *Dispatcher) tick(ctx context.Context) {
	d.sweepExpiry(ctx)
	d.claimDue(ctx)
}

// sweepExpiry transitions pending attempts past the expiry wall to expired and
// archives them, batched so a large pending backlog cannot lock the whole
// matched range in one statement.
func (d *Dispatcher) sweepExpiry(ctx context.Context) {
	wall := ExpiryWall()
	if wall <= 0 {
		return
	}
	wallBeforeMs := d.now().Add(-wall).UnixMilli()

	for {
		expired, err := d.attemptRepo.MarkExpired(ctx, wallBeforeMs, expireBatchSize)
		if err != nil {
			log.Printf("delivery sweep: mark expired err=%v", err)
			return
		}
		for _, a := range expired {
			if err := d.attemptRepo.ArchiveAndDelete(ctx, a, delivery.StatusExpired, a.LastError); err != nil {
				log.Printf("delivery sweep: archive expired attempt_id=%d err=%v", a.ID, err)
			}
		}
		if len(expired) < expireBatchSize {
			return
		}
	}
}

// claimDue claims due attempts and submits each to the worker pool. The claim
// reserves next_at past the request timeout so the next tick does not re-claim
// in-flight rows.
func (d *Dispatcher) claimDue(ctx context.Context) {
	now := d.now()
	reserveUntil := now.Add(d.cfg.RequestTimeout).Add(claimReserveBuffer)

	claimed, err := d.attemptRepo.ClaimDue(ctx, now.UnixMilli(), reserveUntil.UnixMilli(), claimBatchSize)
	if err != nil {
		log.Printf("delivery claim: err=%v", err)
		return
	}

	for _, a := range claimed {
		attempt := a
		if err := d.pool.Go(func() { d.execute(ctx, attempt) }); err != nil {
			log.Printf("delivery claim: pool dropped attempt_id=%d err=%v", attempt.ID, err)
		}
	}
}

func (d *Dispatcher) execute(ctx context.Context, a *delivery.Attempt) {
	p, err := d.postRepo.GetByID(ctx, a.PostID)
	if err != nil {
		d.handleSendError(ctx, a, fmt.Errorf("get post id=%d: %w", a.PostID, err))
		return
	}
	channel, err := d.channelRepo.GetByIDAndUserID(ctx, a.ChannelID, a.UserID)
	if err != nil {
		d.handleSendError(ctx, a, fmt.Errorf("get channel id=%d: %w", a.ChannelID, err))
		return
	}

	if err := d.sender.Send(ctx, p, channel); err != nil {
		d.handleSendError(ctx, a, err)
		return
	}

	if err := d.attemptRepo.ArchiveAndDelete(ctx, a, delivery.StatusDelivered, ""); err != nil {
		log.Printf("delivery execute: archive delivered attempt_id=%d err=%v", a.ID, err)
	}
}

// handleSendError applies the backoff policy to a failed attempt: if the
// sequence is exhausted the attempt is archived as failed, otherwise the
// attempt count is bumped and next_at is advanced by the next backoff step.
func (d *Dispatcher) handleSendError(ctx context.Context, a *delivery.Attempt, sendErr error) {
	nextAttempts := a.Attempts + 1
	lastError := truncateError(sendErr.Error())

	backoff, ok := NextBackoff(nextAttempts - 1)
	if !ok {
		if err := d.attemptRepo.ArchiveAndDelete(ctx, a, delivery.StatusFailed, lastError); err != nil {
			log.Printf("delivery execute: archive failed attempt_id=%d err=%v", a.ID, err)
		}
		return
	}

	nextAt := d.now().Add(backoff).UnixMilli()
	if err := d.attemptRepo.MarkRetry(ctx, a.ID, nextAttempts, lastError, nextAt); err != nil {
		log.Printf("delivery execute: mark retry attempt_id=%d err=%v", a.ID, err)
	}
}

// Stop signals the scheduler to stop and waits for the worker pool to drain.
// It is idempotent.
func (d *Dispatcher) Stop() {
	select {
	case <-d.stop:
		return
	default:
		close(d.stop)
	}
	if d.ticker != nil {
		d.ticker.Stop()
	}
	<-d.done
	d.pool.StopAndWait()
}

func truncateError(s string) string {
	const max = 200
	if len(s) <= max {
		return s
	}
	return s[:max]
}

var _ post.DeliveryEnqueuer = (*Dispatcher)(nil)
