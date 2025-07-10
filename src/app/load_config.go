package app

import (
	"autossh/src/utils"
	"encoding/json"
	"fmt"
	"io/ioutil"
)

// ConfigValidationError 配置验证错误
type ConfigValidationError struct {
	Field string
	Msg   string
}

func (e ConfigValidationError) Error() string {
	return fmt.Sprintf("配置验证失败 [%s]: %s", e.Field, e.Msg)
}

// loadConfig 加载配置文件
func loadConfig(configFile string) (*Config, error) {
	// 解析配置文件路径
	configFile, err := utils.ParsePath(configFile)
	if err != nil {
		return nil, fmt.Errorf("解析配置文件路径失败: %w", err)
	}

	// 检查文件是否存在
	if exists, _ := utils.FileIsExists(configFile); !exists {
		return nil, fmt.Errorf("配置文件不存在: %s", configFile)
	}

	// 读取配置文件
	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	// 解析JSON
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件JSON失败: %w", err)
	}

	// 设置文件路径
	cfg.file = configFile

	// 验证配置
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	// 创建服务器索引
	cfg.createServerIndex()

	utils.Logln("成功加载配置文件: %s", configFile)
	return &cfg, nil
}

// validate 验证配置
func (cfg *Config) validate() error {
	// 验证服务器配置
	for i, server := range cfg.Servers {
		if err := server.validate(); err != nil {
			return ConfigValidationError{
				Field: fmt.Sprintf("servers[%d]", i),
				Msg:   err.Error(),
			}
		}
	}

	// 验证组配置
	for i, group := range cfg.Groups {
		if err := group.validate(); err != nil {
			return ConfigValidationError{
				Field: fmt.Sprintf("groups[%d]", i),
				Msg:   err.Error(),
			}
		}
	}

	return nil
}

// validate 验证服务器配置
func (s *Server) validate() error {
	if s.Name == "" {
		return fmt.Errorf("服务器名称不能为空")
	}
	if s.Ip == "" {
		return fmt.Errorf("服务器IP不能为空")
	}
	if s.User == "" {
		return fmt.Errorf("用户名不能为空")
	}
	if s.Port <= 0 || s.Port > 65535 {
		return fmt.Errorf("端口号必须在1-65535之间")
	}
	if s.Method != "password" && s.Method != "key" {
		return fmt.Errorf("认证方法必须是 'password' 或 'key'")
	}
	if s.Method == "key" && s.Key == "" {
		return fmt.Errorf("使用密钥认证时，密钥路径不能为空")
	}
	return nil
}

// validate 验证组配置
func (g *Group) validate() error {
	if g.GroupName == "" {
		return fmt.Errorf("组名称不能为空")
	}
	if g.Prefix == "" {
		return fmt.Errorf("组前缀不能为空")
	}
	
	for i, server := range g.Servers {
		if err := server.validate(); err != nil {
			return fmt.Errorf("组内服务器[%d]配置错误: %w", i, err)
		}
	}
	
	if g.Proxy != nil {
		if err := g.Proxy.validate(); err != nil {
			return fmt.Errorf("代理配置错误: %w", err)
		}
	}
	
	return nil
}

// validate 验证代理配置
func (p *Proxy) validate() error {
	if p.Type == "" {
		return fmt.Errorf("代理类型不能为空")
	}
	if p.Server == "" {
		return fmt.Errorf("代理服务器地址不能为空")
	}
	if p.Port <= 0 || p.Port > 65535 {
		return fmt.Errorf("代理端口必须在1-65535之间")
	}
	return nil
}
