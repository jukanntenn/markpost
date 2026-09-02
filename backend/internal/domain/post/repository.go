package post

import (
	"context"
	"time"
)

// Repository defines the interface for post data access.
type Repository interface {
	Create(ctx context.Context, title, body string, userID int) (*Post, error)
	CreateBatch(ctx context.Context, posts []Post) (int, error)
	GetByQID(ctx context.Context, qid string) (*Post, error)
	GetByID(ctx context.Context, id int) (*Post, error)
	CountByUserID(ctx context.Context, userID int) (int64, error)
	// GetByUserID retrieves a user's own posts with optional title/body search
	// (B3.3/F.5).
	GetByUserID(ctx context.Context, userID int, search string, offset int, limit int) ([]Post, error)
	CountByUserIDSearch(ctx context.Context, userID int, search string) (int64, error)
	// ListAll retrieves all posts with optional search and username filter
	// (admin; F.9).
	ListAll(ctx context.Context, search, username string, offset int, limit int) ([]Post, error)
	CountAll(ctx context.Context, search, username string) (int64, error)
	// CountSince counts posts created at or after since (stats week delta, D2.4).
	CountSince(ctx context.Context, since time.Time) (int64, error)
	DeleteByID(ctx context.Context, id int) (int64, error)
	// DeleteByQID deletes a post by its QID. When ownerID is non-zero, the row
	// is only deleted if it belongs to that owner (returns affected=0 otherwise);
	// an ownerID of 0 (admin path) deletes by QID with no owner constraint.
	DeleteByQID(ctx context.Context, qid string, ownerID int) (int64, error)
	PruneExpired(ctx context.Context, retentionDays int, batchSize int) ([]string, error)
	CountExpired(ctx context.Context, retentionDays int) (int64, error)
	// CountExpiringForUsers counts posts of the targeted users past the
	// cutoff (retention impact preview); nil cutoff matches nothing, vipOnly
	// targets every VIP user instead of explicit ids.
	CountExpiringForUsers(ctx context.Context, userIDs []int, vipOnly bool, cutoff *time.Time) (int64, error)
}
