package cmd

import (
	"fmt"

	"markpost/conf"
	"markpost/models"
	"markpost/repositories"
)

func RunPruneExpiredPosts(configPath string, dryRun bool, batchSize int) error {
	if err := conf.LoadConfig(configPath); err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	config := conf.Conf()

	database, err := models.NewDatabase(config.DB.DSN)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}

	retentionDays := config.Post.RetentionDays

	postRepo := repositories.NewPostRepo(database)

	if dryRun {
		count, err := postRepo.CountExpiredPosts(retentionDays)
		if err != nil {
			return fmt.Errorf("failed to count expired posts: %w", err)
		}

		fmt.Printf("Dry run: %d posts older than %d days would be deleted\n", count, retentionDays)
		return nil
	}

	if err := postRepo.PruneExpiredPosts(retentionDays, batchSize); err != nil {
		return fmt.Errorf("failed to cleanup expired posts: %w", err)
	}

	fmt.Println("Pruning expired posts completed")
	return nil
}
