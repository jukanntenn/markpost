// Package delivery provides delivery channel business logic and services.
package delivery

import (
	"context"
	"log"

	"markpost/internal/service/post"
)

// DeliveryHandler processes delivery jobs.
type DeliveryHandler interface {
	Deliver(ctx context.Context, job post.DeliveryJob)
}

// DeliveryDispatcher implements post.DeliveryEnqueuer with an in-process job queue.
type DeliveryDispatcher struct {
	jobs    chan post.DeliveryJob
	deliver DeliveryHandler
}

// NewDeliveryDispatcher creates a dispatcher with the given delivery service and buffer size.
func NewDeliveryDispatcher(deliver DeliveryHandler, bufferSize int) *DeliveryDispatcher {
	if bufferSize <= 0 {
		bufferSize = 256
	}

	return &DeliveryDispatcher{
		jobs:    make(chan post.DeliveryJob, bufferSize),
		deliver: deliver,
	}
}

// Start launches the background goroutine that processes delivery jobs.
func (d *DeliveryDispatcher) Start(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case job := <-d.jobs:
				d.deliver.Deliver(ctx, job)
			}
		}
	}()
}

// Enqueue pushes a delivery job onto the queue, dropping it if the queue is full.
func (d *DeliveryDispatcher) Enqueue(job post.DeliveryJob) {
	select {
	case d.jobs <- job:
	default:
		log.Printf("delivery queue full, dropping job user_id=%d post_qid=%s", job.UserID, job.PostQID)
	}
}
