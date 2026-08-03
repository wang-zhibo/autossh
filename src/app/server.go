package app

import (
	"autossh/src/utils"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
	"golang.org/x/net/proxy"
	"golang.org/x/term"
)

type Server struct {
	Name     string                 `json:"name"`
	Ip       string                 `json:"ip"`
	Port     int                    `json:"port"`
	User     string                 `json:"user"`
	Password string                 `json:"password"`
	Method   string                 `json:"method"`
	Key      string                 `json:"key"`
	Options  map[string]interface{} `json:"options"`
	Alias    string                 `json:"alias"`
	Log      ServerLog              `json:"log"`

	termWidth  int
	termHeight int
	groupName  string
	group      *Group
}

// 格式化，赋予默认值
func (server *Server) Format() {
	if server.Port == 0 {
		server.Port = 22
	}

	server.Method = strings.ToLower(strings.TrimSpace(server.Method))
	if server.Method == "" {
		server.Method = "password"
	}
}

// 合并选项
func (server *Server) MergeOptions(options map[string]interface{}, overwrite bool) {
	if server.Options == nil {
		server.Options = make(map[string]interface{})
	}

	for k, v := range options {
		if overwrite {
			server.Options[k] = v
		} else {
			if _, ok := server.Options[k]; !ok {
				server.Options[k] = v
			}
		}
	}
}

// 格式化输出，用于打印 - 修复版本
func (server *Server) FormatPrint(flag string, ShowDetail bool) string {
	var builder strings.Builder
	builder.WriteString(" [")
	builder.WriteString(flag)

	if server.Alias != "" {
		builder.WriteString("|")
		builder.WriteString(server.Alias)
	}
	builder.WriteString("]    ") // 替换制表符为固定空格
	builder.WriteString(server.Name)

	if ShowDetail {
		builder.WriteString(" [")
		builder.WriteString(server.User)
		builder.WriteString("@")
		builder.WriteString(server.Ip)
		builder.WriteString("]")
	}

	return builder.String()
}

// 获取连接超时时间
func (server *Server) getConnectTimeout() time.Duration {
	if val, ok := server.Options["ConnectTimeout"]; ok && val != nil {
		if timeout, ok := val.(float64); ok {
			if timeout > 0 {
				return time.Duration(timeout) * time.Second
			}
		}
	}
	return 30 * time.Second // 默认30秒超时
}

func (server *Server) shouldSkipHostKeyCheck() bool {
	if insecureSkipHostKeyCheck {
		return true
	}

	if server.Options == nil {
		return false
	}

	if val, ok := server.Options["InsecureSkipHostKeyChecking"]; ok {
		if b, ok := toBool(val); ok {
			return b
		}
	}
	if val, ok := server.Options["SkipHostKeyCheck"]; ok {
		if b, ok := toBool(val); ok {
			return b
		}
	}
	if val, ok := server.Options["StrictHostKeyChecking"]; ok {
		if b, ok := toBool(val); ok {
			return !b
		}
		if s, ok := val.(string); ok {
			switch strings.ToLower(strings.TrimSpace(s)) {
			case "no", "false", "0", "off":
				return true
			}
		}
	}

	return false
}

func (server *Server) getKnownHostsFile() (string, error) {
	if server.Options != nil {
		if val, ok := server.Options["KnownHostsFile"]; ok {
			if s, ok := val.(string); ok && strings.TrimSpace(s) != "" {
				return utils.ParsePath(s)
			}
		}
	}
	return utils.ParsePath("~/.ssh/known_hosts")
}

func (server *Server) getHostKeyCallback() (ssh.HostKeyCallback, error) {
	if server.shouldSkipHostKeyCheck() {
		return ssh.InsecureIgnoreHostKey(), nil
	}

	knownHostsFile, err := server.getKnownHostsFile()
	if err != nil {
		return nil, fmt.Errorf("解析 KnownHostsFile 失败: %w", err)
	}

	if _, err := os.Stat(knownHostsFile); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("known_hosts 文件不存在: %s（可使用 --insecure 跳过校验）", knownHostsFile)
		}
		return nil, fmt.Errorf("读取 known_hosts 失败: %w", err)
	}

	cb, err := knownhosts.New(knownHostsFile)
	if err != nil {
		return nil, fmt.Errorf("解析 known_hosts 失败: %w", err)
	}

	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		if err := cb(hostname, remote, key); err != nil {
			fp := ssh.FingerprintSHA256(key)
			return fmt.Errorf("SSH HostKey 校验失败: %s (%s): %w（指纹: %s，known_hosts: %s，可用 --insecure 跳过）", hostname, remote.String(), err, fp, knownHostsFile)
		}
		return nil
	}, nil
}

