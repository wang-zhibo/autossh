package app

import (
	"autossh/src/utils"
	"errors"
	"fmt"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
	"golang.org/x/net/proxy"
	"io/ioutil"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

// 连接池管理
var (
	connectionPool = make(map[string]*ssh.Client)
	poolMutex      sync.RWMutex
	poolCleanup    = time.NewTicker(5 * time.Minute)
)

func init() {
	// 启动连接池清理协程
	go func() {
		for range poolCleanup.C {
			cleanupConnectionPool()
		}
	}()
}

// 清理连接池中的无效连接
func cleanupConnectionPool() {
	poolMutex.Lock()
	defer poolMutex.Unlock()
	
	for key, client := range connectionPool {
		// 测试连接是否还有效
		session, err := client.NewSession()
		if err != nil {
			client.Close()
			delete(connectionPool, key)
			continue
		}
		session.Close()
	}
}

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

// 格式化输出，用于打印 - 优化字符串拼接
func (server *Server) FormatPrint(flag string, ShowDetail bool) string {
	var builder strings.Builder
	builder.WriteString(" [")
	builder.WriteString(flag)
	
	if server.Alias != "" {
		builder.WriteString("|")
		builder.WriteString(server.Alias)
	}
	builder.WriteString("]\t")
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
			return time.Duration(timeout) * time.Second
		}
	}
	return 30 * time.Second // 默认30秒超时
}

// 生成连接键用于连接池
func (server *Server) getConnectionKey() string {
	return fmt.Sprintf("%s@%s:%d", server.User, server.Ip, server.Port)
}

// 从连接池获取或创建SSH Client - 优化版本
func (server *Server) GetSshClient() (*ssh.Client, error) {
	connectionKey := server.getConnectionKey()
	
	// 尝试从连接池获取现有连接
	poolMutex.RLock()
	if client, exists := connectionPool[connectionKey]; exists {
		poolMutex.RUnlock()
		
		// 测试连接是否有效
		session, err := client.NewSession()
		if err == nil {
			session.Close()
			return client, nil
		}
		
		// 连接无效，从池中移除
		poolMutex.Lock()
		delete(connectionPool, connectionKey)
		client.Close()
		poolMutex.Unlock()
	} else {
		poolMutex.RUnlock()
	}
	
	// 创建新连接
	auth, err := parseAuthMethods(server)
	if err != nil {
		return nil, fmt.Errorf("解析认证方法失败: %w", err)
	}

	config := &ssh.ClientConfig{
		User: server.User,
		Auth: auth,
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
		Timeout: server.getConnectTimeout(),
	}

	// 默认端口为22
	if server.Port == 0 {
		server.Port = 22
	}

	addr := server.Ip + ":" + strconv.Itoa(server.Port)

	var client *ssh.Client
	if server.group != nil && server.group.Proxy != nil {
		client, err = server.proxySshClient(server.group.Proxy, addr, config)
	} else {
		client, err = ssh.Dial("tcp", addr, config)
	}
	
	if err != nil {
		return nil, err
	}
	
	// 将新连接添加到池中
	poolMutex.Lock()
	connectionPool[connectionKey] = client
	poolMutex.Unlock()
	
	return client, nil
}

func (server *Server) proxySshClient(p *Proxy, sshServerAddr string, sshConfig *ssh.ClientConfig) (client *ssh.Client, err error) {
	var dialer proxy.Dialer
	switch p.Type {
	case ProxyTypeSocks5:
		var auth proxy.Auth
		if p.User != "" {
			auth = proxy.Auth{
				User:     p.User,
				Password: p.Password,
			}
		}

		dialer, err = proxy.SOCKS5("tcp", p.Server+":"+strconv.Itoa(p.Port), &auth, proxy.Direct)
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

	c, chans, reqs, err := ssh.NewClientConn(conn, sshServerAddr, sshConfig)
	if err != nil {
		return nil, fmt.Errorf("创建SSH客户端连接失败: %w", err)
	}

	return ssh.NewClient(c, chans, reqs), nil
}

// 生成Sftp Client
func (server *Server) GetSftpClient() (*sftp.Client, error) {
	sshClient, err := server.GetSshClient()
	if err != nil {
		return nil, err
	}
	
	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		sshClient.Close()
		return nil, fmt.Errorf("创建SFTP客户端失败: %w", err)
	}
	
	return sftpClient, nil
}

