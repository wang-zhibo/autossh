package app

import (
	"autossh/src/utils"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

var (
	Version string
	Build   string

	c                        string
	v                        bool
	h                        bool
	upgrade                  bool
	cp                       bool
	debug                    bool
	perf                     bool // 性能监控标志
	insecureSkipHostKeyCheck bool
)

func defaultConfigFilePath() string {
	exe, err := os.Executable()
	if err != nil {
		return "config.json"
	}
	return filepath.Join(filepath.Dir(exe), "config.json")
}

func Run() {
	c = defaultConfigFilePath()

	fs := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.Usage = func() { usage(fs.Output()) }

	fs.StringVar(&c, "c", c, "指定配置文件路径")
	fs.StringVar(&c, "config", c, "指定配置文件路径")

	fs.BoolVar(&v, "v", false, "显示版本信息")
	fs.BoolVar(&v, "version", false, "显示版本信息")

	fs.BoolVar(&h, "h", false, "显示帮助信息")
	fs.BoolVar(&h, "help", false, "显示帮助信息")

	fs.BoolVar(&debug, "debug", false, "启用调试模式")
	fs.BoolVar(&perf, "perf", false, "启用性能监控")
	fs.BoolVar(&insecureSkipHostKeyCheck, "insecure", false, "跳过 SSH HostKey 校验（不安全）")
	fs.BoolVar(&insecureSkipHostKeyCheck, "skip-host-key-check", false, "跳过 SSH HostKey 校验（不安全）")

	if err := fs.Parse(os.Args[1:]); err != nil {
		fs.Usage()
		os.Exit(2)
	}

	upgrade = false
	cp = false
	defaultServer = ""
	var cpArgs []string

	if len(fs.Args()) > 0 {
		arg := fs.Args()[0]
		switch strings.ToLower(arg) {
		case "upgrade":
			upgrade = true
		case "cp":
			cp = true
			cpArgs = fs.Args()[1:]
		default:
			defaultServer = arg
		}
	}

	utils.EnablePerformanceMonitoring(perf)

	defer func() {
		if r := recover(); r != nil {
			utils.Errorf("程序发生严重错误: %v", r)
			os.Exit(1)
		}

		// 如果启用了性能监控，在程序结束时打印报告
		if perf {
			utils.PrintPerformanceMetrics()
			utils.PrintMemoryUsage()
		}
	}()

	// 性能监控：应用启动
	var stopTimer func()
	if perf {
		stopTimer = utils.StartTimer("app_startup")
	}

	utils.Info("AutoSSH 启动中...")

	if v {
		showVersion()
	} else if h {
		showHelp()
	} else if upgrade {
		showUpgrade()
	} else if cp {
		showCp(c, cpArgs)
	} else {
		if perf && stopTimer != nil {
			stopTimer()
		}
		showServers(c)
	}
}

// usage 显示使用说明
func usage(w io.Writer) {
	fmt.Fprintf(w, `AutoSSH - 一个简单的SSH连接管理工具

用法:
  autossh [选项] [服务器编号/别名]

选项:
  -c, --config string    指定配置文件路径 (默认: ./config.json)
  -v, --version         显示版本信息
  -h, --help            显示帮助信息
  -debug                启用调试模式
  -perf                 启用性能监控
  --insecure            跳过 SSH HostKey 校验（不安全）

命令:
  upgrade               检查并下载最新版本
  cp                    复制配置文件

示例:
  autossh              显示服务器列表
  autossh 1            连接到编号为1的服务器
  autossh server1      连接到别名为server1的服务器
  autossh -c /path/to/config.json 使用指定配置文件
  autossh -debug       启用调试模式
  autossh -perf        启用性能监控

配置文件格式请参考 config.example.json
`)
}