func toBool(v interface{}) (bool, bool) {
	switch x := v.(type) {
	case bool:
		return x, true
	case float64:
		return x != 0, true
	case int:
		return x != 0, true
	case string:
		switch strings.ToLower(strings.TrimSpace(x)) {
		case "1", "true", "yes", "y", "on":
			return true, true
		case "0", "false", "no", "n", "off":
			return false, true
		default:
			return false, false
		}
	default:
		return false, false
	}
}

func (server *Server) address() string {
	port := server.Port
	if port == 0 {
		port = 22
	}
	return net.JoinHostPort(server.Ip, strconv.Itoa(port))
}

// GetSshClient 为当前操作创建独立的 SSH 连接。交互连接和 SFTP 传输都会在完成后
// 显式关闭该连接，避免跨服务器配置、认证方式或代理复用错误的全局状态。
func (server *Server) GetSshClient() (*ssh.Client, error) {
	auth, err := parseAuthMethods(server)
	if err != nil {
		return nil, fmt.Errorf("解析认证方法失败: %w", err)
	}

	hostKeyCallback, err := server.getHostKeyCallback()
	if err != nil {
		return nil, err
	}

	config := &ssh.ClientConfig{
		User:            server.User,
		Auth:            auth,
		HostKeyCallback: hostKeyCallback,
	}

	addr := server.address()
	timeout := server.getConnectTimeout()
	var conn net.Conn
	if server.group != nil && server.group.Proxy != nil {
		conn, err = server.proxyDial(server.group.Proxy, addr, timeout)
	} else {
		conn, err = (&net.Dialer{Timeout: timeout}).Dial("tcp", addr)
	}
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			_ = conn.Close()
		}
	}()

	if err = conn.SetDeadline(time.Now().Add(timeout)); err != nil {
		return nil, fmt.Errorf("设置 SSH 连接超时失败: %w", err)
	}
	clientConn, channels, requests, err := ssh.NewClientConn(conn, addr, config)
	if err != nil {
		return nil, fmt.Errorf("创建 SSH 客户端连接失败: %w", err)
	}
	if err = conn.SetDeadline(time.Time{}); err != nil {
		_ = clientConn.Close()
		return nil, fmt.Errorf("清除 SSH 连接超时失败: %w", err)
	}

	return ssh.NewClient(clientConn, channels, requests), nil
}

type timeoutDialer struct {
	timeout time.Duration
}

func (dialer timeoutDialer) Dial(network string, address string) (net.Conn, error) {
	conn, err := net.DialTimeout(network, address, dialer.timeout)
	if err != nil {
		return nil, err
	}
	if err := conn.SetDeadline(time.Now().Add(dialer.timeout)); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return conn, nil
}

func (server *Server) proxyDial(p *Proxy, sshServerAddr string, timeout time.Duration) (net.Conn, error) {
	var dialer proxy.Dialer
	switch p.Type {
	case ProxyTypeSocks5:
		var auth *proxy.Auth
		if p.User != "" {
			auth = &proxy.Auth{
				User:     p.User,
				Password: p.Password,
			}
		}

		var err error
		dialer, err = proxy.SOCKS5("tcp", net.JoinHostPort(p.Server, strconv.Itoa(p.Port)), auth, timeoutDialer{timeout: timeout})
		if err != nil {
			return nil, fmt.Errorf("创建SOCKS5代理失败: %w", err)
		}
	default:
		return nil, fmt.Errorf("不支持的代理类型: %s", p.Type)
	}

	conn, err := dialer.Dial("tcp", sshServerAddr)
	if err != nil {
		return nil, fmt.Errorf("通过代理连接失败: %w", err)
	}
	return conn, nil
}

type SftpConnection struct {
	Client    *sftp.Client
	sshClient *ssh.Client
}

func (connection *SftpConnection) Close() error {
	if connection == nil {
		return nil
	}
	var closeErrors []error
	if connection.Client != nil {
		closeErrors = append(closeErrors, connection.Client.Close())
	}
	if connection.sshClient != nil {
		closeErrors = append(closeErrors, connection.sshClient.Close())
	}
	return errors.Join(closeErrors...)
}

