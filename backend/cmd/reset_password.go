// Package cmd provides CLI commands for the application.
package cmd

import (
	"context"
	"fmt"
	"log"

	"markpost/internal/config"
	"markpost/internal/infra"
	"markpost/pkg/utils"
)

// RunResetPassword resets the password for the given user and revokes all active sessions.
func RunResetPassword(configPath, username, password string) error {
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
			log.Printf("Failed to close database: %v", err)
		}
	}()

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

	// C2.3 统一预检：CLI 路径同样遵守密码策略（显式传入的密码）。
	if err := utils.ValidatePasswordPolicy(password); err != nil {
		return fmt.Errorf("password violates policy: %w", err)
	}

	if err := userRepo.SetPassword(context.Background(), u.ID, password); err != nil {
		return fmt.Errorf("failed to reset password: %w", err)
	}

	// C2.6 统一失效原语：重置密码同样自增 token_version，立即踢掉存量会话。
	if err := userRepo.BumpTokenVersion(context.Background(), u.ID); err != nil {
		return fmt.Errorf("failed to bump token version: %w", err)
	}

	if err := tokenRepo.RevokeAllByUserID(context.Background(), u.ID); err != nil {
		return fmt.Errorf("failed to revoke sessions: %w", err)
	}

	if generated {
		fmt.Printf("Password reset for user '%s'. New password: %s\n", username, password)
	} else {
		fmt.Printf("Password reset for user '%s'. All sessions revoked.\n", username)
	}

	return nil
}
