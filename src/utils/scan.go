package utils

import (
	"bufio"
	"io"
	"os"
	"strings"
)

var stdinReader = bufio.NewReader(os.Stdin)

// Scanln 读取一整行输入并保留同一读取器，避免 bufio 预读后丢失下一行输入。
// 当标准输入结束时返回 io.EOF，调用方必须停止交互流程而不是继续重试。
func Scanln(a *string) error {
	data, err := stdinReader.ReadString('\n')
	if len(data) > 0 {
		*a = strings.TrimSuffix(strings.TrimSuffix(data, "\n"), "\r")
		return nil
	}

	if err == nil {
		return io.EOF
	}
	return err
}
