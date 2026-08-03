package utils

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ParsePath 解析配置中的路径。相对路径以可执行文件所在目录为基准，~ 和 ~/...
// 使用当前用户主目录。它不执行 shell，避免依赖用户环境或把路径当作命令解释。
func ParsePath(value string) (string, error) {
	if value == "" {
		return "", errors.New("路径不能为空")
	}

	switch {
	case value == "~":
		return os.UserHomeDir()
	case strings.HasPrefix(value, "~/") || strings.HasPrefix(value, `~\`):
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("获取用户主目录失败: %w", err)
		}
		return filepath.Join(home, value[2:]), nil
	case strings.HasPrefix(value, "~"):
		return "", errors.New("仅支持当前用户主目录写法 ~ 或 ~/...")
	case filepath.IsAbs(value):
		return filepath.Clean(value), nil
	}

	executable, err := os.Executable()
	if err == nil {
		return filepath.Join(filepath.Dir(executable), value), nil
	}
	absolute, absErr := filepath.Abs(value)
	if absErr != nil {
		return "", fmt.Errorf("解析相对路径失败: %w", absErr)
	}
	return absolute, nil
}

// FileIsExists 判断文件是否存在。不存在不是异常，其他访问错误会返回给调用方。
func FileIsExists(file string) (bool, error) {
	file, err := ParsePath(file)
	if err != nil {
		return false, err
	}

	_, err = os.Stat(file)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
