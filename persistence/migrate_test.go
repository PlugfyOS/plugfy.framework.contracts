package persistence

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
)

// fakeSQLDB is a stdlib-only stand-in for an SQLDB engine: it records the
// statements ApplyMigrations issues so the runner's dialect selection, ordering,
// transaction framing and version recording can be asserted without a real driver
// (framework.contracts is a stdlib-only leaf — the concrete engines live in the
// provider repo's adapters). It can be told to fail a specific statement to exercise
// the rollback path.
type fakeSQLDB struct {
	dialect Dialect
	failOn  string // substring; a matching ExecContext returns an error
}

type fakeTx struct {
	db         *fakeSQLDB
	execed     []string
	committed  bool
	rolledback bool
}

func (f *fakeSQLDB) Dialect() Dialect                                  { return f.dialect }
func (f *fakeSQLDB) WithTenant(ctx context.Context, _ string) context.Context { return ctx }
func (f *fakeSQLDB) ExecContext(context.Context, string, ...any) (Result, error) {
	return nil, errors.New("not used")
}
func (f *fakeSQLDB) QueryContext(context.Context, string, ...any) (Rows, error) {
	return nil, errors.New("not used")
}
func (f *fakeSQLDB) QueryRowContext(context.Context, string, ...any) Row { return nil }
func (f *fakeSQLDB) BeginTx(context.Context, *sql.TxOptions) (Tx, error) {
	return &fakeTx{db: f}, nil
}

func (t *fakeTx) ExecContext(_ context.Context, query string, _ ...any) (Result, error) {
	t.execed = append(t.execed, query)
	if t.db.failOn != "" && strings.Contains(query, t.db.failOn) {
		return nil, errors.New("boom")
	}
	return nil, nil
}
func (t *fakeTx) QueryContext(context.Context, string, ...any) (Rows, error) {
	return nil, errors.New("not used")
}
func (t *fakeTx) QueryRowContext(context.Context, string, ...any) Row { return nil }
func (t *fakeTx) Commit() error                                       { t.committed = true; return nil }
func (t *fakeTx) Rollback() error                                     { t.rolledback = true; return nil }

var migrationSet = MigrationSet{
	Postgres: []string{"CREATE TABLE pg_only (id text) PARTITION BY RANGE (id)"},
	SQLite:   []string{"CREATE TABLE sqlite_only (id text)"},
}

func TestApplyMigrations_NilDB(t *testing.T) {
	if err := ApplyMigrations(context.Background(), nil, 1, MigrationSet{}); err == nil {
		t.Fatal("nil SQLDB must error")
	}
}

func TestApplyMigrations_SelectsDialect(t *testing.T) {
	cases := []struct {
		dialect Dialect
		want    string // the dialect-specific DDL that must have been issued
		notWant string
	}{
		{DialectPostgres, "pg_only", "sqlite_only"},
		{DialectSQLite, "sqlite_only", "pg_only"},
	}
	for _, c := range cases {
		db := &fakeSQLDB{dialect: c.dialect}
		// Wrap BeginTx so we can inspect the tx afterwards.
		tx := &fakeTx{db: db}
		fdb := &beginInterceptor{fakeSQLDB: db, tx: tx}
		if err := ApplyMigrations(context.Background(), fdb, 7, migrationSet); err != nil {
			t.Fatalf("%s: ApplyMigrations: %v", c.dialect, err)
		}
		joined := strings.Join(tx.execed, " | ")
		if !strings.Contains(joined, c.want) {
			t.Fatalf("%s: expected %q in issued DDL, got %q", c.dialect, c.want, joined)
		}
		if strings.Contains(joined, c.notWant) {
			t.Fatalf("%s: must NOT issue the other dialect's DDL %q: %q", c.dialect, c.notWant, joined)
		}
		if !strings.Contains(joined, "schema_migrations") {
			t.Fatalf("%s: must create schema_migrations: %q", c.dialect, joined)
		}
		if !strings.Contains(joined, "INSERT INTO schema_migrations") {
			t.Fatalf("%s: must record the version: %q", c.dialect, joined)
		}
		if !tx.committed || tx.rolledback {
			t.Fatalf("%s: expected commit (committed=%v rolledback=%v)", c.dialect, tx.committed, tx.rolledback)
		}
	}
}

func TestApplyMigrations_RollsBackOnError(t *testing.T) {
	db := &fakeSQLDB{dialect: DialectSQLite, failOn: "sqlite_only"}
	tx := &fakeTx{db: db}
	fdb := &beginInterceptor{fakeSQLDB: db, tx: tx}
	err := ApplyMigrations(context.Background(), fdb, 1, migrationSet)
	if err == nil {
		t.Fatal("a failing statement must surface an error")
	}
	if !strings.Contains(err.Error(), "sqlite") {
		t.Fatalf("error should name the dialect: %v", err)
	}
	if tx.committed || !tx.rolledback {
		t.Fatalf("failed migration must roll back, not commit (committed=%v rolledback=%v)", tx.committed, tx.rolledback)
	}
}

// beginInterceptor lets a test inspect the single tx ApplyMigrations opens.
type beginInterceptor struct {
	*fakeSQLDB
	tx *fakeTx
}

func (b *beginInterceptor) BeginTx(context.Context, *sql.TxOptions) (Tx, error) { return b.tx, nil }
