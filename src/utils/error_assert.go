package utils

import (
	"errors"
	"strings"
	"syscall"
)

// ErrorAssert 错误断言 - 检查错误信息是否包含指定字符串
func ErrorAssert(err error, assert string) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), assert)
}

// IsConnectionRefused 检查是否为连接被拒绝错误
func IsConnectionRefused(err error) bool {
	return errors.Is(err, syscall.ECONNREFUSED) || 
		   ErrorAssert(err, "connection refused")
}

// IsTimeout 检查是否为超时错误
func IsTimeout(err error) bool {
	return ErrorAssert(err, "timeout") || 
		   ErrorAssert(err, "timed out") ||
		   ErrorAssert(err, "deadline exceeded")
}

// IsAuthenticationFailed 检查是否为认证失败错误
func IsAuthenticationFailed(err error) bool {
	return ErrorAssert(err, "unable to authenticate") ||
		   ErrorAssert(err, "authentication failed") ||
		   ErrorAssert(err, "invalid credentials")
}

// IsNetworkError 检查是否为网络相关错误
func IsNetworkError(err error) bool {
	return IsConnectionRefused(err) ||
		   IsTimeout(err) ||
		   ErrorAssert(err, "no route to host") ||
		   ErrorAssert(err, "network unreachable")
}

// IsSSHError 检查是否为SSH相关错误
func IsSSHError(err error) bool {
	return ErrorAssert(err, "ssh:") ||
		   ErrorAssert(err, "handshake failed") ||
		   ErrorAssert(err, "unable to authenticate")
}

// GetErrorType 获取错误类型
func GetErrorType(err error) string {
	if err == nil {
		return "none"
	}
	
	if IsConnectionRefused(err) {
		return "connection_refused"
	}
	if IsTimeout(err) {
		return "timeout"
	}
	if IsAuthenticationFailed(err) {
		return "authentication_failed"
	}
	if IsNetworkError(err) {
		return "network_error"
	}
	if IsSSHError(err) {
		return "ssh_error"
	}
	
	return "unknown"
}