// GetSftpClient 创建 SFTP 会话及其底层 SSH 连接。调用方必须关闭返回的连接。
func (server *Server) GetSftpClient() (*SftpConnection, error) {
	sshClient, err := server.GetSshClient()
	if err != nil {
		return nil, err
	}

	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		_ = sshClient.Close()
		return nil, fmt.Errorf("创建SFTP客户端失败: %w", err)
	}

	return &SftpConnection{Client: sftpClient, sshClient: sshClient}, nil
}

// 执行远程连接
func (server *Server) Connect() error {
	// 美化连接过程显示 - 确保左对齐
	fmt.Print("🔗 建立SSH连接中...\n")

	client, err := server.GetSshClient()
	if err != nil {
		errorType := utils.GetErrorType(err)
		switch errorType {
		case "authentication_failed":
			return errors.New("认证失败，请检查用户名和密码/密钥")
		case "connection_refused":
			return errors.New("连接被拒绝，请检查服务器地址和端口")
		case "timeout":
			return errors.New("连接超时，请检查网络连接")
		case "network_error":
			return errors.New("网络错误，请检查网络连接")
		default:
			return fmt.Errorf("SSH连接失败: %w", err)
		}
	}
	defer client.Close()

	fmt.Print("📡 创建SSH会话...\n")
	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("创建SSH会话失败: %w", err)
	}
	defer session.Close()

	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return fmt.Errorf("设置终端原始模式失败: %w", err)
	}
	defer term.Restore(fd, oldState)

	stopKeepAliveLoop := server.startKeepAliveLoop(client)
	defer stopKeepAliveLoop()

	err = server.stdIO(session)
	if err != nil {
		return fmt.Errorf("设置标准IO失败: %w", err)
	}

	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}

	server.termWidth, server.termHeight, _ = term.GetSize(fd)
	termType := os.Getenv("TERM")
	if termType == "" {
		termType = "xterm-256color"
	}
	if err := session.RequestPty(termType, server.termHeight, server.termWidth, modes); err != nil {
		return fmt.Errorf("请求PTY失败: %w", err)
	}

	stopWindowChange := server.listenWindowChange(session, fd)
	defer stopWindowChange()

	// 连接成功提示 - 简化版本
	fmt.Println("✅ SSH连接已建立，正在启动Shell...")
	fmt.Println()

	err = session.Shell()
	if err != nil {
		return fmt.Errorf("启动Shell失败: %w", err)
	}

	if err := session.Wait(); err != nil {
		return fmt.Errorf("SSH会话异常结束: %w", err)
	}
	return nil
}

// 重定向标准输入输出
func (server *Server) stdIO(session *ssh.Session) error {
	session.Stderr = os.Stderr
	session.Stdin = os.Stdin

	if server.Log.Enable {
		ch, err := session.StdoutPipe()
		if err != nil {
			return fmt.Errorf("获取标准输出管道失败: %w", err)
		}

		go func() {
			flag := os.O_WRONLY | os.O_CREATE | os.O_APPEND
			if server.Log.Mode == LogModeCover {
				flag = os.O_WRONLY | os.O_CREATE | os.O_TRUNC
			}

			logFile := server.formatLogFilename(server.Log.Filename)
			f, err := os.OpenFile(logFile, flag, 0600)
			if err != nil {
				utils.Logln(fmt.Sprintf("打开日志文件失败: %v", err))
				return
			}
			defer f.Close()

			utils.Logln(fmt.Sprintf("开始记录会话日志到: %s", logFile))

			buff := make([]byte, 4096)
			for {
				n, err := ch.Read(buff)
				if n > 0 {
					if _, err := f.Write(buff[:n]); err != nil {
						utils.Logln(fmt.Sprintf("写入日志文件失败: %v", err))
					}

					if _, err := os.Stdout.Write(buff[:n]); err != nil {
						utils.Logln(fmt.Sprintf("写入标准输出失败: %v", err))
					}
				}
				if err != nil {
					break
				}
			}
		}()
	} else {
		session.Stdout = os.Stdout
	}

	return nil
}

// 格式化日志文件名
func (server *Server) formatLogFilename(filename string) string {
	kvs := []map[string]string{
		{"%g": server.groupName},
		{"%n": server.Name},
		{"%dt": time.Now().Format("20060102150405")},
		{"%d": time.Now().Format("20060102")},
		{"%u": server.User},
		{"%a": server.Alias},
	}

	for _, kv := range kvs {
		for k, v := range kv {
			filename = strings.ReplaceAll(filename, k, v)
		}
	}

	return filename
}

