package app

import (
	"autossh/src/utils"
	"fmt"
	"io"
	"os"
	"reflect"
	"strconv"
	"strings"
)

// 编辑
func (server *Server) Edit() error {
	keys := []string{"Name", "Ip", "Port", "User", "Password", "Method", "Key", "Alias"}
	for _, key := range keys {
		if err := server.scanVal(key); err != nil {
			return err
		}
	}

	return nil
}

func deftVal(val string) string {
	if val != "" {
		return "(default=" + val + ")"
	} else {
		return ""
	}
}

func (server *Server) scanVal(fieldName string) (err error) {
	elem := reflect.ValueOf(server).Elem()
	field := elem.FieldByName(fieldName)
	for {
		switch field.Type().String() {
		case "int":
			utils.Logln(fieldName + deftVal(strconv.FormatInt(field.Int(), 10)) + ":")
			var ipt string
			if _, err = fmt.Scanln(&ipt); err == nil {
				ipt = strings.TrimSpace(ipt)
				if ipt == "q" || ipt == "exit" {
					os.Exit(0)
				}
				if ipt == "" {
					return nil // 回车跳过
				}
				val, convErr := strconv.Atoi(ipt)
				if convErr != nil {
					utils.Error("请输入有效数字或回车跳过。")
					continue
				}
				field.SetInt(int64(val))
			}
		case "string":
			utils.Logln(fieldName + deftVal(field.String()) + ":")
			var ipt string
			if _, err = fmt.Scanln(&ipt); err == nil {
				ipt = strings.TrimSpace(ipt)
				if ipt == "q" || ipt == "exit" {
					os.Exit(0)
				}
				if ipt == "" {
					return nil // 回车跳过
				}
				field.SetString(ipt)
			}
		}

		if err != nil {
			if err == io.EOF {
				return err
			}

			// 允许输入空行
			if err.Error() == "unexpected newline" {
				return nil
			}
		}
		break
	}
	return nil
}
