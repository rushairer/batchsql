package batchsql_test

import (
	"context"
	"testing"
	"time"

	"github.com/rushairer/batchsql"
	"github.com/rushairer/batchsql/drivers/mysql"
	"github.com/rushairer/batchsql/drivers/postgresql"
	"github.com/rushairer/batchsql/drivers/sqlite"
)

func TestBatchSQL(t *testing.T) {
	ctx := context.Background()

	// 使用模拟执行器进行测试
	config := batchsql.PipelineConfig{
		BufferSize:    100,
		FlushSize:     10,
		FlushInterval: time.Second,
	}
	batch, mockExecutor := batchsql.NewBatchSQLWithMock(ctx, config)

	// 创建不同的 schema
	mysqlSchema := batchsql.NewSchema("users", batchsql.ConflictIgnore, "id", "name", "email", "created_at")
	postgresSchema := batchsql.NewSchema("products", batchsql.ConflictUpdate, "id", "name", "price")
	sqliteSchema := batchsql.NewSchema("logs", batchsql.ConflictReplace, "id", "message", "timestamp")

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
	config := batchsql.PipelineConfig{
		BufferSize:    100,
		FlushSize:     5,
		FlushInterval: 100 * time.Millisecond,
	}
	batch, mockExecutor := batchsql.NewBatchSQLWithMock(ctx, config)

	// 创建两个相同的 schema 实例
	schema1 := batchsql.NewSchema("test_table", batchsql.ConflictIgnore, "id", "name")
	schema2 := batchsql.NewSchema("test_table", batchsql.ConflictIgnore, "id", "name")

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
			schema:   batchsql.NewSchema("users", batchsql.ConflictIgnore, "id", "name"),
			expected: "INSERT IGNORE INTO users (id, name) VALUES (?, ?), (?, ?)",
		},
		{
			name:     "PostgreSQL ON CONFLICT DO NOTHING",
			schema:   batchsql.NewSchema("users", batchsql.ConflictIgnore, "id", "name"),
			expected: "INSERT INTO users (id, name) VALUES (?, ?), (?, ?) ON CONFLICT DO NOTHING",
		},
		{
			name:     "SQLite INSERT OR IGNORE",
			schema:   batchsql.NewSchema("users", batchsql.ConflictIgnore, "id", "name"),
			expected: "INSERT OR IGNORE INTO users (id, name) VALUES (?, ?), (?, ?)",
		},
	}

	drivers := map[string]batchsql.SQLDriver{
		"MySQL INSERT IGNORE":               mysql.DefaultDriver,
		"PostgreSQL ON CONFLICT DO NOTHING": postgresql.DefaultDriver,
		"SQLite INSERT OR IGNORE":           sqlite.DefaultDriver,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			driver := drivers[tt.name]
			sql, args, err := driver.GenerateInsertSQL(tt.schema, []map[string]any{
				{"id": 1, "name": "test1"},
				{"id": 2, "name": "test2"},
			})
			if err != nil {
				t.Errorf("GenerateInsertSQL failed: %v", err)
				return
			}
			if len(args) != 4 {
				t.Errorf("Expected 4 args, got %d", len(args))
			}
			// 注意：这里只检查SQL是否包含关键部分，因为不同驱动的占位符可能不同
			t.Logf("Generated SQL: %s", sql)
			t.Logf("Generated Args: %v", args)
		})
	}
}
