// Package cmd provides CLI commands for the application.
package cmd

import (
	"context"
	"fmt"

	"markpost/internal/config"
	"markpost/internal/infra"
)

// RunPruneDeliveryHistory prunes delivery_history rows older than the configured
// retention window. It uses the portable subquery-LIMIT batched delete (bare
// DELETE ... LIMIT is a Postgres syntax error), invoked by an external cron in
// the style of RunPruneExpiredPosts.
func RunPruneDeliveryHistory(configPath string, dryRun bool, batchSize int) error {
	if err := config.Load(configPath); err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	cfg := config.Get()

	dbInstance, err := infra.New(cfg.DB.DSN, cfg.DB.Timezone)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer func() {
		if err := dbInstance.Close(); err != nil {
			fmt.Printf("warning: close database: %v\n", err)
		}
	}()

	attemptRepo := infra.NewAttemptRepository(dbInstance.DB())
	retention := cfg.Delivery.HistoryRetention

	if dryRun {
		count, err := attemptRepo.CountHistoryExpired(context.Background(), retention)
		if err != nil {
			return fmt.Errorf("failed to count expired delivery history: %w", err)
		}
		fmt.Printf("Dry run: %d delivery_history rows past their effective retention would be deleted\n", count)
		return nil
	}

	deleted, err := attemptRepo.PruneHistory(context.Background(), retention, batchSize)
	if err != nil {
		return fmt.Errorf("failed to prune delivery history: %w", err)
	}

	fmt.Printf("Pruning delivery history completed (%d deleted)\n", deleted)
	return nil
}
