package database

import (
	"database/sql"
	"embed"
	"fmt"
	"log/slog"
	"time"

	migrate "github.com/rubenv/sql-migrate"
	"github.com/uptrace/bun/driver/pgdriver"
)

const dbName = "app"

//go:embed migrations/*.sql
var migrationsFS embed.FS

func NewPostgres(url, host string) (*sql.DB, error) {
	if url == "" {
		url = fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable", dbName, dbName, host, dbName)
	}
	slog.Info("database connection string", "url", url)

	db := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(url)))
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := runMigrations(db); err != nil {
		return nil, fmt.Errorf("running migrations: %v", err)
	}

	return db, nil
}

func runMigrations(db *sql.DB) error {
	source := &migrate.EmbedFileSystemMigrationSource{
		FileSystem: migrationsFS,
		Root:       "migrations",
	}
	if _, err := migrate.Exec(db, "postgres", source, migrate.Up); err != nil {
		return err
	}
	return nil
}
