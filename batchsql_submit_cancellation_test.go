package batchsql_test

import (
	"context"
	"testing"
	"time"

	"github.com/rushairer/batchsql"
)

func TestBatchSQL_Submit_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	cfg := batchsql.PipelineConfig{
		BufferSize:    16,
		FlushSize:     8,
		FlushInterval: 50 * time.Millisecond,
	}
	b, _ := batchsql.NewBatchSQLWithMock(ctx, cfg)

	schema := batchsql.NewSchema("users", batchsql.ConflictIgnore, "id", "name")
	req := batchsql.NewRequest(schema).SetInt64("id", 1).SetString("name", "a")

	if err := b.Submit(ctx, req); err == nil {
		t.Fatalf("expected submit to fail when pipeline context already cancelled")
	}
}

func TestBatchSQL_Submit_ImmediateCtxErr(t *testing.T) {
	ctx := context.Background()
	cfg := batchsql.PipelineConfig{
		BufferSize:    16,
		FlushSize:     8,
		FlushInterval: 50 * time.Millisecond,
	}
	b, _ := batchsql.NewBatchSQLWithMock(ctx, cfg)

	schema := batchsql.NewSchema("users", batchsql.ConflictIgnore, "id")
	req := batchsql.NewRequest(schema).SetInt64("id", 1)

	// pass a cancelled ctx to Submit
	reqCtx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := b.Submit(reqCtx, req); err == nil {
		t.Fatalf("expected submit to fail with cancelled ctx")
	}
}
