package delivery

import "context"

// Repository defines the interface for delivery channel data access.
type Repository interface {
	GetByUserID(ctx context.Context, userID int) ([]Channel, error)
	GetByIDAndUserID(ctx context.Context, id int, userID int) (*Channel, error)
	Create(ctx context.Context, channel *Channel) error
	Update(ctx context.Context, channel *Channel) error
	DeleteByIDAndUserID(ctx context.Context, id int, userID int) (int64, error)
	ListAll(ctx context.Context, offset, limit int) ([]Channel, error)
	CountAll(ctx context.Context) (int64, error)
}
