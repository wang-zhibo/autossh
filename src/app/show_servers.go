package app

import (
	"autossh/src/utils"
	"strconv"
    "fmt"
)

func showServers(configFile string) {
	cfg, err := loadConfig(configFile)
	if err != nil {
		utils.Errorln(err)
		return
	}

	for {
		_ = utils.Clear()
		show(cfg)
		utils.Logln("请输入序号、别名或命令（如 add/edit/remove/exit），输入 q 或 exit 退出：")
		loop, clear, reload := scanInput(cfg)
		if !loop {
			break
		}

		if reload {
			cfg, err = loadConfig(configFile)
		}

		if clear {
			_ = utils.Clear()
		}
	}
}

// 显示服务
func show(cfg *Config) {
	maxlen := separatorLength(*cfg)
	utils.Logln(utils.FormatSeparator(" 欢迎使用 Auto SSH ", "=", maxlen))
    asciiArt := "城南爸爸 \n"+
        "  ___   _      ___   _      ___   _      ___   _      ___   _ \n" +
        " [(_)] |=|    [(_)] |=|    [(_)] |=|    [(_)] |=|    [(_)] |=| \n" +
        "  '-`  |_|     '-`  |_|     '-`  |_|     '-`  |_|     '-`  |_| \n" +
        " /mmm/  /     /mmm/  /     /mmm/  /     /mmm/  /     /mmm/  / \n" +
        "       |____________|____________|____________|____________| \n" +
        "                             |            |            | \n" +
        "                         ___  \\_      ___  \\_      ___  \\_ \n" +
        "                        [(_)] |=|    [(_)] |=|    [(_)] |=| \n" +
        "                         '-`  |_|     '-`  |_|     '-`  |_| \n" +
        "                        /mmm/        /mmm/        /mmm/ \n" + 
        "\n"
    fmt.Println(asciiArt)
	for i, server := range cfg.Servers {
		utils.Logln(server.FormatPrint(strconv.Itoa(i+1), cfg.ShowDetail))
	}

	for _, group := range cfg.Groups {
		if len(group.Servers) == 0 {
			continue
		}

		var collapseNotice = ""
		if group.Collapse {
			collapseNotice = "[" + group.Prefix + " ↓]"
		} else {
			collapseNotice = "[" + group.Prefix + " ↑]"
		}

		utils.Logln(utils.FormatSeparator(" "+group.GroupName+" "+collapseNotice+" ", "_", maxlen))
		if !group.Collapse {
			for i, server := range group.Servers {
				utils.Logln(server.FormatPrint(group.Prefix+strconv.Itoa(i+1), cfg.ShowDetail))
			}
		}
	}

	utils.Logln(utils.FormatSeparator("", "=", maxlen))

	showMenu()

	utils.Logln(utils.FormatSeparator("", "=", maxlen))
	utils.Logln("请输入序号或操作: ")
}

// 计算分隔符长度
func separatorLength(cfg Config) int {
	maxlength := 60
	for _, group := range cfg.Groups {
		length := utils.ZhLen(group.GroupName)
		if length > maxlength {
			maxlength = length + 10
		}
	}

	return maxlength
}
