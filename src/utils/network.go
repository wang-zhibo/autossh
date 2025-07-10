package utils

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

// IsPortOpen 检查端口是否开放
func IsPortOpen(host string, port int, timeout time.Duration) bool {
	address := net.JoinHostPort(host, strconv.Itoa(port))
	conn, err := net.DialTimeout("tcp", address, timeout)
	if err != nil {
		return false
	}
	defer conn.Close()
	return true
}

// ResolveHostname 解析主机名
func ResolveHostname(hostname string) (string, error) {
	ips, err := net.LookupIP(hostname)
	if err != nil {
		return "", fmt.Errorf("解析主机名失败: %w", err)
	}
	if len(ips) == 0 {
		return "", fmt.Errorf("未找到主机名对应的IP地址")
	}
	return ips[0].String(), nil
}

// ValidateIP 验证IP地址格式
func ValidateIP(ip string) bool {
	return net.ParseIP(ip) != nil
}

// ValidatePort 验证端口号
func ValidatePort(port int) bool {
	return port > 0 && port <= 65535
}

// ParseAddress 解析地址字符串 (host:port)
func ParseAddress(address string) (string, int, error) {
	host, portStr, err := net.SplitHostPort(address)
	if err != nil {
		return "", 0, fmt.Errorf("解析地址失败: %w", err)
	}
	
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return "", 0, fmt.Errorf("解析端口号失败: %w", err)
	}
	
	if !ValidatePort(port) {
		return "", 0, fmt.Errorf("端口号无效: %d", port)
	}
	
	return host, port, nil
}

// FormatAddress 格式化地址字符串
func FormatAddress(host string, port int) string {
	return net.JoinHostPort(host, strconv.Itoa(port))
}

// IsLocalAddress 检查是否为本地地址
func IsLocalAddress(host string) bool {
	if host == "localhost" || host == "127.0.0.1" || host == "::1" {
		return true
	}
	
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	
	return ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast()
}

// GetLocalIPs 获取本地IP地址列表
func GetLocalIPs() ([]string, error) {
	var ips []string
	
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil, fmt.Errorf("获取网络接口地址失败: %w", err)
	}
	
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				ips = append(ips, ipnet.IP.String())
			}
		}
	}
	
	return ips, nil
}

// PingHost 简单的网络连通性测试
func PingHost(host string, port int, timeout time.Duration) error {
	address := FormatAddress(host, port)
	conn, err := net.DialTimeout("tcp", address, timeout)
	if err != nil {
		return fmt.Errorf("无法连接到 %s: %w", address, err)
	}
	defer conn.Close()
	return nil
}

// ExtractHostname 从URL或地址中提取主机名
func ExtractHostname(address string) string {
	// 移除协议前缀
	if strings.HasPrefix(address, "ssh://") {
		address = strings.TrimPrefix(address, "ssh://")
	}
	
	// 提取主机名部分
	if colonIndex := strings.Index(address, ":"); colonIndex != -1 {
		return address[:colonIndex]
	}
	
	return address
} 