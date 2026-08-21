package backup

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/shridarpatil/whatomate/internal/config"
	"github.com/shridarpatil/whatomate/internal/storage"
)

const backupPrefix = "backups/postgres"

// RunPostgresBackup creates a compressed PostgreSQL custom-format dump and
// uploads it to the application's configured S3-compatible bucket.
func RunPostgresBackup(ctx context.Context, cfg *config.Config, now time.Time) (string, error) {
	if err := validateBackupConfig(cfg); err != nil {
		return "", err
	}

	tempDir, err := os.MkdirTemp("", "vyapari-nestam-postgres-backup-")
	if err != nil {
		return "", fmt.Errorf("create backup temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	key := BackupObjectKey(cfg.Database.Name, now)
	dumpPath := filepath.Join(tempDir, "database.dump")
	cmd := postgresCommand(ctx, "pg_dump", &cfg.Database,
		"--format=custom", "--no-owner", "--no-privileges", "--file", dumpPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("pg_dump failed: %w: %s", err, strings.TrimSpace(string(output)))
	}

	file, err := os.Open(dumpPath)
	if err != nil {
		return "", fmt.Errorf("open generated backup: %w", err)
	}
	defer file.Close()

	s3Client, err := storage.NewS3Client(&cfg.Storage)
	if err != nil {
		return "", fmt.Errorf("configure backup storage: %w", err)
	}
	if err := s3Client.Upload(ctx, key, file, "application/octet-stream"); err != nil {
		return "", fmt.Errorf("upload backup: %w", err)
	}
	return key, nil
}

// RestorePostgresBackup restores a backup into an existing, empty target
// database. The target must be different from the configured live database.
func RestorePostgresBackup(ctx context.Context, cfg *config.Config, key, targetDatabase string) error {
	if err := validateBackupConfig(cfg); err != nil {
		return err
	}
	if strings.TrimSpace(key) == "" {
		return fmt.Errorf("backup object key is required")
	}
	if strings.TrimSpace(targetDatabase) == "" || targetDatabase == cfg.Database.Name {
		return fmt.Errorf("target database must be a separate, clean database")
	}

	targetCfg := cfg.Database
	targetCfg.Name = targetDatabase
	if err := ensureDatabaseIsEmpty(ctx, &targetCfg); err != nil {
		return err
	}

	tempDir, err := os.MkdirTemp("", "vyapari-nestam-postgres-restore-")
	if err != nil {
		return fmt.Errorf("create restore temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	s3Client, err := storage.NewS3Client(&cfg.Storage)
	if err != nil {
		return fmt.Errorf("configure backup storage: %w", err)
	}
	body, err := s3Client.Download(ctx, key)
	if err != nil {
		return fmt.Errorf("download backup: %w", err)
	}
	defer body.Close()

	dumpPath := filepath.Join(tempDir, "database.dump")
	dumpFile, err := os.Create(dumpPath)
	if err != nil {
		return fmt.Errorf("create restore file: %w", err)
	}
	if _, err := io.Copy(dumpFile, body); err != nil {
		dumpFile.Close()
		return fmt.Errorf("save backup locally: %w", err)
	}
	if err := dumpFile.Close(); err != nil {
		return fmt.Errorf("close restore file: %w", err)
	}

	cmd := postgresCommand(ctx, "pg_restore", &targetCfg,
		"--no-owner", "--no-privileges", "--exit-on-error", dumpPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("pg_restore failed: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func BackupObjectKey(databaseName string, now time.Time) string {
	name := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return '-'
	}, databaseName)
	return fmt.Sprintf("%s/%s/%s-%s.dump", backupPrefix, now.UTC().Format("2006/01/02"), name, now.UTC().Format("20060102T150405Z"))
}

func validateBackupConfig(cfg *config.Config) error {
	if cfg == nil {
		return fmt.Errorf("config is required")
	}
	if cfg.Database.Host == "" || cfg.Database.User == "" || cfg.Database.Name == "" {
		return fmt.Errorf("database host, user, and name are required")
	}
	if cfg.Storage.S3Bucket == "" || cfg.Storage.S3Region == "" {
		return fmt.Errorf("storage.s3_bucket and storage.s3_region are required for backups")
	}
	return nil
}

func ensureDatabaseIsEmpty(ctx context.Context, cfg *config.DatabaseConfig) error {
	query := "SELECT count(*) FROM pg_tables WHERE schemaname = 'public';"
	cmd := postgresCommand(ctx, "psql", cfg, "--tuples-only", "--no-align", "--command", query)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("verify target database is empty: %w: %s", err, strings.TrimSpace(string(output)))
	}
	if strings.TrimSpace(string(output)) != "0" {
		return fmt.Errorf("target database %q is not empty", cfg.Name)
	}
	return nil
}

func postgresCommand(ctx context.Context, name string, cfg *config.DatabaseConfig, args ...string) *exec.Cmd {
	commandArgs := []string{"--host", cfg.Host, "--port", fmt.Sprint(cfg.Port), "--username", cfg.User, "--dbname", cfg.Name}
	commandArgs = append(commandArgs, args...)
	cmd := exec.CommandContext(ctx, name, commandArgs...)
	cmd.Env = append(os.Environ(), "PGPASSWORD="+cfg.Password)
	return cmd
}
