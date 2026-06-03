package persistence

import (
	"context"
	"fmt"
)

// MigrationSet is a per-dialect set of forward-only, idempotent DDL statements
// (EDB F0/#67 D0.4). A unit authors its schema twice — Postgres and SQLite —
// because the DDL diverges (TIMESTAMPTZ->TEXT/INTEGER, JSONB->TEXT, no
// PARTITION/RLS on SQLite) while its DML stays a single $N-placeholder set the
// [SQLDB] seam rebinds. [ApplyMigrations] picks the set for the live [Dialect].
//
// It lives here, in the dialect-neutral persistence seam alongside [Rebind] /
// [Now] / [Upsert], so a data-plane store can author + apply its schema using only
// the contracts package — it never has to import a concrete database driver to run
// migrations. The driver provides the engine ([SQLDB] implementations); this
// package provides the dialect-neutral surface programmed against it.
type MigrationSet struct {
	Postgres []string
	SQLite   []string
}

// ApplyMigrations runs the dialect-appropriate DDL of set against db inside a
// single transaction, then records the applied version in a schema_migrations
// table both engines understand. It is idempotent: every statement is expected
// to be CREATE … IF NOT EXISTS / INSERT … ON CONFLICT, so a re-run on boot is a
// no-op. version is the migration revision recorded (1-based).
//
// The whole set is applied transactionally so a partial schema never lands; on
// any statement error the transaction rolls back and the error is returned with
// the offending dialect for diagnosis.
func ApplyMigrations(ctx context.Context, db SQLDB, version int, set MigrationSet) error {
	if db == nil {
		return fmt.Errorf("migrate: nil SQLDB")
	}
	dialect := db.Dialect()
	stmts := set.Postgres
	if dialect == DialectSQLite {
		stmts = set.SQLite
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("migrate (%s): begin: %w", dialect, err)
	}
	rollback := func(cause error) error {
		_ = tx.Rollback()
		return cause
	}

	// schema_migrations is plain portable DDL both engines accept; it records the
	// applied revision so a future migration runner can branch on it.
	if _, err := tx.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY,
		applied_at TEXT NOT NULL DEFAULT (CURRENT_TIMESTAMP)
	)`); err != nil {
		return rollback(fmt.Errorf("migrate (%s): schema_migrations: %w", dialect, err))
	}

	for i, q := range stmts {
		if _, err := tx.ExecContext(ctx, q); err != nil {
			return rollback(fmt.Errorf("migrate (%s): statement %d: %w", dialect, i+1, err))
		}
	}

	// Record the revision (idempotent). $1 rebinds to "?" on SQLite, passes through
	// on Postgres — the single-DML rule the seam guarantees.
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO schema_migrations (version) VALUES ($1) ON CONFLICT (version) DO NOTHING`,
		version); err != nil {
		return rollback(fmt.Errorf("migrate (%s): record version: %w", dialect, err))
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("migrate (%s): commit: %w", dialect, err)
	}
	return nil
}