// 执行远程连接
func (server *Server) Connect() error {
	utils.Logln(fmt.Sprintf("正在连接到服务器: %s@%s:%d", server.User, server.Ip, server.Port))
	
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

	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("创建SSH会话失败: %w", err)
	}
	defer session.Close()

	fd := int(os.Stdin.Fd())
	oldState, err := terminal.MakeRaw(fd)
	if err != nil {
		return fmt.Errorf("设置终端原始模式失败: %w", err)
	}
	defer terminal.Restore(fd, oldState)

	stopKeepAliveLoop := server.startKeepAliveLoop(session)
	defer close(stopKeepAliveLoop)

	err = server.stdIO(session)
	if err != nil {
		return fmt.Errorf("设置标准IO失败: %w", err)
	}

	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}

	server.termWidth, server.termHeight, _ = terminal.GetSize(fd)
	termType := os.Getenv("TERM")
	if termType == "" {
		termType = "xterm-256color"
	}
	if err := session.RequestPty(termType, server.termHeight, server.termWidth, modes); err != nil {
		return fmt.Errorf("请求PTY失败: %w", err)
	}

	server.listenWindowChange(session, fd)

	utils.Logln("SSH连接已建立")
	err = session.Shell()
	if err != nil {
		return fmt.Errorf("启动Shell失败: %w", err)
	}

	_ = session.Wait()
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
			flag := os.O_RDWR | os.O_CREATE
			switch server.Log.Mode {
			case LogModeAppend:
				flag = flag | os.O_APPEND
			case LogModeCover:
			}

			logFile := server.formatLogFilename(server.Log.Filename)
			f, err := os.OpenFile(logFile, flag, 0644)
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
	server.Key, _ = utils.ParsePath(server.Key)

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

// 发送心跳包
func (server *Server) startKeepAliveLoop(session *ssh.Session) chan struct{} {
	terminate := make(chan struct{})
	go func() {
		for {
			select {
			case <-terminate:
				return
			default:
				if val, ok := server.Options["ServerAliveInterval"]; ok && val != nil {
					_, err := session.SendRequest("keepalive@bbr", true, nil)
					if err != nil {
						utils.Error("发送心跳包失败: %v", err)
					}

					if interval, ok := val.(float64); ok {
						time.Sleep(time.Duration(interval) * time.Second)
					} else {
						time.Sleep(60 * time.Second) // 默认60秒
					}
				} else {
					return
				}
			}
		}
	}()
	return terminate
}

// 监听终端窗口变化
func (server *Server) listenWindowChange(session *ssh.Session, fd int) {
	go func() {
		sigwinchCh := make(chan os.Signal, 1)
		defer close(sigwinchCh)

		signal.Notify(sigwinchCh, syscall.SIGWINCH)
		termWidth, termHeight, err := terminal.GetSize(fd)
		if err != nil {
			utils.Error("获取终端大小失败: %v", err)
		}

		for {
			select {
			case sigwinch := <-sigwinchCh:
				if sigwinch == nil {
					return
				}
				currTermWidth, currTermHeight, err := terminal.GetSize(fd)

				// 判断一下窗口尺寸是否有改变
				if currTermHeight == termHeight && currTermWidth == termWidth {
					continue
				}

				// 更新远端大小
				err = session.WindowChange(currTermHeight, currTermWidth)
				if err != nil {
					utils.Error("更新终端窗口大小失败: %v", err)
					continue
				}

				termWidth, termHeight = currTermWidth, currTermHeight
			}
		}
	}()
}
