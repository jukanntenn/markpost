package cmd

import (
	"fmt"
	"log"

	"markpost/internal/config"
	"markpost/internal/infra"
)

func RunMigrateUp(configPath string) error {
	if err := config.Load(configPath); err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	return infra.MigrateUp(config.Get().DB.DSN)
}

func RunMigrateDown(configPath string, steps int) error {
	if err := config.Load(configPath); err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	return infra.MigrateDown(config.Get().DB.DSN, steps)
}

func RunMigrateForce(configPath string, version int) error {
	if err := config.Load(configPath); err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	return infra.MigrateForce(config.Get().DB.DSN, version)
}

func RunMigrateVersion(configPath string) error {
	if err := config.Load(configPath); err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	v, dirty, err := infra.MigrateVersion(config.Get().DB.DSN)
	if err != nil {
		return err
	}
	log.Printf("version=%d dirty=%v", v, dirty)
	return nil
}
