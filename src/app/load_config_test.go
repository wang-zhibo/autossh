package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfigAppliesServerDefaultsBeforeValidation(t *testing.T) {
	configFile := writeConfig(t, `{
		"servers": [{
			"name": "Web",
			"ip": "192.0.2.10",
			"user": "deploy",
			"password": "secret"
		}]
	}`)

	cfg, err := loadConfig(configFile)
	if err != nil {
		t.Fatalf("loadConfig() error = %v", err)
	}
	server := cfg.Servers[0]
	if server.Port != 22 || server.Method != "password" {
		t.Fatalf("defaults = port %d, method %q; want 22 and password", server.Port, server.Method)
	}
}

func TestLoadConfigRejectsDuplicateLookupKeys(t *testing.T) {
	configFile := writeConfig(t, `{
		"servers": [
			{"name":"Web 1","ip":"192.0.2.10","port":22,"user":"deploy","password":"one","method":"password","alias":"prod"},
			{"name":"Web 2","ip":"192.0.2.11","port":22,"user":"deploy","password":"two","method":"password","alias":"PROD"}
		]
	}`)

	_, err := loadConfig(configFile)
	if err == nil || !strings.Contains(err.Error(), "重复标识") {
		t.Fatalf("loadConfig() error = %v, want duplicate lookup key error", err)
	}
}

func TestLoadConfigRejectsTrailingJSON(t *testing.T) {
	configFile := writeConfig(t, `{"servers": []} {}`)

	_, err := loadConfig(configFile)
	if err == nil || !strings.Contains(err.Error(), "多余内容") {
		t.Fatalf("loadConfig() error = %v, want trailing JSON error", err)
	}
}

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	filename := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(filename, []byte(content), 0600); err != nil {
		t.Fatalf("write config fixture: %v", err)
	}
	return filename
}
