package main

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/bprendie/weazlchat/internal/storage"
)

func TestUnlockFromInheritedFD(t *testing.T) {
	store, err := storage.Open(filepath.Join(t.TempDir(), "vault.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err := store.Migrate(); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateVault("long enough password"); err != nil {
		t.Fatal(err)
	}

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := writer.WriteString("long enough password"); err != nil {
		t.Fatal(err)
	}
	_ = writer.Close()
	t.Setenv("WEAZL_VAULT_KEY_FD", strconv.Itoa(int(reader.Fd())))

	if err := unlockFromInheritedFD(store); err != nil {
		t.Fatalf("unlock: %v", err)
	}
	if !store.Unlocked() {
		t.Fatal("store is locked")
	}
}
