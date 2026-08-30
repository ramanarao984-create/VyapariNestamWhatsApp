package database

import (
	"database/sql"
	"embed"
	"fmt"

	"github.com/pressly/goose/v3"
	"gorm.io/gorm"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

// RunVersionedMigrations applies committed SQL migrations from the embedded
// migration directory. Embedding keeps the production image self-contained.
func RunVersionedMigrations(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("database is nil")
	}
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("get SQL database for migrations: %w", err)
	}
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("configure migration dialect: %w", err)
	}
	if err := baselineLegacySchema(sqlDB); err != nil {
		return fmt.Errorf("baseline legacy schema: %w", err)
	}
	goose.SetBaseFS(migrationFS)
	if err := goose.Up(sqlDB, "migrations"); err != nil {
		return fmt.Errorf("apply versioned migrations: %w", err)
	}
	return nil
}

// baselineLegacySchema bridges installations created before versioned SQL
// migrations existed. It never alters an existing application table: when the
// core organizations table exists but Goose history does not, migration 00001
// is recorded as the already-present baseline and only later migrations run.
func baselineLegacySchema(db *sql.DB) error {
	var organizations sql.NullString
	if err := db.QueryRow("SELECT to_regclass('public.organizations')").Scan(&organizations); err != nil {
		return err
	}
	if !organizations.Valid {
		return nil
	}
	if _, err := goose.EnsureDBVersion(db); err != nil {
		return err
	}
	var initialApplied bool
	if err := db.QueryRow("SELECT EXISTS (SELECT 1 FROM goose_db_version WHERE version_id = $1 AND is_applied = true)", 1).Scan(&initialApplied); err != nil {
		return err
	}
	if initialApplied {
		return nil
	}
	if _, err := db.Exec("INSERT INTO goose_db_version (version_id, is_applied) VALUES ($1, true)", 1); err != nil {
		return err
	}
	return nil
}

// RollbackLastMigration rolls back exactly one migration for an operator.
func RollbackLastMigration(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("database is nil")
	}
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("get SQL database for rollback: %w", err)
	}
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("configure migration dialect: %w", err)
	}
	goose.SetBaseFS(migrationFS)
	if err := goose.Down(sqlDB, "migrations"); err != nil {
		return fmt.Errorf("rollback last migration: %w", err)
	}
	return nil
}
