package db

import "testing"

func TestOpenAppliesSchema(t *testing.T) {
	conn, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer conn.Close()

	for _, table := range []string{"folders", "documents", "ai_jobs"} {
		var name string
		err := conn.QueryRow(
			"SELECT name FROM sqlite_master WHERE type='table' AND name=?", table,
		).Scan(&name)
		if err != nil {
			t.Errorf("table %q not found: %v", table, err)
		}
	}
}

func TestOpenIsIdempotent(t *testing.T) {
	conn, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer conn.Close()

	if err := applySchema(conn); err != nil {
		t.Fatalf("re-applying schema should be safe (CREATE ... IF NOT EXISTS): %v", err)
	}
}
