package main

import (
	"fmt"
	"os"

	"jimu/internal/config"
	"jimu/internal/platform/db"
	"jimu/internal/platform/logger"
	"jimu/tools/generator"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "jimu",
	Short: "Jimu backend framework CLI",
}

var moduleCmd = &cobra.Command{
	Use:   "module",
	Short: "Module management commands",
}

var moduleCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a new module skeleton",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		return generator.GenerateModule(name)
	},
}

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Database migration commands",
}

var migrateUpCmd = &cobra.Command{
	Use:   "up",
	Short: "Run all pending migrations",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		log := logger.New(cfg.Log)
		if err := db.MigrateWithRetry(cfg.DB, log, "up"); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
		fmt.Println("Migrations applied successfully")
		return nil
	},
}

var migrateDownCmd = &cobra.Command{
	Use:   "down",
	Short: "Rollback last migration",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		log := logger.New(cfg.Log)
		if err := db.MigrateWithRetry(cfg.DB, log, "down"); err != nil {
			return fmt.Errorf("rollback failed: %w", err)
		}
		fmt.Println("Rollback successful")
		return nil
	},
}

var migrateStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show migration status",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		log := logger.New(cfg.Log)
		if err := db.MigrateWithRetry(cfg.DB, log, "status"); err != nil {
			return fmt.Errorf("failed to get status: %w", err)
		}
		return nil
	},
}

var migrateRedoCmd = &cobra.Command{
	Use:   "redo",
	Short: "Redo last migration",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		log := logger.New(cfg.Log)
		if err := db.MigrateWithRetry(cfg.DB, log, "redo"); err != nil {
			return fmt.Errorf("redo failed: %w", err)
		}
		fmt.Println("Redo successful")
		return nil
	},
}

var seedCmd = &cobra.Command{
	Use:   "seed",
	Short: "Seed initial data",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		log := logger.New(cfg.Log)
		dbConn, err := db.ConnectWithRetry(cfg.DB, log)
		if err != nil {
			return fmt.Errorf("failed to connect database: %w", err)
		}
		if err := db.RunSeed(dbConn); err != nil {
			return fmt.Errorf("seed failed: %w", err)
		}
		fmt.Println("Seed data inserted successfully")
		return nil
	},
}

func init() {
	moduleCmd.AddCommand(moduleCreateCmd)
	migrateCmd.AddCommand(migrateUpCmd)
	migrateCmd.AddCommand(migrateDownCmd)
	migrateCmd.AddCommand(migrateStatusCmd)
	migrateCmd.AddCommand(migrateRedoCmd)
	rootCmd.AddCommand(moduleCmd)
	rootCmd.AddCommand(migrateCmd)
	rootCmd.AddCommand(seedCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
