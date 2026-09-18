package observability

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func TestDBCollectorCollect(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("sql.Open err = %v", err)
	}
	defer db.Close()

	c := NewDBCollector(db, "test")
	c.Collect() // 不 panic 即通过
}

func TestDBCollectorCollectNilDB(t *testing.T) {
	c := NewDBCollector(nil, "test")
	c.Collect() // nil db 直接返回，不 panic
}

func TestCollectRuntime(t *testing.T) {
	CollectRuntime() // 不 panic 即通过
}
