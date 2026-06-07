// Package delivery provides delivery channel business logic and services.
package delivery

import (
	"context"
	"log"

	"markpost/internal/service/post"
)

// Handler processes delivery jobs.
type Handler interface {
	Deliver(ctx context.Context, job post.DeliveryJob)
}

// Dispatcher implements post.DeliveryEnqueuer with an in-process job queue.
type Dispatcher struct {
	jobs    chan post.DeliveryJob
	deliver Handler
}

// NewDispatcher creates a dispatcher with the given delivery service and buffer size.
func NewDispatcher(deliver Handler, bufferSize int) *Dispatcher {
	if bufferSize <= 0 {
		bufferSize = 256
	}

	return &Dispatcher{
		jobs:    make(chan post.DeliveryJob, bufferSize),
		deliver: deliver,
	}
}

// Start launches the background goroutine that processes delivery jobs.
func (d *Dispatcher) Start(ctx context.Context) {
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
func (d *Dispatcher) Enqueue(job post.DeliveryJob) {
	select {
	case d.jobs <- job:
	default:
		log.Printf("delivery queue full, dropping job user_id=%d post_qid=%s", job.UserID, job.PostQID)
	}
}
