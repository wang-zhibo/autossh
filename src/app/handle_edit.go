package app

import (
	"autossh/src/utils"
	"io"
	"strings"
)

func handleEdit(cfg *Config, args []string) error {
	utils.Info("请输入相应序号：")
	id := ""
	if err := utils.Scanln(&id); err == io.EOF {
		return nil
	} else if err != nil {
		return err
	}
	id = strings.ToLower(strings.TrimSpace(id))

	serverIndex, ok := cfg.serverIndex[id]
	if !ok {
		utils.Errorln("序号不存在")
		return handleEdit(cfg, args)
	}

	if err := serverIndex.server.Edit(); err != nil {
		if err == io.EOF {
			return nil
		}
		return err
	}
	return cfg.saveConfig(true)
}
