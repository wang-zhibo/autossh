package app

import (
	"autossh/src/utils"
	"fmt"
	"os"
	"strings"
)

const (
	InputCmdOpt int = iota
	InputCmdServer
	InputCmdGroupPrefix
)

var defaultServer = ""

// 获取输入
func scanInput(cfg *Config) (loop bool, clear bool, reload bool) {
	for {
		cmd, inputCmd, extInfo := checkInput(cfg)
		switch inputCmd {
		case InputCmdOpt:
			operation := operations[cmd]
			if operation.Process != nil {
				if err := operation.Process(cfg, extInfo.([]string)); err != nil {
					utils.Error("操作失败: ", err)
					continue
				}
				if !operation.End {
					return true, true, true
				}
			}
			return
		case InputCmdServer:
			server := cfg.serverIndex[cmd].server
			utils.Logln("你选择了", server.Name)
			err := server.Connect()
			if err != nil {
				utils.Error("连接失败: ", err)
				utils.Logln("按回车返回主菜单。")
				fmt.Scanln()
			} else {
				utils.Logln("连接已断开，按回车返回主菜单。")
				fmt.Scanln()
			}
			return false, true, false
		case InputCmdGroupPrefix:
			group := cfg.Groups[extInfo.(int)]
			group.Collapse = !group.Collapse
			err := cfg.saveConfig(false)
			if err != nil {
				utils.Error("保存分组折叠状态失败: ", err)
				continue
			}
			return true, true, true
		}
	}
	loop = true
	return
}

// 检查输入
func checkInput(cfg *Config) (cmd string, inputCmd int, extInfo interface{}) {
	for {
		ipt := ""
		skipOpt := false
		// 此处不再输出输入提示，由主菜单统一输出
		if defaultServer == "" {
			utils.Scanln(&ipt)
		} else {
			ipt = defaultServer
			defaultServer = ""
			skipOpt = true
		}

		ipt = strings.TrimSpace(ipt)
		ipt = strings.ToLower(ipt)
		if ipt == "" {
			utils.Error("输入不能为空，请重新输入。")
			continue
		}
		if ipt == "q" || ipt == "exit" {
			utils.Logln("已退出程序。欢迎下次使用！")
			os.Exit(0)
		}

		ipts := strings.Fields(ipt)
		cmd = ipts[0]

		if !skipOpt {
			if _, exists := operations[cmd]; exists {
				inputCmd = InputCmdOpt
				extInfo = ipts[1:]
				break
			}
		}

		if _, ok := cfg.serverIndex[cmd]; ok {
			inputCmd = InputCmdServer
			break
		}

		groupIndex := -1
		for index, group := range cfg.Groups {
			if strings.ToLower(group.Prefix) == cmd {
				inputCmd = InputCmdGroupPrefix
				groupIndex = index
				extInfo = index
				break
			}
		}
		if groupIndex != -1 {
			break
		}

		utils.Error("输入有误，请重新输入。例如：1 或 add 或 l")
	}
	return cmd, inputCmd, extInfo
}
