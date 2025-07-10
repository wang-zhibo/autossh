package app

import (
	"autossh/src/utils"
	"fmt"
	"io"
	"strings"
)

func handleRemove(cfg *Config, args []string) error {
	utils.Logln("请输入相应序号（输入 q 或 exit 取消）：")

	id := ""
	_, err := fmt.Scanln(&id)
	if err == io.EOF {
		return nil
	}
	id = strings.TrimSpace(id)
	if id == "q" || id == "exit" {
		utils.Logln("已取消删除服务器。按回车返回主菜单。")
		fmt.Scanln()
		return nil
	}

	serverIndex, ok := cfg.serverIndex[id]
	if !ok {
		utils.Errorln("序号不存在")
		return handleRemove(cfg, args)
	}

	if serverIndex.indexType == IndexTypeServer {
		servers := cfg.Servers
		cfg.Servers = append(servers[:serverIndex.serverIndex], servers[serverIndex.serverIndex+1:]...)
	} else {
		servers := cfg.Groups[serverIndex.groupIndex].Servers
		servers = append(servers[:serverIndex.serverIndex], servers[serverIndex.serverIndex+1:]...)
		cfg.Groups[serverIndex.groupIndex].Servers = servers
	}

	err = cfg.saveConfig(true)
	if err != nil {
		utils.Error("保存配置失败: ", err)
	} else {
		utils.Logln("服务器删除成功！按回车返回主菜单。")
		fmt.Scanln()
	}
	return nil
}
