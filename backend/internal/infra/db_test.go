package infra

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureSQLiteDirCreatesParentDirectory(t *testing.T) {
	tempDir := t.TempDir()
	t.Chdir(tempDir)

	err := ensureSQLiteDir("file:./data/markpost.db?_foreign_keys=on&_journal_mode=WAL")
	if err != nil {
		t.Fatalf("ensureSQLiteDir error: %v", err)
	}

	info, err := os.Stat(filepath.Join(tempDir, "data"))
	if err != nil {
		t.Fatalf("Stat error: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("expected data to be a directory")
	}
}

func TestEnsureSQLiteDirMemoryDSN(t *testing.T) {
	if err := ensureSQLiteDir(":memory:"); err != nil {
		t.Fatalf("unexpected error for :memory: DSN: %v", err)
	}
}

func TestEnsureSQLiteDirFileMemoryDSN(t *testing.T) {
	if err := ensureSQLiteDir("file::memory:?cache=shared"); err != nil {
		t.Fatalf("unexpected error for file::memory: DSN: %v", err)
	}
}

func TestEnsureSQLiteDirBareFilename(t *testing.T) {
	if err := ensureSQLiteDir("markpost.db"); err != nil {
		t.Fatalf("unexpected error for bare filename DSN: %v", err)
	}
}
