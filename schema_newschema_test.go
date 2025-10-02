package batchsql_test

import (
	"testing"

	"github.com/rushairer/batchsql"
)

func TestNewSchema_Basic(t *testing.T) {
	s := batchsql.NewSchema("users", batchsql.ConflictIgnore, "id", "name", "email")
	if s.Name != "users" {
		t.Fatalf("schema name expected users, got %s", s.Name)
	}
	if len(s.Columns) != 3 {
		t.Fatalf("expected 3 columns, got %d", len(s.Columns))
	}
	if s.Columns[0] != "id" || s.Columns[1] != "name" || s.Columns[2] != "email" {
		t.Fatalf("columns order unexpected: %#v", s.Columns)
	}
}
