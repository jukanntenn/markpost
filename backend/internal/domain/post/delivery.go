package post

// DeliveryJob is the post aggregate's delivery contract: the pure-data
// description of a post that needs to be pushed to delivery channels. It lives
// in the domain so the delivery service can consume it without a service-layer
// import (architecture.md 偏离点 #3). Fields are all basic types.
type DeliveryJob struct {
	UserID  int
	PostID  int
	PostQID string
	Title   string
	Body    string
}

// DeliveryEnqueuer is the port the post aggregate exposes for enqueueing a
// delivery job. Implemented by the delivery service's Dispatcher; consumed by
// the post service. Keeping the interface in the domain keeps the dependency
// direction service/delivery → domain/post (inward), not service/delivery →
// service/post.
type DeliveryEnqueuer interface {
	Enqueue(job DeliveryJob)
}
