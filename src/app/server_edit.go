package app

import (
	"autossh/src/utils"
	"os"
	"reflect"
	"strconv"
	"strings"

	"golang.org/x/term"
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
			if err = utils.Scanln(&ipt); err != nil {
				return err
			}
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
		case "string":
			defaultValue := field.String()
			if fieldName == "Password" && defaultValue != "" {
				defaultValue = "已设置"
			}
			var ipt string
			if fieldName == "Password" && term.IsTerminal(int(os.Stdin.Fd())) {
				utils.Log(fieldName + deftVal(defaultValue) + ": ")
				var password []byte
				password, err = term.ReadPassword(int(os.Stdin.Fd()))
				utils.Logln()
				ipt = string(password)
			} else {
				utils.Logln(fieldName + deftVal(defaultValue) + ":")
				err = utils.Scanln(&ipt)
			}
			if err != nil {
				return err
			}
			ipt = strings.TrimSpace(ipt)
			if ipt == "q" || ipt == "exit" {
				os.Exit(0)
			}
			if ipt == "" {
				return nil // 回车跳过
			}
			field.SetString(ipt)
		}
		break
	}
	return nil
}