// 解析鉴权方式
func parseAuthMethods(server *Server) ([]ssh.AuthMethod, error) {
	var authMethods []ssh.AuthMethod

	switch strings.ToLower(server.Method) {
	case "password":
		if server.Password == "" {
			return nil, errors.New("密码认证模式下密码不能为空")
		}
		authMethods = append(authMethods, ssh.Password(server.Password))

	case "key":
		method, err := pemKey(server)
		if err != nil {
			return nil, err
		}
		authMethods = append(authMethods, method)

	default:
		return nil, fmt.Errorf("不支持的认证方法: %s", server.Method)
	}

	return authMethods, nil
}

// 解析密钥
func pemKey(server *Server) (ssh.AuthMethod, error) {
	if server.Key == "" {
		server.Key = "~/.ssh/id_rsa"
	}
	keyPath, err := utils.ParsePath(server.Key)
	if err != nil {
		return nil, fmt.Errorf("解析密钥文件路径失败: %w", err)
	}
	server.Key = keyPath

	pemBytes, err := ioutil.ReadFile(server.Key)
	if err != nil {
		return nil, fmt.Errorf("读取密钥文件失败: %w", err)
	}

	var signer ssh.Signer
	if server.Password == "" {
		signer, err = ssh.ParsePrivateKey(pemBytes)
	} else {
		signer, err = ssh.ParsePrivateKeyWithPassphrase(pemBytes, []byte(server.Password))
	}

	if err != nil {
		return nil, fmt.Errorf("解析密钥失败: %w", err)
	}

	return ssh.PublicKeys(signer), nil
}

func (server *Server) getKeepAliveInterval() time.Duration {
	if server.Options == nil {
		return 0
	}
	if value, ok := server.Options["ServerAliveInterval"].(float64); ok && value > 0 {
		return time.Duration(value) * time.Second
	}
	return 0
}

func (server *Server) getServerAliveCountMax() int {
	if server.Options != nil {
		if value, ok := server.Options["ServerAliveCountMax"].(float64); ok && value > 0 {
			return int(value)
		}
	}
	return 3
}

// startKeepAliveLoop sends standard SSH global keepalive requests. The returned
// function is idempotent for the single owner in Connect and stops the goroutine
// before the session and client are closed.
func (server *Server) startKeepAliveLoop(client *ssh.Client) func() {
	interval := server.getKeepAliveInterval()
	if interval <= 0 {
		return func() {}
	}

	terminate := make(chan struct{})
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		failedChecks := 0
		for {
			select {
			case <-terminate:
				return
			case <-ticker.C:
				if _, _, err := client.SendRequest("keepalive@openssh.com", true, nil); err != nil {
					failedChecks++
					utils.Errorf("发送心跳包失败 (%d/%d): %v", failedChecks, server.getServerAliveCountMax(), err)
					if failedChecks >= server.getServerAliveCountMax() {
						_ = client.Close()
						return
					}
					continue
				}
				failedChecks = 0
			}
		}
	}()

	var stopOnce sync.Once
	return func() {
		stopOnce.Do(func() {
			close(terminate)
		})
	}
}

// listenWindowChange forwards terminal resize events and unregisters signal
// delivery when the SSH session ends.
func (server *Server) listenWindowChange(session *ssh.Session, fd int) func() {
	terminate := make(chan struct{})
	sigwinchCh := make(chan os.Signal, 1)
	signal.Notify(sigwinchCh, syscall.SIGWINCH)

	go func() {
		defer signal.Stop(sigwinchCh)

		termWidth, termHeight, err := term.GetSize(fd)
		if err != nil {
			utils.Errorf("获取终端大小失败: %v", err)
		}

		for {
			select {
			case <-terminate:
				return
			case <-sigwinchCh:
				currTermWidth, currTermHeight, err := term.GetSize(fd)
				if err != nil {
					utils.Errorf("获取当前终端大小失败: %v", err)
					continue
				}

				if currTermHeight == termHeight && currTermWidth == termWidth {
					continue
				}

				if err := session.WindowChange(currTermHeight, currTermWidth); err != nil {
					utils.Errorf("更新终端窗口大小失败: %v", err)
					continue
				}

				termWidth, termHeight = currTermWidth, currTermHeight
			}
		}
	}()

	var stopOnce sync.Once
	return func() {
		stopOnce.Do(func() {
			close(terminate)
		})
	}
}
