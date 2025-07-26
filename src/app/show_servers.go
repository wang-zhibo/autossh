package app

import (
	"autossh/src/utils"
	"strconv"
	"strings"
	"sync"
	"fmt"
)

var (
	// 缓存显示内容以避免重复计算
	displayCache = make(map[string]string)
	displayCacheMutex   sync.RWMutex
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
		
		utils.Logln("请输入序号、别名或命令（如 add/edit/remove/exit），输入 q 或 exit 退出：")
		
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
	utils.Logln(utils.FormatSeparator(" 欢迎使用 Auto SSH ", "=", maxlen))
	
	// 简化的欢迎信息
	fmt.Println("城南爸爸")
	fmt.Println("  ___   _      ___   _      ___   _      ___   _      ___   _ ")
	fmt.Println(" [(_)] |=|    [(_)] |=|    [(_)] |=|    [(_)] |=|    [(_)] |=| ")
	fmt.Println("  '-`  |_|     '-`  |_|     '-`  |_|     '-`  |_|     '-`  |_| ")
	fmt.Println(" /mmm/  /     /mmm/  /     /mmm/  /     /mmm/  /     /mmm/  / ")
	fmt.Println("       |____________|____________|____________|____________| ")
	fmt.Println("                             |            |            | ")
	fmt.Println("                         ___  \\\\      ___  \\\\      ___  \\\\ ")
	fmt.Println("                        [(_)] |=|    [(_)] |=|    [(_)] |=| ")
	fmt.Println("                         '-`  |_|     '-`  |_|     '-`  |_| ")
	fmt.Println("                        /mmm/        /mmm/        /mmm/ ")
	fmt.Println()
	
	// 使用字符串构建器优化输出
	var output strings.Builder
	
	for i, server := range cfg.Servers {
		output.WriteString(server.FormatPrint(strconv.Itoa(i+1), cfg.ShowDetail))
		output.WriteString("\n")
	}

	for _, group := range cfg.Groups {
		if len(group.Servers) == 0 {
			continue
		}

		var collapseNotice string
		if group.Collapse {
			collapseNotice = "[" + group.Prefix + " ↓]"
		} else {
			collapseNotice = "[" + group.Prefix + " ↑]"
		}

		output.WriteString(utils.FormatSeparator(" "+group.GroupName+" "+collapseNotice+" ", "_", maxlen))
		output.WriteString("\n")
		
		if !group.Collapse {
			for i, server := range group.Servers {
				output.WriteString(server.FormatPrint(group.Prefix+strconv.Itoa(i+1), cfg.ShowDetail))
				output.WriteString("\n")
			}
		}
	}

	// 一次性输出所有内容
	fmt.Print(output.String())
	
	utils.Logln(utils.FormatSeparator("", "=", maxlen))

	showMenu()

	utils.Logln(utils.FormatSeparator("", "=", maxlen))
	utils.Logln("请输入序号或操作: ")
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
	
	maxlength := 60
	for _, group := range cfg.Groups {
		length := utils.ZhLen(group.GroupName)
		if length > maxlength {
			maxlength = length + 10
		}
	}

	// 缓存结果
	displayCacheMutex.Lock()
	displayCache[cacheKey] = strconv.Itoa(maxlength)
	displayCacheMutex.Unlock()

	return maxlength
}
