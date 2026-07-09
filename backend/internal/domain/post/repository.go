package post

import "context"

// Repository defines the interface for post data access.
type Repository interface {
	Create(ctx context.Context, title, body string, userID int) (*Post, error)
	CreateBatch(ctx context.Context, posts []Post) (int, error)
	GetByQID(ctx context.Context, qid string) (*Post, error)
	GetByID(ctx context.Context, id int) (*Post, error)
	CountByUserID(ctx context.Context, userID int) (int64, error)
	GetByUserID(ctx context.Context, userID int, offset int, limit int) ([]Post, error)
	ListAll(ctx context.Context, search string, offset int, limit int) ([]Post, error)
	CountAll(ctx context.Context, search string) (int64, error)
	DeleteByID(ctx context.Context, id int) (int64, error)
	// DeleteByQID deletes a post by its QID. When ownerID is non-zero, the row
	// is only deleted if it belongs to that owner (returns affected=0 otherwise);
	// an ownerID of 0 (admin path) deletes by QID with no owner constraint.
	DeleteByQID(ctx context.Context, qid string, ownerID int) (int64, error)
	PruneExpired(ctx context.Context, retentionDays int, batchSize int) ([]string, error)
	CountExpired(ctx context.Context, retentionDays int) (int64, error)
}
