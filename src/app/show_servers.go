package app

import (
	"autossh/src/utils"
	"fmt"
	"strconv"
	"strings"
	"sync"
)

var (
	// 缓存显示内容以避免重复计算
	displayCache      = make(map[string]string)
	displayCacheMutex sync.RWMutex
)

func showServers(configFile string) {
	// 性能监控：配置加载
	stopTimer := utils.StartTimer("config_load")
	cfg, err := loadConfig(configFile)
	stopTimer()

	if err != nil {
		utils.Errorln(err)
		return
	}

	for {
		// 性能监控：界面渲染
		stopTimer := utils.StartTimer("ui_render")
		_ = utils.Clear()
		show(cfg)
		stopTimer()

		// 性能监控：用户输入处理
		stopTimer = utils.StartTimer("input_processing")
		loop, clear, reload := scanInput(cfg)
		stopTimer()

		if !loop {
			break
		}

		if reload {
			// 清理缓存
			displayCacheMutex.Lock()
			displayCache = make(map[string]string)
			displayCacheMutex.Unlock()

			// 性能监控：配置重新加载
			stopTimer := utils.StartTimer("config_reload")
			cfg, err = loadConfig(configFile)
			stopTimer()
		}

		if clear {
			_ = utils.Clear()
		}
	}
}

// 显示服务 - 彻底修复版本
func show(cfg *Config) {
	// 收集所有要显示的行
	var lines []string
	
	// 添加服务器行
	for i, server := range cfg.Servers {
		serverText := server.FormatPrint(strconv.Itoa(i+1), cfg.ShowDetail)
		lines = append(lines, serverText)
	}

	// 添加分组行
	for _, group := range cfg.Groups {
		if len(group.Servers) == 0 {
			continue
		}

		var collapseNotice string
		if group.Collapse {
			collapseNotice = "📁 [" + group.Prefix + " 展开]"
		} else {
			collapseNotice = "📂 [" + group.Prefix + " 收起]"
		}

		groupTitle := "🏷️  " + group.GroupName + " " + collapseNotice
		lines = append(lines, groupTitle)

		if !group.Collapse {
			for i, server := range group.Servers {
				serverInfo := "  └─ " + server.FormatPrint(group.Prefix+strconv.Itoa(i+1), cfg.ShowDetail)
				lines = append(lines, serverInfo)
			}
		}
	}

	// 计算最大宽度
	maxWidth := 80
	for _, line := range lines {
		width := utils.ZhLen(line) + 4 // 加上边框和空格
		if width > maxWidth {
			maxWidth = width
		}
	}

	// 确保宽度至少能容纳标题和菜单
	titleWidth := utils.ZhLen("🚀 AutoSSH 管理工具") + 4
	menuWidth := utils.ZhLen("📝 add     - 添加新服务器    │  ✏️  edit   - 编辑服务器") + 4
	if titleWidth > maxWidth {
		maxWidth = titleWidth
	}
	if menuWidth > maxWidth {
		maxWidth = menuWidth
	}

	// 优化标题显示 - 使用动态宽度
	fmt.Println()
	titleBorder := "╔" + strings.Repeat("═", maxWidth-2) + "╗"
	fmt.Println(titleBorder)
	
	title1 := "🚀 AutoSSH 管理工具"
	title1Padding := (maxWidth - utils.ZhLen(title1) - 2) / 2
	title1Line := "║" + strings.Repeat(" ", title1Padding) + title1 + strings.Repeat(" ", maxWidth-utils.ZhLen(title1)-title1Padding-2) + "║"
	fmt.Println(title1Line)
	
	title2 := "SSH连接管理 - 简单高效"
	title2Padding := (maxWidth - utils.ZhLen(title2) - 2) / 2
	title2Line := "║" + strings.Repeat(" ", title2Padding) + title2 + strings.Repeat(" ", maxWidth-utils.ZhLen(title2)-title2Padding-2) + "║"
	fmt.Println(title2Line)
	
	titleBottomBorder := "╚" + strings.Repeat("═", maxWidth-2) + "╝"
	fmt.Println(titleBottomBorder)
	fmt.Println()

	// 简化的ASCII艺术 - 使用动态宽度
	artWidth := maxWidth
	artPadding := (artWidth - 45) / 2 // 45是ASCII艺术的大致宽度
	if artPadding < 0 {
		artPadding = 0
	}
	
	fmt.Println(strings.Repeat(" ", artPadding) + "┌─────┐  ┌─────┐  ┌─────┐  ┌─────┐  ┌─────┐")
	fmt.Println(strings.Repeat(" ", artPadding) + "│ SSH │  │ SSH │  │ SSH │  │ SSH │  │ SSH │")
	fmt.Println(strings.Repeat(" ", artPadding) + "└─────┘  └─────┘  └─────┘  └─────┘  └─────┘")
	fmt.Println(strings.Repeat(" ", artPadding) + "   │        │        │        │        │")
	fmt.Println(strings.Repeat(" ", artPadding) + "┌──┴────────┴────────┴────────┴────────┴──┐")
	fmt.Println(strings.Repeat(" ", artPadding) + "│            🌐 网络连接管理              │")
	fmt.Println(strings.Repeat(" ", artPadding) + "└─────────────────────────────────────────┘")
	fmt.Println()

	// 服务器列表标题
	if len(lines) > 0 {
		fmt.Println("📋 可用服务器列表:")
		fmt.Println("┌" + strings.Repeat("─", maxWidth-2) + "┐")
	}

	// 输出所有行
	serverCount := len(cfg.Servers)
	for i, line := range lines {
		fmt.Print("│ ")
		fmt.Print(line)
		
		// 计算需要的空格数
		lineWidth := utils.ZhLen(line)
		padding := maxWidth - lineWidth - 4 // 减去"│ "和" │"
		if padding < 0 {
			padding = 0
		}
		
		fmt.Print(strings.Repeat(" ", padding))
		fmt.Println(" │")
		
		// 在服务器和分组之间添加分隔线
		if i == serverCount-1 && serverCount > 0 && len(cfg.Groups) > 0 {
			fmt.Println("├" + strings.Repeat("─", maxWidth-2) + "┤")
		}
		
		// 在分组之间添加分隔线
		if i >= serverCount {
			groupIndex := 0
			linesSoFar := serverCount
			for _, group := range cfg.Groups {
				if len(group.Servers) == 0 {
					continue
				}
				
				groupLines := 1 // 分组标题
				if !group.Collapse {
					groupLines += len(group.Servers)
				}
				
				if i == linesSoFar + groupLines - 1 && groupIndex < len(cfg.Groups)-1 {
					// 检查后面还有分组
					hasMoreGroups := false
					for j := groupIndex + 1; j < len(cfg.Groups); j++ {
						if len(cfg.Groups[j].Servers) > 0 {
							hasMoreGroups = true
							break
						}
					}
					if hasMoreGroups {
						fmt.Println("├" + strings.Repeat("─", maxWidth-2) + "┤")
					}
					break
				}
				
				linesSoFar += groupLines
				groupIndex++
			}
		}
	}

	if len(lines) > 0 {
		fmt.Println("└" + strings.Repeat("─", maxWidth-2) + "┘")
	}

	fmt.Println()

	// 显示操作菜单 - 使用动态宽度
	fmt.Println("🛠️  可用操作:")
	showMenuWithWidth(maxWidth)

	fmt.Println()
	fmt.Println("💡 提示: 输入服务器编号直接连接，输入 'q' 或 'exit' 退出程序")
	fmt.Print("👉 请输入您的选择: ")
}

