package store_test

import (
	"path/filepath"
	"testing"

	"github.com/nmarakushev/kv-store/internal/store"
)

func TestCrashRecovery(t *testing.T) {
	walPath := filepath.Join(t.TempDir(), "test_crash.wal")

	wal, err := store.NewWAL(walPath)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}

	db := store.NewShardedStore(256, wal)

	db.Set("persistent_key_1", "value_1", 0)
	db.Set("persistent_key_2", "value_2", 3600)
	db.Delete("persistent_key_1")
	db.Set("persistent_key_3", "value_3", 0)

	db.Close()

	newWAL, err := store.NewWAL(walPath)
	if err != nil {
		t.Fatalf("Failed to reopen WAL: %v", err)
	}

	newDB := store.NewShardedStore(256, newWAL)

	if err := newDB.ReplayWAL(); err != nil {
		t.Fatalf("Failed to replay WAL: %v", err)
	}

	if val, ok := newDB.Get("persistent_key_1"); ok {
		t.Errorf("Expected key_1 to be deleted, but got: %s", val)
	}

	if val, ok := newDB.Get("persistent_key_2"); !ok || val != "value_2" {
		t.Errorf("Expected key_2 to be 'value_2', got: %s, exists: %v", val, ok)
	}

	if val, ok := newDB.Get("persistent_key_3"); !ok || val != "value_3" {
		t.Errorf("Expected key_3 to be 'value_3', got: %s, exists: %v", val, ok)
	}

	newDB.Close()
}
