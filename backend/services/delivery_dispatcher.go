package services

import (
	"context"
	"log"
)

type DeliveryEnqueuer interface {
	Enqueue(job DeliveryJob)
}

type DeliveryDispatcher struct {
	jobs    chan DeliveryJob
	deliver *PostDeliveryService
}

func NewDeliveryDispatcher(deliver *PostDeliveryService, bufferSize int) *DeliveryDispatcher {
	if bufferSize <= 0 {
		bufferSize = 256
	}

	return &DeliveryDispatcher{
		jobs:    make(chan DeliveryJob, bufferSize),
		deliver: deliver,
	}
}

func (d *DeliveryDispatcher) Start(ctx context.Context) {
	if ctx == nil {
		ctx = context.Background()
	}

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

func (d *DeliveryDispatcher) Enqueue(job DeliveryJob) {
	select {
	case d.jobs <- job:
	default:
		log.Printf("delivery queue full, dropping job user_id=%d post_qid=%s", job.UserID, job.PostQID)
	}
}
