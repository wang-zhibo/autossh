package app

import (
	"golang.org/x/crypto/ssh"
	"golang.org/x/net/proxy"
	"net"
	"os"
	"testing"
)

func TestServer_Connect(t *testing.T) {
	if os.Getenv("AUTOSSH_INTEGRATION") != "1" {
		t.Skip("set AUTOSSH_INTEGRATION=1 to run integration test")
	}

	var server = Server{
		Ip:     "172.18.36.217",
		Method: "key",
	}
	auth, err := parseAuthMethods(&server)
	if err != nil {
		t.Fatal(err)
	}
	sshConfig := &ssh.ClientConfig{
		User: "work",
		Auth: auth,
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
		// Auth: .... fill out with keys etc as normal
	}

	client, err := proxiedSSHClient("127.0.0.1:1080", "172.18.36.217:22", sshConfig)
	if err != nil {
		t.Fatal(err)
	}

	session, _ := client.NewSession()
	output, _ := session.CombinedOutput("ls")
	t.Log(string(output))

	// get a session etc...

}

func proxiedSSHClient(proxyAddress, sshServerAddress string, sshConfig *ssh.ClientConfig) (*ssh.Client, error) {
	dialer, err := proxy.SOCKS5("tcp", proxyAddress, nil, proxy.Direct)
	if err != nil {
		return nil, err
	}

	conn, err := dialer.Dial("tcp", sshServerAddress)
	if err != nil {
		return nil, err
	}

	c, chans, reqs, err := ssh.NewClientConn(conn, sshServerAddress, sshConfig)
	if err != nil {
		return nil, err
	}

	return ssh.NewClient(c, chans, reqs), nil
}
