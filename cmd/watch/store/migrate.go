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
//
// Some migrations (see 0002_*) rebuild a table other tables reference via
// foreign key — SQLite refuses to DROP such a table while foreign key
// enforcement is on, and PRAGMA foreign_keys is a no-op once a transaction
// is already open, which golang-migrate starts automatically per migration
// file. So enforcement is turned off here, before golang-migrate opens any
// transaction, for the whole migration run, then verified and turned back
// on afterward. This requires db to be limited to a single connection (see
// store.Open's SetMaxOpenConns(1)) — otherwise there's no guarantee the
// connection this toggles is the same one golang-migrate's transaction
// later runs on.
func migrateFromSource(db *sql.DB, sourceDriver source.Driver) error {
	if _, err := db.Exec(`PRAGMA foreign_keys = OFF`); err != nil {
		return fmt.Errorf("disable foreign keys for migration: %w", err)
	}

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

	if err := checkForeignKeys(db); err != nil {
		return err
	}
	if _, err := db.Exec(`PRAGMA foreign_keys = ON`); err != nil {
		return fmt.Errorf("re-enable foreign keys after migration: %w", err)
	}
	return nil
}

// checkForeignKeys fails loudly if any migration left a dangling foreign
// key reference, rather than silently re-enabling enforcement over a
// database that's already inconsistent.
func checkForeignKeys(db *sql.DB) error {
	rows, err := db.Query(`PRAGMA foreign_key_check`)
	if err != nil {
		return fmt.Errorf("check foreign keys after migration: %w", err)
	}
	defer rows.Close()

	if rows.Next() {
		return fmt.Errorf("migration left dangling foreign key references")
	}
	return rows.Err()
}
