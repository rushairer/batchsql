package batchsql_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rushairer/batchsql"
)

// failOnceProcessor 第一次失败，其余成功
type failOnceProcessor struct {
	done int32
}

func (p *failOnceProcessor) GenerateOperations(ctx context.Context, schema *batchsql.Schema, data []map[string]any) (batchsql.Operations, error) {
	return batchsql.Operations{}, nil
}

func (p *failOnceProcessor) ExecuteOperations(ctx context.Context, ops batchsql.Operations) error {
	if atomic.CompareAndSwapInt32(&p.done, 0, 1) {
		return errors.New("temporary failure")
	}
	return nil
}

// 轻量的 Metrics 计数器
type execMetrics struct {
	observeCalls int32
	setConcCalls int32
	lastStatus   atomic.Value // string
}

func (m *execMetrics) ObserveEnqueueLatency(d time.Duration) {}
func (m *execMetrics) ObserveBatchAssemble(d time.Duration)  {}
func (m *execMetrics) ObserveExecuteDuration(table string, n int, d time.Duration, status string) {
	atomic.AddInt32(&m.observeCalls, 1)
	m.lastStatus.Store(status)
}
func (m *execMetrics) ObserveBatchSize(n int)      {}
func (m *execMetrics) SetConcurrency(n int)        { atomic.AddInt32(&m.setConcCalls, 1) }
func (m *execMetrics) SetQueueLength(n int)        {}
func (m *execMetrics) IncInflight()                {}
func (m *execMetrics) DecInflight()                {}
func (m *execMetrics) IncError(table, kind string) {}

func TestExecutor_MetricsCallbacks_SuccessAndFailAndSetConcurrency(t *testing.T) {
	ctx := context.Background()
	schema := batchsql.NewSchema("users", batchsql.ConflictIgnore, "id")

	// 1) 成功路径覆盖 ObserveExecuteDuration
	exec1 := batchsql.NewThrottledBatchExecutor(okProcessor{})
	m1 := &execMetrics{}
	exec1.WithMetricsReporter(m1)
	exec1.WithConcurrencyLimit(4) // 触发 SetConcurrency
	if err := exec1.ExecuteBatch(ctx, schema, []map[string]any{{"id": 1}}); err != nil {
		t.Fatalf("exec1 success expected, got err: %v", err)
	}
	if atomic.LoadInt32(&m1.observeCalls) == 0 {
		t.Fatalf("expected ObserveExecuteDuration called for success")
	}
	if atomic.LoadInt32(&m1.setConcCalls) == 0 {
		t.Fatalf("expected SetConcurrency called at least once")
	}

	// 2) 失败路径覆盖 ObserveExecuteDuration（status=fail）
	exec2 := batchsql.NewThrottledBatchExecutor(&failOnceProcessor{})
	m2 := &execMetrics{}
	exec2.WithMetricsReporter(m2)
	// 不开启重试，确保直接失败一次
	err := exec2.ExecuteBatch(ctx, schema, []map[string]any{{"id": 2}})
	if err == nil {
		t.Fatalf("exec2 expected failure, got nil")
	}
	if atomic.LoadInt32(&m2.observeCalls) == 0 {
		t.Fatalf("expected ObserveExecuteDuration called for failure")
	}
}
