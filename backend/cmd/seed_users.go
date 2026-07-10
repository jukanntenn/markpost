// Package cmd provides CLI commands for the application.
package cmd

import (
	"context"
	"fmt"
	"log"
	"os"

	"markpost/internal/config"
	"markpost/internal/domain/delivery"
	"markpost/internal/infra"
)

// RunSeedUsers creates test users (with unique post_keys via the production
// user-repo path) and optionally attaches Feishu delivery channels to each, for
// write-path load testing. The generated post_keys are printed to stdout, one
// per line, so a seed script can capture them into a target file.
func RunSeedUsers(configPath string, count int, prefix, password string, channels int, keywords string) error {
	if err := config.Load(configPath); err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	cfg := config.Get()

	dbInstance, err := infra.New(cfg.DB.DSN)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer func() {
		if err := dbInstance.Close(); err != nil {
			log.Printf("Failed to close database: %v", err)
		}
	}()

	userRepo := infra.NewUserRepository(dbInstance.DB(), cfg.PostKeyLength)
	channelRepo := infra.NewDeliveryChannelRepository(dbInstance.DB())
	ctx := context.Background()

	keys := make([]string, 0, count)
	for i := 1; i <= count; i++ {
		username := fmt.Sprintf("%s_%d", prefix, i)
		email := fmt.Sprintf("%s_%d@localhost", prefix, i)

		u, err := userRepo.Create(ctx, email, username, password)
		if err != nil {
			return fmt.Errorf("create user %s: %w", username, err)
		}
		keys = append(keys, u.PostKey)

		for c := 1; c <= channels; c++ {
			ch := &delivery.Channel{
				UserID:  u.ID,
				Kind:    delivery.ChannelKindFeishu,
				Name:    fmt.Sprintf("%s-channel-%d", username, c),
				Enabled: true,
				Configuration: delivery.ChannelConfiguration{
					"webhook_url":   "http://localhost:9999/no-op",
					"card_link_url": "",
				},
				Keywords: keywords,
			}
			if err := channelRepo.Create(ctx, ch); err != nil {
				return fmt.Errorf("create channel for user %s: %w", username, err)
			}
		}
	}

	// Print post_keys to stdout for the seed script to capture.
	for _, k := range keys {
		fmt.Println(k)
	}

	fmt.Fprintf(os.Stderr, "Seeded %d users (%d channels each)\n", count, channels)
	return nil
}
