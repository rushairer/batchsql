package batchsql_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rushairer/batchsql"
)

// 总是返回可重试错误的 Processor
type alwaysRetryProcessor struct{}

func (alwaysRetryProcessor) GenerateOperations(ctx context.Context, schema *batchsql.Schema, data []map[string]any) (batchsql.Operations, error) {
	return batchsql.Operations{}, nil
}

func (alwaysRetryProcessor) ExecuteOperations(ctx context.Context, ops batchsql.Operations) error {
	return errors.New("timeout: temporary network failure")
}

// 计数 final 错误的假 Reporter（作用于 Executor）
type retryMetrics struct {
	retry    int32
	final    int32
	inflight int32
}

func (m *retryMetrics) ObserveEnqueueLatency(d time.Duration)                                      {}
func (m *retryMetrics) ObserveBatchAssemble(d time.Duration)                                       {}
func (m *retryMetrics) ObserveExecuteDuration(table string, n int, d time.Duration, status string) {}
func (m *retryMetrics) ObserveBatchSize(n int)                                                     {}
func (m *retryMetrics) SetConcurrency(n int)                                                       {}
func (m *retryMetrics) SetQueueLength(n int)                                                       {}
func (m *retryMetrics) IncInflight()                                                               { atomic.AddInt32(&m.inflight, 1) }
func (m *retryMetrics) DecInflight()                                                               { atomic.AddInt32(&m.inflight, -1) }
func (m *retryMetrics) IncError(table, kind string) {
	if len(kind) >= 6 && kind[:6] == "retry:" {
		atomic.AddInt32(&m.retry, 1)
	} else if len(kind) >= 6 && kind[:6] == "final:" {
		atomic.AddInt32(&m.final, 1)
	}
}

func TestThrottledExecutor_Retry_FinalFailure(t *testing.T) {
	exec := batchsql.NewThrottledBatchExecutor(alwaysRetryProcessor{})
	exec.WithRetryConfig(batchsql.RetryConfig{
		Enabled:     true,
		MaxAttempts: 2, // 1次初始 + 1次重试，仍失败
		BackoffBase: 1 * time.Millisecond,
		MaxBackoff:  2 * time.Millisecond,
	})
	m := &retryMetrics{}
	exec.WithMetricsReporter(m)

	ctx := context.Background()
	schema := batchsql.NewSchema("users", batchsql.ConflictIgnore, "id")
	err := exec.ExecuteBatch(ctx, schema, []map[string]any{{"id": 1}})
	if err == nil {
		t.Fatalf("expected final failure, got nil")
	}
	// 至少出现一次 retry、一次 final 记录
	if atomic.LoadInt32(&m.retry) == 0 {
		t.Fatalf("expected at least one retry error recorded")
	}
	if atomic.LoadInt32(&m.final) == 0 {
		t.Fatalf("expected at least one final error recorded")
	}
	// in-flight 必须归零
	if atomic.LoadInt32(&m.inflight) != 0 {
		t.Fatalf("inflight should be 0 at end, got %d", m.inflight)
	}
}
