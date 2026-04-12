package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCopyFile(t *testing.T) {
	src := filepath.Join(os.TempDir(), "selfupdate_test_src.exe")
	dst := filepath.Join(os.TempDir(), "selfupdate_test_dst.exe")
	defer os.Remove(src)
	defer os.Remove(dst)

	// Create a fake exe with MZ header + padding
	data := make([]byte, 2*1024*1024) // 2MB
	data[0] = 'M'
	data[1] = 'Z'
	if err := os.WriteFile(src, data, 0644); err != nil {
		t.Fatal(err)
	}

	if err := copyFile(src, dst); err != nil {
		t.Fatal("copyFile failed:", err)
	}

	info, err := os.Stat(dst)
	if err != nil {
		t.Fatal("dst not created:", err)
	}
	if info.Size() != 2*1024*1024 {
		t.Fatalf("size mismatch: got %d, want %d", info.Size(), 2*1024*1024)
	}
}

func TestVerifyExe_Valid(t *testing.T) {
	path := filepath.Join(os.TempDir(), "selfupdate_test_valid.exe")
	defer os.Remove(path)

	data := make([]byte, 2*1024*1024)
	data[0] = 'M'
	data[1] = 'Z'
	os.WriteFile(path, data, 0644)

	if err := verifyExe(path); err != nil {
		t.Fatal("valid exe rejected:", err)
	}
}

func TestVerifyExe_InvalidHeader(t *testing.T) {
	path := filepath.Join(os.TempDir(), "selfupdate_test_invalid.exe")
	defer os.Remove(path)

	data := make([]byte, 2*1024*1024)
	data[0] = 'P'
	data[1] = 'K' // ZIP header, not exe
	os.WriteFile(path, data, 0644)

	if err := verifyExe(path); err == nil {
		t.Fatal("invalid exe should be rejected")
	}
}

func TestVerifyExe_TooSmall(t *testing.T) {
	path := filepath.Join(os.TempDir(), "selfupdate_test_small.exe")
	defer os.Remove(path)

	data := []byte("MZ tiny")
	os.WriteFile(path, data, 0644)

	if err := verifyExe(path); err == nil {
		t.Fatal("tiny exe should be rejected")
	}
}

func TestVerifyExe_RealExe(t *testing.T) {
	// Test with the actual built exe
	if err := verifyExe("screenshoter_admin.exe"); err != nil {
		t.Fatal("real exe rejected:", err)
	}
}

func TestRenameFlow(t *testing.T) {
	dir := filepath.Join(os.TempDir(), "selfupdate_rename_test")
	os.MkdirAll(dir, 0755)
	defer os.RemoveAll(dir)

	current := filepath.Join(dir, "app.exe")
	newExe := filepath.Join(dir, "app.exe.new")
	oldExe := filepath.Join(dir, "app.exe.old")

	// Create fake "current" and "new" exe
	os.WriteFile(current, []byte("old version"), 0644)
	os.WriteFile(newExe, []byte("new version"), 0644)

	// Simulate the rename flow
	if err := os.Rename(current, oldExe); err != nil {
		t.Fatal("rename current->old failed:", err)
	}
	if err := os.Rename(newExe, current); err != nil {
		t.Fatal("rename new->current failed:", err)
	}

	// Verify
	data, _ := os.ReadFile(current)
	if string(data) != "new version" {
		t.Fatal("current should be new version, got:", string(data))
	}

	data, _ = os.ReadFile(oldExe)
	if string(data) != "old version" {
		t.Fatal("old should be old version, got:", string(data))
	}

	// Cleanup (simulates CleanupOldExe)
	os.Remove(oldExe)
	if _, err := os.Stat(oldExe); !os.IsNotExist(err) {
		t.Fatal("old exe should be deleted")
	}
}
