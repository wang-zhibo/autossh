package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSaveConfigPersistsChangedGroupState(t *testing.T) {
	configFile := filepath.Join(t.TempDir(), "config.json")
	cfg := &Config{
		Groups: []*Group{{
			GroupName: "生产环境",
			Prefix:    "p",
			Collapse:  true,
			Servers: []Server{{
				Name:     "Web",
				Ip:       "192.0.2.10",
				Port:     22,
				User:     "deploy",
				Password: "test-password",
				Method:   "password",
			}},
		}},
		file: configFile,
	}

	if err := cfg.saveConfig(false); err != nil {
		t.Fatalf("saveConfig() error = %v", err)
	}

	content, err := os.ReadFile(configFile)
	if err != nil {
		t.Fatalf("read saved config: %v", err)
	}

	var saved Config
	if err := json.Unmarshal(content, &saved); err != nil {
		t.Fatalf("decode saved config: %v", err)
	}
	if len(saved.Groups) != 1 || !saved.Groups[0].Collapse {
		t.Fatalf("saved collapse state = %#v, want true", saved.Groups)
	}
}

func TestSaveConfigAndBackupUseOwnerOnlyPermissions(t *testing.T) {
	configFile := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(configFile, []byte("{}\n"), 0600); err != nil {
		t.Fatalf("create config fixture: %v", err)
	}

	cfg := &Config{
		Servers: []*Server{{
			Name:     "Web",
			Ip:       "192.0.2.10",
			Port:     22,
			User:     "deploy",
			Password: "test-password",
			Method:   "password",
		}},
		file: configFile,
	}

	if err := cfg.saveConfig(true); err != nil {
		t.Fatalf("saveConfig() error = %v", err)
	}

	assertOwnerOnlyMode(t, configFile)
	backups, err := filepath.Glob(filepath.Join(filepath.Dir(configFile), "config-*.json"))
	if err != nil {
		t.Fatalf("find backup: %v", err)
	}
	if len(backups) != 1 {
		t.Fatalf("backup count = %d, want 1", len(backups))
	}
	assertOwnerOnlyMode(t, backups[0])
}

func assertOwnerOnlyMode(t *testing.T, filename string) {
	t.Helper()
	info, err := os.Stat(filename)
	if err != nil {
		t.Fatalf("stat %s: %v", filename, err)
	}
	if mode := info.Mode().Perm(); mode != 0600 {
		t.Fatalf("permissions for %s = %04o, want 0600", filename, mode)
	}
}
