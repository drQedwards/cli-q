package mcp

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCreateZip_NonGitDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main"), 0600); err != nil {
		t.Fatal(err)
	}
	path, err := createZip(dir)
	if err != nil {
		t.Fatalf("createZip: %v", err)
	}
	defer os.Remove(path)
	if _, err := os.Stat(path); err != nil {
		t.Errorf("zip file not created: %v", err)
	}
}

func TestCreateZip_NonExistentDir(t *testing.T) {
	_, err := createZip("/nonexistent-dir-mcp-createzip-xyz")
	if err == nil {
		t.Error("createZip should fail when directory does not exist")
	}
}
