package storage

import (
	"path/filepath"
	"testing"
)

func TestMemoryRoundTrip(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "test.sqlite3"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer store.Close()
	if err := store.Migrate(); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if err := store.CreateVault("pw"); err != nil {
		t.Fatalf("CreateVault: %v", err)
	}
	if err := store.Remember("project", "WeazlChat uses local tools.", "chat,tools"); err != nil {
		t.Fatalf("Remember: %v", err)
	}

	memories, err := store.RecallMemories("local tools", 10)
	if err != nil {
		t.Fatalf("RecallMemories: %v", err)
	}
	if len(memories) != 1 {
		t.Fatalf("len(memories) = %d, want 1", len(memories))
	}
	if memories[0].Key != "project" || memories[0].Value != "WeazlChat uses local tools." {
		t.Fatalf("memory = %#v", memories[0])
	}

	if err := store.ForgetMemory("project"); err != nil {
		t.Fatalf("ForgetMemory: %v", err)
	}
	memories, err = store.Memories(10)
	if err != nil {
		t.Fatalf("Memories: %v", err)
	}
	if len(memories) != 0 {
		t.Fatalf("len(memories) = %d, want 0", len(memories))
	}
}