// 计算分隔符长度 - 简化版本
func separatorLength(cfg Config) int {
	maxlength := 80 // 基础宽度

	// 检查服务器名称长度
	for i, server := range cfg.Servers {
		serverText := server.FormatPrint(strconv.Itoa(i+1), cfg.ShowDetail)
		width := utils.ZhLen(serverText) + 4 // 加上边框和空格的宽度
		if width > maxlength {
			maxlength = width
		}
	}

	// 检查分组标题长度
	for _, group := range cfg.Groups {
		// 检查展开状态的标题
		groupTitleExpanded := "🏷️  " + group.GroupName + " 📁 [" + group.Prefix + " 展开]"
		width := utils.ZhLen(groupTitleExpanded) + 4
		if width > maxlength {
			maxlength = width
		}

		// 检查收起状态的标题
		groupTitleCollapsed := "🏷️  " + group.GroupName + " 📂 [" + group.Prefix + " 收起]"
		width = utils.ZhLen(groupTitleCollapsed) + 4
		if width > maxlength {
			maxlength = width
		}

		// 检查分组内服务器长度
		for i, server := range group.Servers {
			serverInfo := "  └─ " + server.FormatPrint(group.Prefix+strconv.Itoa(i+1), cfg.ShowDetail)
			width := utils.ZhLen(serverInfo) + 4
			if width > maxlength {
				maxlength = width
			}
		}
	}

	return maxlength
}
