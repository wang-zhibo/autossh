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
			// 清屏并显示连接信息
			utils.Clear()

			// 性能监控：服务器连接
			stopTimer := utils.StartTimer("server_connect")
			server := cfg.serverIndex[cmd].server

			// 收集所有要显示的文本
			serverName := server.Name
			serverAddr := fmt.Sprintf("%s@%s:%d", server.User, server.Ip, server.Port)

			nameText := "🚀 正在连接到服务器: " + serverName
			addrText := "📍 地址: " + serverAddr
			waitText := "⏳ 请稍候..."

			// 计算最大宽度
			maxWidth := 60
			texts := []string{nameText, addrText, waitText}
			for _, text := range texts {
				width := utils.ZhLen(text) + 8 // 加上边框和空格
				if width > maxWidth {
					maxWidth = width
				}
			}

			// 美化连接提示
			fmt.Println()
			fmt.Println("╔" + strings.Repeat("═", maxWidth-2) + "╗")

			// 服务器名称行
			namePadding := maxWidth - utils.ZhLen(nameText) - 4
			if namePadding < 0 {
				namePadding = 0
			}
			fmt.Printf("║ %s%s ║\n", nameText, strings.Repeat(" ", namePadding))

			// 地址行
			addrPadding := maxWidth - utils.ZhLen(addrText) - 4
			if addrPadding < 0 {
				addrPadding = 0
			}
			fmt.Printf("║ %s%s ║\n", addrText, strings.Repeat(" ", addrPadding))

			// 等待行
			waitPadding := maxWidth - utils.ZhLen(waitText) - 4
			if waitPadding < 0 {
				waitPadding = 0
			}
			fmt.Printf("║ %s%s ║\n", waitText, strings.Repeat(" ", waitPadding))

			fmt.Println("╚" + strings.Repeat("═", maxWidth-2) + "╝")
			fmt.Println()

			err := server.Connect()
			stopTimer()

			if err != nil {
				// 美化错误提示
				errorMsg := err.Error()

				failText := "❌ 连接失败"
				errText := "📝 错误信息: " + errorMsg
				tipText := "💡 请检查服务器配置和网络连接"

				// 计算错误提示的最大宽度
				errorWidth := 60
				errorTexts := []string{failText, errText, tipText}
				for _, text := range errorTexts {
					width := utils.ZhLen(text) + 8
					if width > errorWidth {
						errorWidth = width
					}
				}

				fmt.Println()
				fmt.Println("╔" + strings.Repeat("═", errorWidth-2) + "╗")

				// 错误标题
				failPadding := errorWidth - utils.ZhLen(failText) - 4
				if failPadding < 0 {
					failPadding = 0
				}
				fmt.Printf("║ %s%s ║\n", failText, strings.Repeat(" ", failPadding))

				// 错误信息
				errPadding := errorWidth - utils.ZhLen(errText) - 4
				if errPadding < 0 {
					errPadding = 0
				}
				fmt.Printf("║ %s%s ║\n", errText, strings.Repeat(" ", errPadding))

				// 提示信息
				tipPadding := errorWidth - utils.ZhLen(tipText) - 4
				if tipPadding < 0 {
					tipPadding = 0
				}
				fmt.Printf("║ %s%s ║\n", tipText, strings.Repeat(" ", tipPadding))

				fmt.Println("╚" + strings.Repeat("═", errorWidth-2) + "╝")
				fmt.Println()
				fmt.Print("按回车键返回主菜单...")
				fmt.Scanln()
				return false, true, false
			} else {
				// 美化断开提示 - 彻底修复对齐问题
				displayName := server.Name
				if server.Alias != "" {
					displayName = server.Alias
				}

				// 收集所有要显示的文本
				endText := "✅ SSH会话已结束"
				srvText := "🏠 服务器: " + displayName
				byeText := "👋 感谢使用 AutoSSH,再见!"

				// 计算最大宽度 - 使用更精确的计算方法
				exitWidth := 60
				exitTexts := []string{endText, srvText, byeText}
				for _, text := range exitTexts {
					width := utils.ZhLen(text) + 8
					if width > exitWidth {
						exitWidth = width
					}
				}

				// 确保宽度是偶数，避免对齐问题
				if exitWidth%2 != 0 {
					exitWidth++
				}

				fmt.Println()
				fmt.Println("╔" + strings.Repeat("═", exitWidth-2) + "╗")

				// 结束标题
				endPadding := exitWidth - utils.ZhLen(endText) - 4
				if endPadding < 0 {
					endPadding = 0
				}
				fmt.Printf("║ %s%s ║\n", endText, strings.Repeat(" ", endPadding))

				// 服务器名称
				srvPadding := exitWidth - utils.ZhLen(srvText) - 4
				if srvPadding < 0 {
					srvPadding = 0
				}
				fmt.Printf("║ %s%s ║\n", srvText, strings.Repeat(" ", srvPadding))

				// 感谢信息 - 特殊处理
				byeLen := utils.ZhLen(byeText)
				byePadding := exitWidth - byeLen - 4
				if byePadding < 0 {
					byePadding = 0
				}
				fmt.Printf("║ %s%s ║\n", byeText, strings.Repeat(" ", byePadding))

				fmt.Println("╚" + strings.Repeat("═", exitWidth-2) + "╝")
				fmt.Println()
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
