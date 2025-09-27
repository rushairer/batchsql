package batchsql_test

import (
	"context"
	"testing"
	"time"

	"github.com/rushairer/batchsql"
)

func TestBatchSQL(t *testing.T) {
	ctx := context.Background()

	// 使用模拟执行器进行测试
	batch, mockExecutor := batchsql.NewBatchSQLWithMock(ctx, 100, 10, time.Second)
	defer batch.Close()

	// 创建不同的 schema
	mysqlSchema := batchsql.NewSchema("users", batchsql.ConflictIgnore, batchsql.MySQL, "id", "name", "email", "created_at")
	postgresSchema := batchsql.NewSchema("products", batchsql.ConflictUpdate, batchsql.PostgreSQL, "id", "name", "price")
	sqliteSchema := batchsql.NewSchema("logs", batchsql.ConflictReplace, batchsql.SQLite, "id", "message", "timestamp")

	// 提交不同类型的请求
	for i := 0; i < 50; i++ {
		// MySQL 用户数据
		userRequest := batchsql.NewRequest(mysqlSchema).
			SetInt64("id", int64(i)).
			SetString("name", "User"+string(rune(i))).
			SetString("email", "user"+string(rune(i))+"@example.com").
			SetTime("created_at", time.Now())

		if err := batch.Submit(ctx, userRequest); err != nil {
			t.Errorf("submit user request failed: %v", err)
		}

		// PostgreSQL 产品数据
		if i%2 == 0 {
			productRequest := batchsql.NewRequest(postgresSchema).
				SetInt64("id", int64(i/2)).
				SetString("name", "Product"+string(rune(i/2))).
				SetFloat64("price", float64(i)*10.5)

			if err := batch.Submit(ctx, productRequest); err != nil {
				t.Errorf("submit product request failed: %v", err)
			}
		}

		// SQLite 日志数据
		if i%3 == 0 {
			logRequest := batchsql.NewRequest(sqliteSchema).
				SetInt64("id", int64(i/3)).
				SetString("message", "Log message "+string(rune(i))).
				SetTime("timestamp", time.Now())

			if err := batch.Submit(ctx, logRequest); err != nil {
				t.Errorf("submit log request failed: %v", err)
			}
		}
	}

	// 等待批量处理完成
	time.Sleep(2 * time.Second)

	// 验证执行结果
	if len(mockExecutor.ExecutedBatches) == 0 {
		t.Error("No batches were executed")
	}

	t.Logf("Total executed batches: %d", len(mockExecutor.ExecutedBatches))
	for i, batch := range mockExecutor.ExecutedBatches {
		t.Logf("Batch %d: %d requests", i, len(batch))
	}
}

func TestSchemaGrouping(t *testing.T) {
	ctx := context.Background()
	batch, mockExecutor := batchsql.NewBatchSQLWithMock(ctx, 100, 5, 100*time.Millisecond)
	defer batch.Close()

	// 创建两个相同的 schema 实例
	schema1 := batchsql.NewSchema("test_table", batchsql.ConflictIgnore, batchsql.MySQL, "id", "name")
	schema2 := batchsql.NewSchema("test_table", batchsql.ConflictIgnore, batchsql.MySQL, "id", "name")

	// 提交使用不同 schema 实例的请求
	for i := 0; i < 3; i++ {
		req1 := batchsql.NewRequest(schema1).SetInt64("id", int64(i)).SetString("name", "name1_"+string(rune(i)))
		req2 := batchsql.NewRequest(schema2).SetInt64("id", int64(i+10)).SetString("name", "name2_"+string(rune(i)))

		if err := batch.Submit(ctx, req1); err != nil {
			t.Errorf("submit req1 failed: %v", err)
		}
		if err := batch.Submit(ctx, req2); err != nil {
			t.Errorf("submit req2 failed: %v", err)
		}
	}

	// 等待处理完成
	time.Sleep(300 * time.Millisecond)

	// 验证是否按 schema 指针正确分组
	if len(mockExecutor.ExecutedBatches) == 0 {
		t.Error("No batches were executed")
	}

	t.Logf("Schema grouping test - executed batches: %d", len(mockExecutor.ExecutedBatches))
}

func TestSQLGeneration(t *testing.T) {
	tests := []struct {
		name     string
		schema   *batchsql.Schema
		expected string
	}{
		{
			name:     "MySQL INSERT IGNORE",
			schema:   batchsql.NewSchema("users", batchsql.ConflictIgnore, batchsql.MySQL, "id", "name"),
			expected: "INSERT IGNORE INTO users (id, name) VALUES (?, ?), (?, ?)",
		},
		{
			name:     "PostgreSQL ON CONFLICT DO NOTHING",
			schema:   batchsql.NewSchema("users", batchsql.ConflictIgnore, batchsql.PostgreSQL, "id", "name"),
			expected: "INSERT INTO users (id, name) VALUES (?, ?), (?, ?) ON CONFLICT DO NOTHING",
		},
		{
			name:     "SQLite INSERT OR IGNORE",
			schema:   batchsql.NewSchema("users", batchsql.ConflictIgnore, batchsql.SQLite, "id", "name"),
			expected: "INSERT OR IGNORE INTO users (id, name) VALUES (?, ?), (?, ?)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql := tt.schema.GenerateInsertSQL(2)
			if sql != tt.expected {
				t.Errorf("Expected: %s\nGot: %s", tt.expected, sql)
			}
		})
	}
}
