package main

import (
	"fmt"
	"os"

	"jimu/internal/app"
	apikeycli "jimu/internal/capabilities/apikey/cli"
	"jimu/internal/config"
	"jimu/internal/kernel/db"
	"jimu/internal/kernel/logger"
	"jimu/tools/generator"

	"github.com/spf13/cobra"
)

// version 版本号，通过 ldflags 注入：-ldflags "-X main.version=v0.1.0"
var version = "dev"

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
		caps, err := activeDescriptors()
		if err != nil {
			return err
		}
		if err := db.MigrateWithRetry(cfg.DB, caps, log, "up"); err != nil {
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
		caps, err := activeDescriptors()
		if err != nil {
			return err
		}
		if err := db.MigrateWithRetry(cfg.DB, caps, log, "down"); err != nil {
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
		caps, err := activeDescriptors()
		if err != nil {
			return err
		}
		if err := db.MigrateWithRetry(cfg.DB, caps, log, "status"); err != nil {
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
		caps, err := activeDescriptors()
		if err != nil {
			return err
		}
		if err := db.MigrateWithRetry(cfg.DB, caps, log, "redo"); err != nil {
			return fmt.Errorf("redo failed: %w", err)
		}
		fmt.Println("Redo successful")
		return nil
	},
}

var migrateAdoptCmd = &cobra.Command{
	Use:   "adopt-capabilities",
	Short: "Register per-capability version baselines for an existing database",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		caps, err := activeDescriptors()
		if err != nil {
			return err
		}
		baseline, err := db.AdoptCapabilities(cfg.DB, caps)
		if err != nil {
			return fmt.Errorf("adopt failed: %w", err)
		}
		for _, d := range caps {
			if vs, ok := baseline[d.Name]; ok {
				fmt.Printf("%s: %d migrations adopted\n", d.Name, len(vs))
			}
		}
		return nil
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(version)
	},
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configuration commands",
}

var configCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Load and validate configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("config invalid: %w", err)
		}
		fmt.Printf("config valid (env=%s, http.mode=%s, db=%s:%d/%s)\n",
			os.Getenv("APP_ENV"), cfg.HTTP.Mode, cfg.DB.Host, cfg.DB.Port, cfg.DB.Database)
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
		if err := db.InitSnowflake(cfg.ID.WorkerID); err != nil {
			return fmt.Errorf("failed to init snowflake: %w", err)
		}
		dbConn, err := db.ConnectWithRetry(cfg.DB, log)
		if err != nil {
			return fmt.Errorf("failed to connect database: %w", err)
		}
		caps, err := activeDescriptors()
		if err != nil {
			return err
		}
		if err := app.RunSeedWithCasbin(dbConn, caps); err != nil {
			return fmt.Errorf("seed failed: %w", err)
		}
		fmt.Println("Seed data inserted successfully (with Casbin policies)")
		return nil
	},
}

func init() {
	moduleCmd.AddCommand(moduleCreateCmd)
	migrateCmd.AddCommand(migrateUpCmd)
	migrateCmd.AddCommand(migrateDownCmd)
	migrateCmd.AddCommand(migrateStatusCmd)
	migrateCmd.AddCommand(migrateRedoCmd)
	migrateCmd.AddCommand(migrateAdoptCmd)
	rootCmd.AddCommand(moduleCmd)
	rootCmd.AddCommand(migrateCmd)
	rootCmd.AddCommand(seedCmd)
	rootCmd.AddCommand(versionCmd)
	configCmd.AddCommand(configCheckCmd)
	rootCmd.AddCommand(configCmd)
	// 能力自带的命令（P2.6 接缝）：能力在自己的包内提供 Commands()，此处显式注册。
	rootCmd.AddCommand(apikeycli.Commands()...)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
