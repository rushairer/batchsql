package batchsql_test

import (
	"testing"
	"time"

	"github.com/rushairer/batchsql"
)

func TestRequest_Setters_Getters_Validate(t *testing.T) {
	s := batchsql.NewSchema("t", batchsql.ConflictStrategy(0), "i32", "i64", "f32", "f64", "s", "b", "ts", "bin")
	r := batchsql.NewRequest(s).
		SetInt32("i32", 1).
		SetInt64("i64", 2).
		SetFloat32("f32", 3.5).
		SetFloat64("f64", 4.5).
		SetString("s", "x").
		SetBool("b", true).
		SetTime("ts", time.Unix(0, 0)).
		SetBytes("bin", []byte{1, 2, 3})
	// 还测试通用 Set/SetNull 不破坏 Columns()
	r.Set("custom", 123).SetNull("custom")

	cols := r.Columns()
	if len(cols) < 9 {
		t.Fatalf("Columns len=%d, want >=9", len(cols))
	}

	if v, err := r.GetInt32("i32"); err != nil || v != 1 {
		t.Fatalf("GetInt32=%v,%v", v, err)
	}
	if v, err := r.GetInt64("i64"); err != nil || v != 2 {
		t.Fatalf("GetInt64=%v,%v", v, err)
	}
	if v, err := r.GetFloat64("f64"); err != nil || v != 4.5 {
		t.Fatalf("GetFloat64=%v,%v", v, err)
	}
	if v, err := r.GetBool("b"); err != nil || v != true {
		t.Fatalf("GetBool=%v,%v", v, err)
	}
	if v, err := r.GetTime("ts"); err != nil || v.IsZero() {
		t.Fatalf("GetTime zero=%v, err=%v", v.IsZero(), err)
	}

	if err := r.Validate(); err != nil {
		t.Fatalf("Validate err=%v", err)
	}

	// 未设置的 schema 列应该在 GetX 时报错
	if _, err := r.GetInt32("not_exists"); err == nil {
		t.Fatalf("expect error for missing column")
	}
}
