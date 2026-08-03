package app

import (
	"crypto/ed25519"
	"crypto/rand"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

func TestServerAddressUsesJoinHostPort(t *testing.T) {
	tests := []struct {
		name   string
		server Server
		want   string
	}{
		{name: "default port", server: Server{Ip: "192.0.2.10"}, want: "192.0.2.10:22"},
		{name: "IPv6", server: Server{Ip: "2001:db8::1", Port: 2200}, want: "[2001:db8::1]:2200"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.server.address(); got != test.want {
				t.Fatalf("address() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestGetConnectTimeoutFallsBackForNonPositiveValues(t *testing.T) {
	server := Server{Options: map[string]interface{}{"ConnectTimeout": float64(0)}}
	if got, want := server.getConnectTimeout(), 30*time.Second; got != want {
		t.Fatalf("getConnectTimeout() = %s, want %s", got, want)
	}
}

func TestKeepAliveOptions(t *testing.T) {
	server := Server{Options: map[string]interface{}{
		"ServerAliveInterval": float64(15),
		"ServerAliveCountMax": float64(5),
	}}
	if got, want := server.getKeepAliveInterval(), 15*time.Second; got != want {
		t.Fatalf("getKeepAliveInterval() = %s, want %s", got, want)
	}
	if got, want := server.getServerAliveCountMax(), 5; got != want {
		t.Fatalf("getServerAliveCountMax() = %d, want %d", got, want)
	}
}

func TestKeepAliveOptionsIgnoreInvalidValues(t *testing.T) {
	server := Server{Options: map[string]interface{}{
		"ServerAliveInterval": float64(0),
		"ServerAliveCountMax": float64(0),
	}}
	if got := server.getKeepAliveInterval(); got != 0 {
		t.Fatalf("getKeepAliveInterval() = %s, want 0", got)
	}
	if got, want := server.getServerAliveCountMax(), 3; got != want {
		t.Fatalf("getServerAliveCountMax() = %d, want %d", got, want)
	}
}

func TestShouldSkipHostKeyCheck(t *testing.T) {
	oldGlobalValue := insecureSkipHostKeyCheck
	t.Cleanup(func() { insecureSkipHostKeyCheck = oldGlobalValue })
	insecureSkipHostKeyCheck = false

	tests := []struct {
		name    string
		options map[string]interface{}
		want    bool
	}{
		{name: "secure by default", want: false},
		{name: "explicit insecure option", options: map[string]interface{}{"InsecureSkipHostKeyChecking": true}, want: true},
		{name: "strict checking enabled", options: map[string]interface{}{"StrictHostKeyChecking": true}, want: false},
		{name: "strict checking disabled", options: map[string]interface{}{"StrictHostKeyChecking": "off"}, want: true},
		{name: "invalid option", options: map[string]interface{}{"StrictHostKeyChecking": "invalid"}, want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := Server{Options: test.options}
			if got := server.shouldSkipHostKeyCheck(); got != test.want {
				t.Fatalf("shouldSkipHostKeyCheck() = %t, want %t", got, test.want)
			}
		})
	}

	insecureSkipHostKeyCheck = true
	if !(&Server{}).shouldSkipHostKeyCheck() {
		t.Fatal("global insecure option should override server configuration")
	}
}

func TestHostKeyCallbackAcceptsKnownHostAndRejectsUnknownKey(t *testing.T) {
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := ssh.NewSignerFromKey(privateKey)
	if err != nil {
		t.Fatal(err)
	}

	knownHostsFile := filepath.Join(t.TempDir(), "known_hosts")
	line := knownhosts.Line([]string{"[example.test]:2222"}, signer.PublicKey())
	if err := os.WriteFile(knownHostsFile, []byte(line+"\n"), 0600); err != nil {
		t.Fatal(err)
	}

	server := Server{Options: map[string]interface{}{"KnownHostsFile": knownHostsFile}}
	callback, err := server.getHostKeyCallback()
	if err != nil {
		t.Fatalf("getHostKeyCallback() error = %v", err)
	}

	remote := &net.TCPAddr{IP: net.ParseIP("192.0.2.10"), Port: 2222}
	if err := callback("[example.test]:2222", remote, signer.PublicKey()); err != nil {
		t.Fatalf("known host was rejected: %v", err)
	}

	_, unknownPrivateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	unknownSigner, err := ssh.NewSignerFromKey(unknownPrivateKey)
	if err != nil {
		t.Fatal(err)
	}
	if err := callback("[example.test]:2222", remote, unknownSigner.PublicKey()); err == nil || !strings.Contains(err.Error(), "HostKey 校验失败") {
		t.Fatalf("unknown host key error = %v, want HostKey validation error", err)
	}
}

func TestParseAuthMethods(t *testing.T) {
	methods, err := parseAuthMethods(&Server{Method: "password", Password: "secret"})
	if err != nil {
		t.Fatalf("password authentication error = %v", err)
	}
	if len(methods) != 1 {
		t.Fatalf("password authentication methods = %d, want 1", len(methods))
	}

	if _, err := parseAuthMethods(&Server{Method: "password"}); err == nil {
		t.Fatal("empty password error = nil, want error")
	}
	if _, err := parseAuthMethods(&Server{Method: "unsupported"}); err == nil {
		t.Fatal("unsupported authentication method error = nil, want error")
	}
}
