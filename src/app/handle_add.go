package app

import (
	"autossh/src/utils"
	"fmt"
	"io"
	"strings"
)

func handleAdd(cfg *Config, _ []string) error {
	groups := make(map[string]*Group)
	for i := range cfg.Groups {
		group := cfg.Groups[i]
		groups[group.Prefix] = group
		utils.Logln("["+group.Prefix+"]"+group.GroupName, "\t")
	}
	utils.Logln("[其他值]默认组")
	utils.Logln("请输入要插入的组（输入 q 或 exit 取消）：")
	g := ""
	if _, err := fmt.Scanln(&g); err == io.EOF {
		return nil
	}
	g = strings.TrimSpace(g)
	if g == "q" || g == "exit" {
		utils.Logln("已取消添加服务器。按回车返回主菜单。")
		fmt.Scanln()
		return nil
	}

	server := Server{}
	server.Format()
	if err := server.Edit(); err != nil {
		if err == io.EOF {
			return nil
		}
		return err
	}

	group, ok := groups[g]
	if ok {
		group.Servers = append(group.Servers, server)
		server.groupName = group.GroupName
	} else {
		cfg.Servers = append(cfg.Servers, &server)
	}

	err := cfg.saveConfig(true)
	if err != nil {
		utils.Error("保存配置失败: ", err)
	} else {
		utils.Logln("服务器添加成功！按回车返回主菜单。")
		fmt.Scanln()
	}
	return nil
}
