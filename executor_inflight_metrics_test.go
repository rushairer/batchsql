package batchsql_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/rushairer/batchsql"
)

type inflightMetrics struct {
	mu       sync.Mutex
	inflight int
	max      int
}

func (m *inflightMetrics) ObserveEnqueueLatency(d time.Duration) {}
func (m *inflightMetrics) ObserveBatchAssemble(d time.Duration)  {}
func (m *inflightMetrics) ObserveExecuteDuration(table string, n int, d time.Duration, status string) {
}
func (m *inflightMetrics) ObserveBatchSize(n int) {}
func (m *inflightMetrics) SetConcurrency(n int)   {}
func (m *inflightMetrics) SetQueueLength(n int)   {}
func (m *inflightMetrics) IncInflight() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.inflight++
	if m.inflight > m.max {
		m.max = m.inflight
	}
}

func (m *inflightMetrics) DecInflight() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.inflight--
}
func (m *inflightMetrics) IncError(table, kind string) {}

type okProcessor struct{}

func (okProcessor) GenerateOperations(ctx context.Context, schema *batchsql.Schema, data []map[string]any) (batchsql.Operations, error) {
	return batchsql.Operations{}, nil
}

func (okProcessor) ExecuteOperations(ctx context.Context, ops batchsql.Operations) error {
	return nil
}

func TestExecutor_IncDecInflight_Matches(t *testing.T) {
	// Processor that always succeeds
	exec := batchsql.NewThrottledBatchExecutor(okProcessor{})
	m := &inflightMetrics{}
	exec.WithMetricsReporter(m)

	ctx := context.Background()
	schema := batchsql.NewSchema("users", batchsql.ConflictIgnore, "id")
	var wg sync.WaitGroup
	N := 20
	wg.Add(N)
	for i := 0; i < N; i++ {
		go func(n int) {
			defer wg.Done()
			_ = exec.ExecuteBatch(ctx, schema, []map[string]any{{"id": n}})
		}(i)
	}
	wg.Wait()

	if m.inflight != 0 {
		t.Fatalf("inflight should return to 0, got %d", m.inflight)
	}
	if m.max <= 0 {
		t.Fatalf("expected max inflight > 0")
	}
}
