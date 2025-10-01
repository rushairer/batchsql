package batchsql_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/rushairer/batchsql"
)

func TestBoundary_EmptyData(t *testing.T) {
	ctx := context.Background()
	config := batchsql.PipelineConfig{
		BufferSize:    10,
		FlushSize:     5,
		FlushInterval: time.Second,
	}

	batch, _ := batchsql.NewBatchSQLWithMock(ctx, config)

	// 测试空字符串
	schema := batchsql.NewSchema("test_table", batchsql.ConflictIgnore, "name", "value")
	request := batchsql.NewRequest(schema).
		SetString("name", "").
		SetString("value", "")

	err := batch.Submit(ctx, request)
	if err != nil {
		t.Errorf("Should handle empty strings: %v", err)
	}
}

func TestBoundary_NilValues(t *testing.T) {
	ctx := context.Background()
	config := batchsql.PipelineConfig{
		BufferSize:    10,
		FlushSize:     5,
		FlushInterval: time.Second,
	}

	batch, _ := batchsql.NewBatchSQLWithMock(ctx, config)

	// 测试 nil 值
	schema := batchsql.NewSchema("test_table", batchsql.ConflictIgnore, "name", "value")
	request := batchsql.NewRequest(schema).
		SetString("name", "test").
		SetNull("value")

	err := batch.Submit(ctx, request)
	if err != nil {
		t.Errorf("Should handle nil values: %v", err)
	}
}

func TestBoundary_LargeStrings(t *testing.T) {
	ctx := context.Background()
	config := batchsql.PipelineConfig{
		BufferSize:    10,
		FlushSize:     5,
		FlushInterval: time.Second,
	}

	batch, _ := batchsql.NewBatchSQLWithMock(ctx, config)

	// 创建大字符串 (1MB)
	largeString := strings.Repeat("A", 1024*1024)

	schema := batchsql.NewSchema("test_table", batchsql.ConflictIgnore, "id", "large_text")
	request := batchsql.NewRequest(schema).
		SetInt64("id", 1).
		SetString("large_text", largeString)

	err := batch.Submit(ctx, request)
	if err != nil {
		t.Errorf("Should handle large strings: %v", err)
	}
}

func TestBoundary_MaxInt64(t *testing.T) {
	ctx := context.Background()
	config := batchsql.PipelineConfig{
		BufferSize:    10,
		FlushSize:     5,
		FlushInterval: time.Second,
	}

	batch, _ := batchsql.NewBatchSQLWithMock(ctx, config)

	schema := batchsql.NewSchema("test_table", batchsql.ConflictIgnore, "max_val", "min_val")
	request := batchsql.NewRequest(schema).
		SetInt64("max_val", 9223372036854775807). // math.MaxInt64
		SetInt64("min_val", -9223372036854775808) // math.MinInt64

	err := batch.Submit(ctx, request)
	if err != nil {
		t.Errorf("Should handle max/min int64 values: %v", err)
	}
}

func TestBoundary_MaxFloat64(t *testing.T) {
	ctx := context.Background()
	config := batchsql.PipelineConfig{
		BufferSize:    10,
		FlushSize:     5,
		FlushInterval: time.Second,
	}

	batch, _ := batchsql.NewBatchSQLWithMock(ctx, config)

	schema := batchsql.NewSchema("test_table", batchsql.ConflictIgnore, "max_float", "min_float", "zero_float")
	request := batchsql.NewRequest(schema).
		SetFloat64("max_float", 1.7976931348623157e+308).  // math.MaxFloat64
		SetFloat64("min_float", -1.7976931348623157e+308). // -math.MaxFloat64
		SetFloat64("zero_float", 0.0)

	err := batch.Submit(ctx, request)
	if err != nil {
		t.Errorf("Should handle max/min float64 values: %v", err)
	}
}

func TestBoundary_SpecialFloats(t *testing.T) {
	ctx := context.Background()
	config := batchsql.PipelineConfig{
		BufferSize:    10,
		FlushSize:     5,
		FlushInterval: time.Second,
	}

	batch, _ := batchsql.NewBatchSQLWithMock(ctx, config)

	schema := batchsql.NewSchema("test_table", batchsql.ConflictIgnore, "id", "special_float")

	// 测试 NaN - 使用一个变量来避免编译时除零错误
	zero := 0.0
	nanValue := zero / zero
	request1 := batchsql.NewRequest(schema).
		SetInt64("id", 1).
		SetFloat64("special_float", nanValue) // NaN

	err := batch.Submit(ctx, request1)
	if err != nil {
		t.Errorf("Should handle NaN values: %v", err)
	}

	// 测试正无穷
	one := 1.0
	posInf := one / zero
	request2 := batchsql.NewRequest(schema).
		SetInt64("id", 2).
		SetFloat64("special_float", posInf) // +Inf

	err = batch.Submit(ctx, request2)
	if err != nil {
		t.Errorf("Should handle +Inf values: %v", err)
	}

	// 测试负无穷
	negOne := -1.0
	negInf := negOne / zero
	request3 := batchsql.NewRequest(schema).
		SetInt64("id", 3).
		SetFloat64("special_float", negInf) // -Inf

	err = batch.Submit(ctx, request3)
	if err != nil {
		t.Errorf("Should handle -Inf values: %v", err)
	}
}

