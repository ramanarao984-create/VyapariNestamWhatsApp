package database

import (
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
	goose.SetBaseFS(migrationFS)
	if err := goose.Up(sqlDB, "migrations"); err != nil {
		return fmt.Errorf("apply versioned migrations: %w", err)
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
