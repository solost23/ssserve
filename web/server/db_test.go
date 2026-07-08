package main

import (
	"path/filepath"
	"testing"
)

func TestAddStatsBatchesPositiveIncrements(t *testing.T) {
	db, err := initDB(filepath.Join(t.TempDir(), "sub.db"))
	if err != nil {
		t.Fatalf("init db: %v", err)
	}
	defer db.Close()

	if _, err := db.CreateToken("phone", "token-1", "uuid-1", 443, nil, nil); err != nil {
		t.Fatalf("create token 1: %v", err)
	}
	if _, err := db.CreateToken("laptop", "token-2", "uuid-2", 443, nil, nil); err != nil {
		t.Fatalf("create token 2: %v", err)
	}

	if err := db.AddStats(map[string]int64{
		"token-1":  1024,
		"token-2":  2048,
		"skip-0":   0,
		"skip-neg": -1,
	}); err != nil {
		t.Fatalf("add stats: %v", err)
	}
	if err := db.AddStats(map[string]int64{
		"token-1": 512,
		"token-2": 0,
	}); err != nil {
		t.Fatalf("add stats again: %v", err)
	}

	t1, err := db.GetToken("token-1")
	if err != nil {
		t.Fatalf("get token 1: %v", err)
	}
	if t1.UsedBytes != 1536 {
		t.Fatalf("token 1 used_bytes = %d, want 1536", t1.UsedBytes)
	}

	t2, err := db.GetToken("token-2")
	if err != nil {
		t.Fatalf("get token 2: %v", err)
	}
	if t2.UsedBytes != 2048 {
		t.Fatalf("token 2 used_bytes = %d, want 2048", t2.UsedBytes)
	}
}
