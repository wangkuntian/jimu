package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"jimu/internal/config"
	"jimu/internal/platform/db"
	"jimu/internal/platform/logger"
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

var (
	errBackupOutRequired   = errors.New("--out is required")
	errRestoreInRequired   = errors.New("--in is required")
	errRestoreNotConfirmed = errors.New("restore overwrites existing data, pass --yes to confirm")
)

var (
	backupOut        string
	backupTimeoutSec int
	restoreIn        string
	restoreYes       bool
	restoreTimeout   int
)

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Dump the database to a plain SQL file",
	Long: "Dump the configured database to a plain SQL file.\n" +
		"Requires mysqldump (mysql) or pg_dump (postgres) in PATH; run inside the server container\n" +
		"if the host has no client tools. The password is passed via MYSQL_PWD / PGPASSWORD.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if backupOut == "" {
			return errBackupOutRequired
		}
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		file, err := os.Create(backupOut)
		if err != nil {
			return fmt.Errorf("failed to create backup file: %w", err)
		}
		defer func() { _ = file.Close() }()

		spec, err := db.BuildBackup(cfg.DB, file)
		if err != nil {
			return err
		}
		spec.Stderr = os.Stderr
		fmt.Printf("running: %s\n", spec.RedactedCommand())

		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(backupTimeoutSec)*time.Second)
		defer cancel()
		if err := spec.Run(ctx); err != nil {
			_ = os.Remove(backupOut) // 失败的 dump 不留半截文件
			return err
		}
		if err := file.Sync(); err != nil {
			return fmt.Errorf("failed to flush backup file: %w", err)
		}
		info, err := file.Stat()
		if err != nil {
			return fmt.Errorf("failed to stat backup file: %w", err)
		}
		fmt.Printf("backup written: %s (%d bytes)\n", backupOut, info.Size())
		return nil
	},
}

var restoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "Restore the database from a dump file (destructive)",
	Long: "Restore the configured database from a plain SQL dump produced by `jimu backup`.\n" +
		"This overwrites existing data, so it requires --yes. Requires the mysql client or psql in PATH.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if restoreIn == "" {
			return errRestoreInRequired
		}
		if !restoreYes {
			return errRestoreNotConfirmed
		}
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		file, err := os.Open(restoreIn)
		if err != nil {
			return fmt.Errorf("failed to open dump file: %w", err)
		}
		defer func() { _ = file.Close() }()

		spec, err := db.BuildRestore(cfg.DB, file)
		if err != nil {
			return err
		}
		spec.Stdout = os.Stdout
		spec.Stderr = os.Stderr
		fmt.Printf("running: %s < %s\n", spec.RedactedCommand(), restoreIn)

		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(restoreTimeout)*time.Second)
		defer cancel()
		if err := spec.Run(ctx); err != nil {
			return err
		}
		fmt.Println("restore finished")
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
		if err := db.RunSeedWithCasbin(dbConn); err != nil {
			return fmt.Errorf("seed failed: %w", err)
		}
		fmt.Println("Seed data inserted successfully (with Casbin policies)")
		return nil
	},
}

func init() {
	backupCmd.Flags().StringVar(&backupOut, "out", "", "output SQL file (required)")
	backupCmd.Flags().IntVar(&backupTimeoutSec, "timeout-sec", 1800, "dump timeout in seconds")
	restoreCmd.Flags().StringVar(&restoreIn, "in", "", "input SQL file (required)")
	restoreCmd.Flags().BoolVar(&restoreYes, "yes", false, "confirm the destructive restore")
	restoreCmd.Flags().IntVar(&restoreTimeout, "timeout-sec", 1800, "restore timeout in seconds")
	rootCmd.AddCommand(backupCmd)
	rootCmd.AddCommand(restoreCmd)

	moduleCmd.AddCommand(moduleCreateCmd)
	migrateCmd.AddCommand(migrateUpCmd)
	migrateCmd.AddCommand(migrateDownCmd)
	migrateCmd.AddCommand(migrateStatusCmd)
	migrateCmd.AddCommand(migrateRedoCmd)
	rootCmd.AddCommand(moduleCmd)
	rootCmd.AddCommand(migrateCmd)
	rootCmd.AddCommand(seedCmd)
	rootCmd.AddCommand(versionCmd)
	configCmd.AddCommand(configCheckCmd)
	rootCmd.AddCommand(configCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
