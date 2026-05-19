package store

import (
	"context"
	"database/sql"
	"embed"
	"fmt"

	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func Open(ctx context.Context, dialect, uri string) (*sql.DB, error) {
	driver := dialect
	if dialect == "sqlite3" {
		driver = "sqlite3"
	}
	db, err := sql.Open(driver, uri)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}
	return db, nil
}

func Migrate(db *sql.DB, dialect string) error {
	gooseDialect := dialect
	if dialect == "sqlite3" {
		gooseDialect = "sqlite3"
	}
	if err := goose.SetDialect(gooseDialect); err != nil {
		return err
	}
	goose.SetBaseFS(migrationsFS)
	return goose.Up(db, "migrations")
}
