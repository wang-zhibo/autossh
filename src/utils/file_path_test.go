package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParsePathRejectsEmptyPath(t *testing.T) {
	if _, err := ParsePath(""); err == nil {
		t.Fatal("ParsePath(\"\") error = nil, want error")
	}
}

func TestParsePathExpandsCurrentUserHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	got, err := ParsePath("~/.ssh/id_ed25519")
	if err != nil {
		t.Fatalf("ParsePath() error = %v", err)
	}
	want := filepath.Join(home, ".ssh", "id_ed25519")
	if got != want {
		t.Fatalf("ParsePath() = %q, want %q", got, want)
	}
}

func TestParsePathRejectsOtherUserHome(t *testing.T) {
	if _, err := ParsePath("~another-user/key"); err == nil {
		t.Fatal("ParsePath(~another-user/key) error = nil, want error")
	}
}

func TestFileIsExistsReturnsFalseForMissingFile(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing")
	exists, err := FileIsExists(missing)
	if err != nil {
		t.Fatalf("FileIsExists() error = %v", err)
	}
	if exists {
		t.Fatal("FileIsExists() = true, want false")
	}

	if err := os.WriteFile(missing, []byte("ok"), 0600); err != nil {
		t.Fatal(err)
	}
	exists, err = FileIsExists(missing)
	if err != nil {
		t.Fatalf("FileIsExists() error = %v", err)
	}
	if !exists {
		t.Fatal("FileIsExists() = false, want true")
	}
}