func TestBoundary_UnicodeStrings(t *testing.T) {
	ctx := context.Background()
	config := batchsql.PipelineConfig{
		BufferSize:    10,
		FlushSize:     5,
		FlushInterval: time.Second,
	}

	batch, _ := batchsql.NewBatchSQLWithMock(ctx, config)

	schema := batchsql.NewSchema("test_table", batchsql.ConflictIgnore, "id", "unicode_text")

	// 测试各种 Unicode 字符
	unicodeStrings := []string{
		"Hello, 世界",      // 中文
		"🚀🎉💻",            // Emoji
		"Ñoël",           // 重音符号
		"Здравствуй мир", // 俄文
		"مرحبا بالعالم",  // 阿拉伯文
		"こんにちは世界",        // 日文
		"🏳️‍🌈🏳️‍⚧️",      // 复合 Emoji
	}

	for i, str := range unicodeStrings {
		request := batchsql.NewRequest(schema).
			SetInt64("id", int64(i)).
			SetString("unicode_text", str)

		err := batch.Submit(ctx, request)
		if err != nil {
			t.Errorf("Should handle Unicode string '%s': %v", str, err)
		}
	}
}

func TestBoundary_SpecialCharacters(t *testing.T) {
	ctx := context.Background()
	config := batchsql.PipelineConfig{
		BufferSize:    10,
		FlushSize:     5,
		FlushInterval: time.Second,
	}

	batch, _ := batchsql.NewBatchSQLWithMock(ctx, config)

	schema := batchsql.NewSchema("test_table", batchsql.ConflictIgnore, "id", "special_text")

	// 测试特殊字符
	specialStrings := []string{
		"'single quotes'",
		"\"double quotes\"",
		"back\\slash",
		"new\nline",
		"tab\ttab",
		"carriage\rreturn",
		"null\x00byte",
		"control\x01\x02\x03chars",
	}

	for i, str := range specialStrings {
		request := batchsql.NewRequest(schema).
			SetInt64("id", int64(i)).
			SetString("special_text", str)

		err := batch.Submit(ctx, request)
		if err != nil {
			t.Errorf("Should handle special string '%s': %v", str, err)
		}
	}
}

func TestBoundary_ZeroTime(t *testing.T) {
	ctx := context.Background()
	config := batchsql.PipelineConfig{
		BufferSize:    10,
		FlushSize:     5,
		FlushInterval: time.Second,
	}

	batch, _ := batchsql.NewBatchSQLWithMock(ctx, config)

	schema := batchsql.NewSchema("test_table", batchsql.ConflictIgnore, "id", "zero_time", "unix_epoch")
	request := batchsql.NewRequest(schema).
		SetInt64("id", 1).
		SetTime("zero_time", time.Time{}).     // 零值时间
		SetTime("unix_epoch", time.Unix(0, 0)) // Unix 纪元

	err := batch.Submit(ctx, request)
	if err != nil {
		t.Errorf("Should handle zero time values: %v", err)
	}
}

func TestBoundary_ManyColumns(t *testing.T) {
	ctx := context.Background()
	config := batchsql.PipelineConfig{
		BufferSize:    10,
		FlushSize:     5,
		FlushInterval: time.Second,
	}

	batch, _ := batchsql.NewBatchSQLWithMock(ctx, config)

	// 创建有很多列的 schema
	columns := make([]string, 100)
	for i := 0; i < 100; i++ {
		columns[i] = "col" + string(rune('0'+i%10))
	}

	schema := batchsql.NewSchema("test_table", batchsql.ConflictIgnore, columns...)
	request := batchsql.NewRequest(schema)

	// 设置所有列的值
	for i, col := range columns {
		request.SetInt64(col, int64(i))
	}

	err := batch.Submit(ctx, request)
	if err != nil {
		t.Errorf("Should handle many columns: %v", err)
	}
}

func TestBoundary_SingleColumn(t *testing.T) {
	ctx := context.Background()
	config := batchsql.PipelineConfig{
		BufferSize:    10,
		FlushSize:     5,
		FlushInterval: time.Second,
	}

	batch, _ := batchsql.NewBatchSQLWithMock(ctx, config)

	// 只有一列的 schema
	schema := batchsql.NewSchema("test_table", batchsql.ConflictIgnore, "single_col")
	request := batchsql.NewRequest(schema).SetString("single_col", "value")

	err := batch.Submit(ctx, request)
	if err != nil {
		t.Errorf("Should handle single column: %v", err)
	}
}

func TestBoundary_BufferSizeOne(t *testing.T) {
	ctx := context.Background()
	config := batchsql.PipelineConfig{
		BufferSize:    1, // 最小缓冲区
		FlushSize:     1, // 最小刷新大小
		FlushInterval: time.Second,
	}

	batch, _ := batchsql.NewBatchSQLWithMock(ctx, config)

	schema := batchsql.NewSchema("test_table", batchsql.ConflictIgnore, "id")
	request := batchsql.NewRequest(schema).SetInt64("id", 1)

	err := batch.Submit(ctx, request)
	if err != nil {
		t.Errorf("Should handle buffer size 1: %v", err)
	}

	// 等待处理完成
	time.Sleep(100 * time.Millisecond)
}

func TestBoundary_VeryShortFlushInterval(t *testing.T) {
	ctx := context.Background()
	config := batchsql.PipelineConfig{
		BufferSize:    10,
		FlushSize:     5,
		FlushInterval: time.Nanosecond, // 极短的刷新间隔
	}

	batch, _ := batchsql.NewBatchSQLWithMock(ctx, config)

	schema := batchsql.NewSchema("test_table", batchsql.ConflictIgnore, "id")

	// 快速提交多个请求
	for i := 0; i < 10; i++ {
		request := batchsql.NewRequest(schema).SetInt64("id", int64(i))
		err := batch.Submit(ctx, request)
		if err != nil {
			t.Errorf("Should handle very short flush interval: %v", err)
		}
	}

	// 等待处理完成
	time.Sleep(100 * time.Millisecond)
}
