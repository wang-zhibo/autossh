package app

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCpParseRequiresSourceAndTarget(t *testing.T) {
	cp := Cp{cfg: &Config{serverIndex: map[string]ServerIndex{}}}
	if err := cp.parse([]string{"only-target"}); err == nil {
		t.Fatal("parse() error = nil, want missing source error")
	}
}

func TestNewTransferObjectPreservesColonInRemotePath(t *testing.T) {
	server := &Server{Name: "Web"}
	cfg := &Config{serverIndex: map[string]ServerIndex{
		"prod": {server: server},
	}}

	object, err := newTransferObject(cfg, "PROD:/var/log/app:2026-08-03.log")
	if err != nil {
		t.Fatalf("newTransferObject() error = %v", err)
	}
	if object.server != server {
		t.Fatal("remote server was not resolved")
	}
	if object.path != "/var/log/app:2026-08-03.log" {
		t.Fatalf("path = %q", object.path)
	}
}

func TestCpParseRejectsTwoLocalPaths(t *testing.T) {
	cp := Cp{cfg: &Config{serverIndex: map[string]ServerIndex{}}}
	err := cp.parse([]string{"source.txt", "target.txt"})
	if err == nil || !strings.Contains(err.Error(), "同时为本地地址") {
		t.Fatalf("parse() error = %v, want local-to-local error", err)
	}
}

func TestCpTransferDirectoryPreservesEmptyDirectories(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	target := filepath.Join(root, "target", "nested")
	if err := os.MkdirAll(filepath.Join(source, "empty"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(source, "files"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "files", "hello.txt"), []byte("hello"), 0600); err != nil {
		t.Fatal(err)
	}

	cp := Cp{isDir: true}
	client := &LocalIOClient{}
	if file, err := cp.transferNew(client, client, source, target, ""); err != nil {
		t.Fatalf("transferNew() %s: %v", file, err)
	}

	info, err := os.Stat(filepath.Join(target, "empty"))
	if err != nil {
		t.Fatalf("empty directory was not copied: %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("empty = %v, want directory", info.Mode())
	}
	content, err := os.ReadFile(filepath.Join(target, "files", "hello.txt"))
	if err != nil {
		t.Fatalf("copied file missing: %v", err)
	}
	if string(content) != "hello" {
		t.Fatalf("copied content = %q, want hello", content)
	}
}

func TestCpTransferRejectsSymbolicLinks(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	if err := os.MkdirAll(source, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(".", filepath.Join(source, "loop")); err != nil {
		t.Skipf("symbolic links are unavailable: %v", err)
	}

	cp := Cp{isDir: true}
	client := &LocalIOClient{}
	if _, err := cp.transferNew(client, client, source, filepath.Join(root, "target"), ""); err == nil || !strings.Contains(err.Error(), "符号链接") {
		t.Fatalf("transferNew() error = %v, want symbolic-link rejection", err)
	}
}

func TestCpTransferRemovesPartialFileOnReadFailure(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source.txt")
	if err := os.WriteFile(source, make([]byte, 64*1024+1), 0600); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(root, "target", "result.txt")

	cp := Cp{}
	sourceClient := &failingReadIOClient{LocalIOClient: &LocalIOClient{}}
	if _, err := cp.transferNew(sourceClient, &LocalIOClient{}, source, target, ""); err == nil {
		t.Fatal("transferNew() error = nil, want read failure")
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("partial target exists or stat failed: %v", err)
	}

	entries, err := os.ReadDir(filepath.Dir(target))
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}
	for _, entry := range entries {
		if strings.Contains(entry.Name(), ".autossh-part-") {
			t.Errorf("temporary transfer file was not removed: %s", entry.Name())
		}
	}
}

type failingReadIOClient struct {
	*LocalIOClient
}

func (client *failingReadIOClient) Open(file string) (FileLike, error) {
	opened, err := client.LocalIOClient.Open(file)
	if err != nil {
		return nil, err
	}
	return &failingReadFile{FileLike: opened}, nil
}

type failingReadFile struct {
	FileLike
	readOnce bool
}

func (file *failingReadFile) Read(p []byte) (int, error) {
	if file.readOnce {
		return 0, errors.New("simulated source read failure")
	}
	file.readOnce = true
	return file.FileLike.Read(p)
}
