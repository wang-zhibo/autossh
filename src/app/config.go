package app

import (
	"autossh/src/utils"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

type Config struct {
	ShowDetail bool                   `json:"show_detail"`
	Servers    []*Server              `json:"servers"`
	Groups     []*Group               `json:"groups"`
	Options    map[string]interface{} `json:"options"`

	// 服务器map索引，可通过编号、别名快速定位到某一个服务器
	serverIndex map[string]ServerIndex
	file        string
	
	// 性能优化：添加缓存和锁
	mu          sync.RWMutex
	lastModTime time.Time
	isDirty     bool
}

type Group struct {
	GroupName string   `json:"group_name"`
	Prefix    string   `json:"prefix"`
	Servers   []Server `json:"servers"`
	Collapse  bool     `json:"collapse"`
	Proxy     *Proxy   `json:"proxy"`
}

type ProxyType string

const (
	ProxyTypeSocks5 ProxyType = "SOCKS5"
)

type Proxy struct {
	Type     ProxyType `json:"type"`
	Server   string    `json:"server"`
	Port     int       `json:"port"`
	User     string    `json:"user"`
	Password string    `json:"password"`
}

type LogMode string

const (
	LogModeCover  LogMode = "cover"
	LogModeAppend LogMode = "append"
)

type ServerLog struct {
	Enable   bool    `json:"enable"`
	Filename string  `json:"filename"`
	Mode     LogMode `json:"mode"`
}

const (
	IndexTypeServer IndexType = iota
	IndexTypeGroup
)

type IndexType int
type ServerIndex struct {
	indexType   IndexType
	groupIndex  int
	serverIndex int
	server      *Server
}

// 创建服务器索引
func (cfg *Config) createServerIndex() {
	cfg.serverIndex = make(map[string]ServerIndex)
	for i := range cfg.Servers {
		server := cfg.Servers[i]
		server.Format()
		index := strconv.Itoa(i + 1)

		if _, ok := cfg.serverIndex[index]; ok {
			continue
		}

		server.MergeOptions(cfg.Options, false)
		cfg.serverIndex[index] = ServerIndex{
			indexType:   IndexTypeServer,
			groupIndex:  -1,
			serverIndex: i,
			server:      server,
		}
		if server.Alias != "" {
			cfg.serverIndex[server.Alias] = cfg.serverIndex[index]
		}
	}

	for i := range cfg.Groups {
		group := cfg.Groups[i]
		for j := range group.Servers {
			server := &group.Servers[j]
			server.Format()
			server.groupName = group.GroupName
			server.group = group
			index := group.Prefix + strconv.Itoa(j+1)

			if _, ok := cfg.serverIndex[index]; ok {
				continue
			}

			server.MergeOptions(cfg.Options, false)
			cfg.serverIndex[index] = ServerIndex{
				indexType:   IndexTypeGroup,
				groupIndex:  i,
				serverIndex: j,
				server:      server,
			}
			if server.Alias != "" {
				cfg.serverIndex[server.Alias] = cfg.serverIndex[index]
			}
		}
	}
}

// 保存配置文件 - 优化版本
func (cfg *Config) saveConfig(backup bool) error {
	cfg.mu.Lock()
	defer cfg.mu.Unlock()
	
	// 如果配置没有变化，跳过保存
	if !cfg.isDirty {
		return nil
	}
	
	// 验证配置
	if err := cfg.validate(); err != nil {
		return fmt.Errorf("配置验证失败: %w", err)
	}

	// 使用更高效的JSON编码器
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetIndent("", "\t")
	encoder.SetEscapeHTML(false)
	
	if err := encoder.Encode(cfg); err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}

	// 创建备份
	if backup {
		if err := cfg.backup(); err != nil {
			return fmt.Errorf("创建备份失败: %w", err)
		}
	}

	// 原子性写入
	if err := cfg.atomicWrite(buf.Bytes()); err != nil {
		return err
	}

	cfg.isDirty = false
	cfg.lastModTime = time.Now()
	utils.Logf("配置文件已保存: %s", cfg.file)
	return nil
}

// 原子性写入文件
func (cfg *Config) atomicWrite(data []byte) error {
	tempFile := cfg.file + ".tmp"
	
	// 写入临时文件
	if err := ioutil.WriteFile(tempFile, data, 0644); err != nil {
		return fmt.Errorf("写入临时文件失败: %w", err)
	}

	// 原子性地重命名文件
	if err := os.Rename(tempFile, cfg.file); err != nil {
		os.Remove(tempFile) // 清理临时文件
		return fmt.Errorf("保存配置文件失败: %w", err)
	}

	return nil
}

// 标记配置为已修改
func (cfg *Config) markDirty() {
	cfg.mu.Lock()
	defer cfg.mu.Unlock()
	cfg.isDirty = true
}

// 检查配置文件是否需要重新加载
func (cfg *Config) needReload() bool {
	cfg.mu.RLock()
	defer cfg.mu.RUnlock()
	
	info, err := os.Stat(cfg.file)
	if err != nil {
		return false
	}
	
	return info.ModTime().After(cfg.lastModTime)
}

// 备份配置文件
func (cfg *Config) backup() error {
	// 检查源文件是否存在
	if _, err := os.Stat(cfg.file); os.IsNotExist(err) {
		utils.Warn("源配置文件不存在，跳过备份")
		return nil
	}

	// 打开源文件
	srcFile, err := os.Open(cfg.file)
	if err != nil {
		return fmt.Errorf("打开源文件失败: %w", err)
	}
	defer srcFile.Close()

	// 生成备份文件名
	path, _ := filepath.Abs(filepath.Dir(cfg.file))
	timestamp := time.Now().Format("20060102150405")
	backupFile := filepath.Join(path, fmt.Sprintf("config-%s.json", timestamp))

	// 创建备份文件
	destFile, err := os.Create(backupFile)
	if err != nil {
		return fmt.Errorf("创建备份文件失败: %w", err)
	}
	defer destFile.Close()

	// 复制文件内容
	if _, err := io.Copy(destFile, srcFile); err != nil {
		return fmt.Errorf("复制文件内容失败: %w", err)
	}

	utils.Logf("配置文件已备份: %s", backupFile)
	return nil
}

// 清理旧备份文件
func (cfg *Config) cleanupOldBackups(maxBackups int) error {
	path, _ := filepath.Abs(filepath.Dir(cfg.file))
	pattern := filepath.Join(path, "config-*.json")
	
	files, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("查找备份文件失败: %w", err)
	}

	// 按修改时间排序
	type fileInfo struct {
		path    string
		modTime time.Time
	}
	
	var fileInfos []fileInfo
	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil {
			continue
		}
		fileInfos = append(fileInfos, fileInfo{
			path:    file,
			modTime: info.ModTime(),
		})
	}

	// 按修改时间降序排序
	for i := 0; i < len(fileInfos)-1; i++ {
		for j := i + 1; j < len(fileInfos); j++ {
			if fileInfos[i].modTime.Before(fileInfos[j].modTime) {
				fileInfos[i], fileInfos[j] = fileInfos[j], fileInfos[i]
			}
		}
	}

	// 删除多余的备份文件
	if len(fileInfos) > maxBackups {
		for _, fileInfo := range fileInfos[maxBackups:] {
			if err := os.Remove(fileInfo.path); err != nil {
				utils.Errorf("删除旧备份文件失败: %s, %v", fileInfo.path, err)
			} else {
				utils.Logf("已删除旧备份文件: %s", fileInfo.path)
			}
		}
	}

	return nil
}
