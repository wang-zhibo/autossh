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
		// 性能监控：输入检查
		stopTimer := utils.StartTimer("input_check")
		cmd, inputCmd, extInfo := checkInput(cfg)
		stopTimer()
		
		switch inputCmd {
		case InputCmdOpt:
			// 性能监控：操作处理
			stopTimer := utils.StartTimer("operation_process")
			operation := operations[cmd]
			if operation.Process != nil {
				if err := operation.Process(cfg, extInfo.([]string)); err != nil {
					utils.Error("操作失败: ", err)
					stopTimer()
					continue
				}
				if !operation.End {
					stopTimer()
					return true, true, true
				}
			}
			stopTimer()
			return
		case InputCmdServer:
			// 性能监控：服务器连接
			stopTimer := utils.StartTimer("server_connect")
			server := cfg.serverIndex[cmd].server
			utils.Logln("你选择了", server.Name)
			err := server.Connect()
			stopTimer()
			
			if err != nil {
				utils.Error("连接失败: ", err)
				utils.Logln("按回车返回主菜单。")
				fmt.Scanln()
				return false, true, false
			} else {
				utils.Logln("连接已断开，程序退出。")
				os.Exit(0)
			}
		case InputCmdGroupPrefix:
			// 性能监控：分组操作
			stopTimer := utils.StartTimer("group_toggle")
			group := cfg.Groups[extInfo.(int)]
			group.Collapse = !group.Collapse
			err := cfg.saveConfig(false)
			stopTimer()
			
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
			// 性能监控：用户输入扫描
			stopTimer := utils.StartTimer("user_input_scan")
			utils.Scanln(&ipt)
			stopTimer()
		} else {
			ipt = defaultServer
			defaultServer = ""
			skipOpt = true
		}

		// 性能监控：输入解析
		stopTimer := utils.StartTimer("input_parse")
		ipt = strings.TrimSpace(ipt)
		ipt = strings.ToLower(ipt)
		if ipt == "" {
			utils.Error("输入不能为空，请重新输入。")
			stopTimer()
			continue
		}
		if ipt == "q" || ipt == "exit" {
			utils.Logln("已退出程序。欢迎下次使用！")
			stopTimer()
			os.Exit(0)
		}

		ipts := strings.Fields(ipt)
		cmd = ipts[0]
		stopTimer()

		if !skipOpt {
			// 性能监控：操作查找
			stopTimer := utils.StartTimer("operation_lookup")
			if _, exists := operations[cmd]; exists {
				inputCmd = InputCmdOpt
				extInfo = ipts[1:]
				stopTimer()
				break
			}
			stopTimer()
		}

		// 性能监控：服务器查找
		stopTimer = utils.StartTimer("server_lookup")
		if _, ok := cfg.serverIndex[cmd]; ok {
			inputCmd = InputCmdServer
			stopTimer()
			break
		}
		stopTimer()

		// 性能监控：分组查找
		stopTimer = utils.StartTimer("group_lookup")
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
			stopTimer()
			break
		}
		stopTimer()

		utils.Error("输入有误，请重新输入。例如：1 或 add 或 l")
	}
	return cmd, inputCmd, extInfo
}
