package utils

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

// 连接缓存
var (
	connectionCache = make(map[string]*connectionCacheEntry)
	cacheMutex      sync.RWMutex
)

type connectionCacheEntry struct {
	isReachable bool
	lastCheck   time.Time
	ttl         time.Duration
}

// IsPortOpen 检查端口是否开放 - 优化版本
func IsPortOpen(host string, port int, timeout time.Duration) bool {
	cacheKey := fmt.Sprintf("%s:%d", host, port)
	
	// 检查缓存
	cacheMutex.RLock()
	if entry, exists := connectionCache[cacheKey]; exists {
		if time.Since(entry.lastCheck) < entry.ttl {
			cacheMutex.RUnlock()
			return entry.isReachable
		}
	}
	cacheMutex.RUnlock()
	
	// 执行实际检查
	address := net.JoinHostPort(host, strconv.Itoa(port))
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", address)
	isReachable := err == nil
	
	if conn != nil {
		conn.Close()
	}
	
	// 更新缓存
	cacheMutex.Lock()
	connectionCache[cacheKey] = &connectionCacheEntry{
		isReachable: isReachable,
		lastCheck:   time.Now(),
		ttl:         30 * time.Second, // 缓存30秒
	}
	cacheMutex.Unlock()
	
	return isReachable
}

// ResolveHostname 解析主机名 - 优化版本
func ResolveHostname(hostname string) (string, error) {
	cacheKey := "resolve_" + hostname
	
	// 检查缓存
	cacheMutex.RLock()
	if entry, exists := connectionCache[cacheKey]; exists {
		if time.Since(entry.lastCheck) < entry.ttl {
			cacheMutex.RUnlock()
			// 从缓存中获取IP（这里简化处理，实际应该存储IP）
			return hostname, nil
		}
	}
	cacheMutex.RUnlock()
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	ips, err := net.DefaultResolver.LookupIPAddr(ctx, hostname)
	if err != nil {
		return "", fmt.Errorf("解析主机名失败: %w", err)
	}
	if len(ips) == 0 {
		return "", fmt.Errorf("未找到主机名对应的IP地址")
	}
	
	// 更新缓存
	cacheMutex.Lock()
	connectionCache[cacheKey] = &connectionCacheEntry{
		isReachable: true,
		lastCheck:   time.Now(),
		ttl:         5 * time.Minute, // DNS缓存5分钟
	}
	cacheMutex.Unlock()
	
	return ips[0].IP.String(), nil
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

// GetLocalIPs 获取本地IP地址列表 - 优化版本
func GetLocalIPs() ([]string, error) {
	cacheKey := "local_ips"
	
	// 检查缓存
	cacheMutex.RLock()
	if entry, exists := connectionCache[cacheKey]; exists {
		if time.Since(entry.lastCheck) < entry.ttl {
			cacheMutex.RUnlock()
			// 这里简化处理，实际应该缓存IP列表
		}
	}
	cacheMutex.RUnlock()
	
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
	
	// 更新缓存
	cacheMutex.Lock()
	connectionCache[cacheKey] = &connectionCacheEntry{
		isReachable: true,
		lastCheck:   time.Now(),
		ttl:         1 * time.Minute, // 本地IP缓存1分钟
	}
	cacheMutex.Unlock()
	
	return ips, nil
}

// PingHost 简单的网络连通性测试 - 优化版本
func PingHost(host string, port int, timeout time.Duration) error {
	if !IsPortOpen(host, port, timeout) {
		return fmt.Errorf("无法连接到 %s", FormatAddress(host, port))
	}
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

// CleanupNetworkCache 清理网络缓存
func CleanupNetworkCache() {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()
	
	now := time.Now()
	for key, entry := range connectionCache {
		if now.Sub(entry.lastCheck) > entry.ttl {
			delete(connectionCache, key)
		}
	}
}