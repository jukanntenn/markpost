// Package settings provides the runtime settings domain.
package settings

import "context"

// Repository defines the interface for settings data access.
type Repository interface {
	// Get returns the setting row for key, or domain.ErrNotFound.
	Get(ctx context.Context, key string) (*Setting, error)
	// GetAll returns every setting row, ordered by key.
	GetAll(ctx context.Context) ([]Setting, error)
	// Set upserts the value for key, recording the acting admin.
	Set(ctx context.Context, key string, value SettingValue, updatedBy int) error
	// VIPStrategyEnabled reports whether the GitHub-login VIP grant strategy
	// is on. It doubles as the auth service's read port; callers decide how
	// to degrade when it errors (the login path fails toward not-granting).
	VIPStrategyEnabled(ctx context.Context) (bool, error)
}
