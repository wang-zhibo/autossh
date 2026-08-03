package utils

import (
	"os"
	"strings"
	"testing"
)

func TestLoggerWritesCategoryAndClosesIdempotently(t *testing.T) {
	filename := t.TempDir() + "/autossh.log"
	logger := NewLogger(filename, DEBUG)
	logger.Category("ssh").Debug("connected to %s", "example.com")

	if err := logger.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if err := logger.Close(); err != nil {
		t.Fatalf("second Close() error = %v", err)
	}

	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if got, want := string(content), "[DEBUG] [ssh] connected to example.com"; !strings.Contains(got, want) {
		t.Errorf("log content = %q, want it to contain %q", got, want)
	}

	info, err := os.Stat(filename)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if info.Mode().Perm()&0o077 != 0 {
		t.Errorf("log mode = %o, want no group/world permissions", info.Mode().Perm())
	}
}

func TestSetLevelControlsDebugOutput(t *testing.T) {
	oldLevel := globalLogLevel.Load()
	t.Cleanup(func() { globalLogLevel.Store(oldLevel) })

	SetLevel(int(INFO))
	if shouldLog(DEBUG) {
		t.Error("debug logs should be disabled at INFO level")
	}

	SetLevel(int(DEBUG))
	if !shouldLog(DEBUG) {
		t.Error("debug logs should be enabled at DEBUG level")
	}

	SetLevel(100)
	if !shouldLog(DEBUG) {
		t.Error("invalid level should not change the current log level")
	}
}
