package main

import (
	"autossh/src/utils"
	"fmt"
	"runtime"
	"strings"
	"time"
)

func main() {
	fmt.Println("=== AutoSSH 性能测试 ===")

	// 记录初始内存使用
	var m1 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)
	fmt.Printf("初始内存使用: %.2f MB\n", float64(m1.Alloc)/1024/1024)

	// 测试配置加载性能
	fmt.Println("\n--- 配置加载性能测试 ---")

	// 多次加载配置文件测试缓存效果
	for i := 0; i < 10; i++ {
		stopTimer := utils.StartTimer("config_load_test")
		// 注意：这里需要根据实际的函数名调整
		// cfg, err := app.LoadConfig(configFile)
		stopTimer()

		fmt.Printf("第%d次配置加载测试完成\n", i+1)

		// 短暂延迟
		time.Sleep(10 * time.Millisecond)
	}

	// 测试网络工具函数性能
	fmt.Println("\n--- 网络工具函数性能测试 ---")
	testHosts := []string{"localhost", "127.0.0.1"}

	for _, host := range testHosts {
		// 测试端口检查（添加超时参数）
		stopTimer := utils.StartTimer("port_check_test")
		isOpen := utils.IsPortOpen(host, 80, 5*time.Second)
		stopTimer()
		fmt.Printf("端口检查 %s:80 - %v\n", host, isOpen)

		// 测试主机名解析
		stopTimer = utils.StartTimer("hostname_resolve_test")
		ip, err := utils.ResolveHostname(host)
		stopTimer()
		if err == nil {
			fmt.Printf("主机名解析 %s -> %s\n", host, ip)
		}
	}

	// 测试字符串构建性能
	fmt.Println("\n--- 字符串构建性能测试 ---")
	testStringBuilding()

	// 记录最终内存使用
	var m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m2)
	fmt.Printf("\n最终内存使用: %.2f MB\n", float64(m2.Alloc)/1024/1024)
	fmt.Printf("内存增长: %.2f MB\n", float64(m2.Alloc-m1.Alloc)/1024/1024)

	// 打印性能统计
	fmt.Println("\n=== 性能统计报告 ===")
	utils.PrintPerformanceMetrics()

	fmt.Println("\n=== 内存使用报告 ===")
	utils.PrintMemoryUsage()
}

func testStringBuilding() {
	// 测试传统字符串拼接
	stopTimer := utils.StartTimer("string_concat_traditional")
	result := ""
	for i := 0; i < 1000; i++ {
		result += fmt.Sprintf("Line %d\n", i)
	}
	stopTimer()

	// 测试StringBuilder
	stopTimer = utils.StartTimer("string_concat_builder")
	var builder strings.Builder
	for i := 0; i < 1000; i++ {
		builder.WriteString(fmt.Sprintf("Line %d\n", i))
	}
	result2 := builder.String()
	stopTimer()

	fmt.Printf("传统拼接结果长度: %d\n", len(result))
	fmt.Printf("StringBuilder结果长度: %d\n", len(result2))
}
