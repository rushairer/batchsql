package batchsql_test

import (
	"context"
	"strings"
	"testing"

	"github.com/rushairer/batchsql"
)

func TestMockDriver_GenerateInsertSQL_Variants(t *testing.T) {
	schema := batchsql.NewSchema("users", batchsql.ConflictStrategy(0), "id", "name")
	data := []map[string]any{
		{"id": 1, "name": "a"},
		{"id": 2, "name": "b"},
	}

	tests := []struct {
		name   string
		driver *batchsql.MockDriver
		conf   batchsql.ConflictStrategy
		wantIn string
	}{
		{"mysql_base", batchsql.NewMockDriver("mysql"), batchsql.ConflictStrategy(255), "INSERT INTO users (id, name) VALUES"},
		{"mysql_ignore", batchsql.NewMockDriver("mysql"), batchsql.ConflictIgnore, "INSERT IGNORE INTO users"},
		{"mysql_replace", batchsql.NewMockDriver("mysql"), batchsql.ConflictReplace, "REPLACE INTO users"},
		{"mysql_update", batchsql.NewMockDriver("mysql"), batchsql.ConflictUpdate, "ON DUPLICATE KEY UPDATE"},
		{"pg_none", batchsql.NewMockDriver("postgresql"), batchsql.ConflictStrategy(0), "INSERT INTO users (id, name) VALUES"},
		{"pg_ignore", batchsql.NewMockDriver("postgresql"), batchsql.ConflictIgnore, "ON CONFLICT DO NOTHING"},
		{"pg_update", batchsql.NewMockDriver("postgresql"), batchsql.ConflictUpdate, "ON CONFLICT (id) DO UPDATE SET"},
		{"sqlite_base", batchsql.NewMockDriver("sqlite"), batchsql.ConflictStrategy(255), "INSERT INTO users (id, name) VALUES"},
		{"sqlite_ignore", batchsql.NewMockDriver("sqlite"), batchsql.ConflictIgnore, "INSERT OR IGNORE INTO users"},
		{"sqlite_replace", batchsql.NewMockDriver("sqlite"), batchsql.ConflictReplace, "INSERT OR REPLACE INTO users"},
		{"sqlite_update", batchsql.NewMockDriver("sqlite"), batchsql.ConflictUpdate, "ON CONFLICT DO UPDATE SET"},
		{"default_none", batchsql.NewMockDriver("unknown"), batchsql.ConflictStrategy(0), "INSERT INTO users (id, name) VALUES"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := batchsql.NewSchema("users", tt.conf, "id", "name")
			sql, args, err := tt.driver.GenerateInsertSQL(context.Background(), s, data)
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if !stringsContains(sql, tt.wantIn) {
				t.Fatalf("sql %q does not contain %q", sql, tt.wantIn)
			}
			if len(args) != 4 {
				t.Fatalf("args len = %d, want 4", len(args))
			}
		})
	}

	t.Run("empty_data", func(t *testing.T) {
		sql, args, err := batchsql.NewMockDriver("mysql").GenerateInsertSQL(context.Background(), schema, nil)
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if sql != "" || args != nil {
			t.Fatalf("expect empty sql and nil args, got %q %#v", sql, args)
		}
	})

	t.Run("ctx_cancel", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, _, err := batchsql.NewMockDriver("mysql").GenerateInsertSQL(ctx, schema, data)
		if err == nil {
			t.Fatalf("expected context error")
		}
		// reflect to ensure it's a context error (not strict type)
		if !isContextError(err) {
			t.Fatalf("expected context-related error, got %v", err)
		}
	})
}

func stringsContains(s, sub string) bool { return strings.Contains(s, sub) }
func isContextError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "context canceled") || strings.Contains(msg, "context deadline")
}
