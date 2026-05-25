// Package cmd provides CLI commands for the application.
package cmd

import (
	"context"
	"fmt"

	"markpost/internal/config"
	"markpost/internal/infra"
	"markpost/pkg/utils"
)

func RunResetPassword(configPath, username, password string) error {
	if err := config.Load(configPath); err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	cfg := config.Get()

	dbInstance, err := infra.New(cfg.DB.DSN)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer dbInstance.Close()

	userRepo := infra.NewUserRepository(dbInstance.DB(), cfg.PostKeyLength)
	tokenRepo := infra.NewTokenRepository(dbInstance.DB())

	u, err := userRepo.GetByUsername(context.Background(), username)
	if err != nil {
		return fmt.Errorf("user '%s' not found: %w", username, err)
	}

	generated := false
	if password == "" {
		generated = true
		pwd, err := utils.GenerateRandomPassword(16)
		if err != nil {
			return fmt.Errorf("failed to generate password: %w", err)
		}
		password = pwd
	}

	if err := userRepo.SetPassword(context.Background(), u.ID, password); err != nil {
		return fmt.Errorf("failed to reset password: %w", err)
	}

	if err := tokenRepo.DeleteRefreshTokensByUserID(context.Background(), u.ID); err != nil {
		return fmt.Errorf("failed to revoke sessions: %w", err)
	}

	if generated {
		fmt.Printf("Password reset for user '%s'. New password: %s\n", username, password)
	} else {
		fmt.Printf("Password reset for user '%s'. All sessions revoked.\n", username)
	}

	return nil
}
