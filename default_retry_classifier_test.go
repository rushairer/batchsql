package batchsql_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rushairer/batchsql"
)

// 通过公开的重试路径间接覆盖分类逻辑：构造一个 Processor，前两次返回可重试错误，第三次成功
type retryingProcessor struct {
	attempt int
}

func (p *retryingProcessor) GenerateOperations(ctx context.Context, schema *batchsql.Schema, data []map[string]any) (batchsql.Operations, error) {
	return batchsql.Operations{}, nil
}

func (p *retryingProcessor) ExecuteOperations(ctx context.Context, ops batchsql.Operations) error {
	p.attempt++
	switch p.attempt {
	case 1:
		return errors.New("timeout: i/o timeout") // 应被判为可重试
	case 2:
		return errors.New("deadlock detected") // 应被判为可重试
	default:
		return nil
	}
}

func TestDefaultRetryClassifier_ThroughExecutor(t *testing.T) {
	exec := batchsql.NewThrottledBatchExecutor(&retryingProcessor{})
	// 配一个较小的 backoff 与最大尝试次数，确保覆盖重试路径
	exec.WithRetryConfig(batchsql.RetryConfig{
		Enabled:     true,
		MaxAttempts: 3,
		BackoffBase: 1 * time.Millisecond,
		MaxBackoff:  2 * time.Millisecond,
	})

	ctx := context.Background()
	schema := batchsql.NewSchema("users", batchsql.ConflictIgnore, "id")
	if err := exec.ExecuteBatch(ctx, schema, []map[string]any{{"id": 1}}); err != nil {
		t.Fatalf("expected success after retries, got: %v", err)
	}
}
