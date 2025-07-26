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

// 显示服务 - 优化版本
func show(cfg *Config) {
	maxlen := separatorLength(*cfg)

	// 优化标题显示
	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║                        🚀 AutoSSH 管理工具                   ║")
	fmt.Println("║                      SSH连接管理 - 简单高效                   ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// 简化的ASCII艺术
	fmt.Println("                    🏰 城南爸爸的SSH工具箱")
	fmt.Println("    ┌─────┐  ┌─────┐  ┌─────┐  ┌─────┐  ┌─────┐")
	fmt.Println("    │ SSH │  │ SSH │  │ SSH │  │ SSH │  │ SSH │")
	fmt.Println("    └─────┘  └─────┘  └─────┘  └─────┘  └─────┘")
	fmt.Println("       │        │        │        │        │")
	fmt.Println("    ┌──┴────────┴────────┴────────┴────────┴──┐")
	fmt.Println("    │            🌐 网络连接管理              │")
	fmt.Println("    └─────────────────────────────────────────┘")
	fmt.Println()

	// 服务器列表标题
	if len(cfg.Servers) > 0 {
		fmt.Println("📋 可用服务器列表:")
		fmt.Println("┌" + strings.Repeat("─", maxlen-2) + "┐")
	}

	// 使用字符串构建器优化输出
	var output strings.Builder

	for i, server := range cfg.Servers {
		serverText := server.FormatPrint(strconv.Itoa(i+1), cfg.ShowDetail)
		output.WriteString("│ ")
		output.WriteString(serverText)
		// 使用utils.ZhLen计算填充空格数量
		textWidth := utils.ZhLen(serverText)
		padding := maxlen - textWidth - 4 // 减去边框和空格的宽度
		if padding > 0 {
			output.WriteString(strings.Repeat(" ", padding))
		}
		output.WriteString(" │\n")
	}

	// 分组服务器
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

		if len(cfg.Servers) > 0 {
			output.WriteString("├" + strings.Repeat("─", maxlen-2) + "┤\n")
		}

		groupTitle := "🏷️  " + group.GroupName + " " + collapseNotice
		output.WriteString("│ ")
		output.WriteString(groupTitle)
		// 使用utils.ZhLen计算填充空格数量
		titleWidth := utils.ZhLen(groupTitle)
		padding := maxlen - titleWidth - 4
		if padding > 0 {
			output.WriteString(strings.Repeat(" ", padding))
		}
		output.WriteString(" │\n")

		if !group.Collapse {
			for i, server := range group.Servers {
				serverInfo := "  └─ " + server.FormatPrint(group.Prefix+strconv.Itoa(i+1), cfg.ShowDetail)
				output.WriteString("│ ")
				output.WriteString(serverInfo)
				// 使用utils.ZhLen计算填充空格数量
				infoWidth := utils.ZhLen(serverInfo)
				padding := maxlen - infoWidth - 4
				if padding > 0 {
					output.WriteString(strings.Repeat(" ", padding))
				}
				output.WriteString(" │\n")
			}
		}
	}

	if len(cfg.Servers) > 0 || len(cfg.Groups) > 0 {
		output.WriteString("└" + strings.Repeat("─", maxlen-2) + "┘\n")
	}

	// 一次性输出所有内容
	fmt.Print(output.String())
	fmt.Println()

	// 显示操作菜单
	fmt.Println("🛠️  可用操作:")
	showMenu()

	fmt.Println()
	fmt.Println("💡 提示: 输入服务器编号直接连接，输入 'q' 或 'exit' 退出程序")
	fmt.Print("👉 请输入您的选择: ")
}

// 计算分隔符长度 - 优化版本
func separatorLength(cfg Config) int {
	// 使用缓存避免重复计算
	cacheKey := "separator_length"
	displayCacheMutex.RLock()
	if cached, exists := displayCache[cacheKey]; exists {
		displayCacheMutex.RUnlock()
		if length, err := strconv.Atoi(cached); err == nil {
			return length
		}
	}
	displayCacheMutex.RUnlock()

	maxlength := 70 // 增加基础宽度以适应新的显示格式

	// 检查服务器名称长度
	for _, server := range cfg.Servers {
		serverText := server.FormatPrint("1", cfg.ShowDetail)
		width := utils.ZhLen(serverText)
		if width > maxlength {
			maxlength = width + 10
		}
	}

	// 检查分组标题长度
	for _, group := range cfg.Groups {
		groupTitle := "🏷️  " + group.GroupName + " 📁 [" + group.Prefix + " 展开]"
		width := utils.ZhLen(groupTitle)
		if width > maxlength {
			maxlength = width + 10
		}

		// 检查分组内服务器长度
		for _, server := range group.Servers {
			serverInfo := "  └─ " + server.FormatPrint(group.Prefix+"1", cfg.ShowDetail)
			width := utils.ZhLen(serverInfo)
			if width > maxlength {
				maxlength = width + 10
			}
		}
	}

	// 缓存结果
	displayCacheMutex.Lock()
	displayCache[cacheKey] = strconv.Itoa(maxlength)
	displayCacheMutex.Unlock()

	return maxlength
}
