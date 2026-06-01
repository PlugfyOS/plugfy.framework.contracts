package persistence

import (
	"database/sql"
	"errors"
	"testing"
)

func TestErrNoRowsIsSQLErrNoRows(t *testing.T) {
	if !errors.Is(ErrNoRows, sql.ErrNoRows) {
		t.Fatal("ErrNoRows must alias sql.ErrNoRows")
	}
}

func TestRebindPostgresUnchanged(t *testing.T) {
	q := "SELECT * FROM t WHERE a = $1 AND b = $2"
	if got := Rebind(DialectPostgres, q); got != q {
		t.Errorf("Rebind(postgres) = %q, want unchanged %q", got, q)
	}
}

func TestRebindSQLite(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{
			"SELECT * FROM t WHERE a = $1 AND b = $2",
			"SELECT * FROM t WHERE a = ? AND b = ?",
		},
		{
			"INSERT INTO t (a,b,c) VALUES ($1,$2,$3)",
			"INSERT INTO t (a,b,c) VALUES (?,?,?)",
		},
		{
			// Two-digit index must consume both digits.
			"x = $10 OR y = $1",
			"x = ? OR y = ?",
		},
		{
			// Doubled $$ collapses to a literal $, and a lone $ stays.
			"price = $$5 AND note = '$'",
			"price = $5 AND note = '$'",
		},
		{
			// No placeholders -> unchanged.
			"SELECT 1",
			"SELECT 1",
		},
	}
	for _, c := range cases {
		if got := Rebind(DialectSQLite, c.in); got != c.want {
			t.Errorf("Rebind(sqlite, %q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestNowFragment(t *testing.T) {
	if got := Now(DialectPostgres); got != "now()" {
		t.Errorf("Now(postgres) = %q", got)
	}
	if got := Now(DialectSQLite); got != "CURRENT_TIMESTAMP" {
		t.Errorf("Now(sqlite) = %q", got)
	}
}

func TestJSONExtractFragment(t *testing.T) {
	if got := JSONExtract(DialectPostgres, "doc", "name"); got != "doc ->> 'name'" {
		t.Errorf("JSONExtract(postgres) = %q", got)
	}
	if got := JSONExtract(DialectSQLite, "doc", "name"); got != "json_extract(doc, '$.name')" {
		t.Errorf("JSONExtract(sqlite) = %q", got)
	}
}

func TestUpsertFragment(t *testing.T) {
	got := Upsert(DialectPostgres, []string{"namespace", "key"}, []string{"value = EXCLUDED.value", "version = t.version + 1"})
	want := "ON CONFLICT (namespace, key) DO UPDATE SET value = EXCLUDED.value, version = t.version + 1"
	if got != want {
		t.Errorf("Upsert = %q, want %q", got, want)
	}
	// Both dialects produce the same clause.
	if Upsert(DialectSQLite, []string{"k"}, []string{"v = EXCLUDED.v"}) != Upsert(DialectPostgres, []string{"k"}, []string{"v = EXCLUDED.v"}) {
		t.Error("Upsert must be dialect-stable")
	}
}

func TestDialectTokens(t *testing.T) {
	if string(DialectPostgres) != "postgres" {
		t.Errorf("DialectPostgres = %q", DialectPostgres)
	}
	if string(DialectSQLite) != "sqlite" {
		t.Errorf("DialectSQLite = %q", DialectSQLite)
	}
}
