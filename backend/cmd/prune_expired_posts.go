package cmd

import (
	"fmt"

	"markpost/internal/config"
	"markpost/internal/infra/database"
)

func RunPruneExpiredPosts(configPath string, dryRun bool, batchSize int) error {
	if err := config.Load(configPath); err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	cfg := config.Get()

	dbInstance, err := database.New(cfg.DB.DSN)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}

	retentionDays := cfg.Post.RetentionDays

	postRepo := database.NewPostRepository(dbInstance.DB())

	if dryRun {
		count, err := postRepo.CountExpired(nil, retentionDays)
		if err != nil {
			return fmt.Errorf("failed to count expired posts: %w", err)
		}

		fmt.Printf("Dry run: %d posts older than %d days would be deleted\n", count, retentionDays)
		return nil
	}

	if err := postRepo.PruneExpired(nil, retentionDays, batchSize); err != nil {
		return fmt.Errorf("failed to cleanup expired posts: %w", err)
	}

	fmt.Println("Pruning expired posts completed")
	return nil
}
