package store

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Migrate applies every pending migration embedded in this package to db.
// Applying it twice is a safe no-op: migrate.ErrNoChange is swallowed, not
// treated as a failure. A migration that fails partway is left by
// golang-migrate's sqlite3 driver marked "dirty" rather than silently
// applied — Migrate reports that as an error rather than pretending the
// schema is usable.
func Migrate(db *sql.DB) error {
	sourceDriver, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("load embedded migrations: %w", err)
	}
	return migrateFromSource(db, sourceDriver)
}

// migrateFromSource is Migrate's implementation, parameterized on the
// migration source so tests can exercise failure paths (e.g. a deliberately
// broken migration) without touching this package's real, embedded
// migrations.
func migrateFromSource(db *sql.DB, sourceDriver source.Driver) error {
	dbDriver, err := sqlite3.WithInstance(db, &sqlite3.Config{})
	if err != nil {
		return fmt.Errorf("init migration driver: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", sourceDriver, "sqlite3", dbDriver)
	if err != nil {
		return fmt.Errorf("init migrator: %w", err)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		if version, dirty, verErr := m.Version(); verErr == nil && dirty {
			return fmt.Errorf("migration to version %d failed and left the database dirty: %w", version, err)
		}
		return fmt.Errorf("run migrations: %w", err)
	}
	return nil
}
