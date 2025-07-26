package utils

import (
	"fmt"
	"runtime"
	"sync"
	"time"
)

// PerformanceMonitor 性能监控器
type PerformanceMonitor struct {
	mu      sync.RWMutex
	metrics map[string]*Metric
}

// Metric 性能指标
type Metric struct {
	Name      string
	Count     int64
	TotalTime time.Duration
	MinTime   time.Duration
	MaxTime   time.Duration
	LastTime  time.Time
}

var globalMonitor = NewPerformanceMonitor()

// NewPerformanceMonitor 创建新的性能监控器
func NewPerformanceMonitor() *PerformanceMonitor {
	return &PerformanceMonitor{
		metrics: make(map[string]*Metric),
	}
}

// StartTimer 开始计时
func StartTimer(name string) func() {
	start := time.Now()
	return func() {
		duration := time.Since(start)
		globalMonitor.Record(name, duration)
	}
}

// Record 记录性能指标
func (pm *PerformanceMonitor) Record(name string, duration time.Duration) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	metric, exists := pm.metrics[name]
	if !exists {
		metric = &Metric{
			Name:    name,
			MinTime: duration,
			MaxTime: duration,
		}
		pm.metrics[name] = metric
	}
	
	metric.Count++
	metric.TotalTime += duration
	metric.LastTime = time.Now()
	
	if duration < metric.MinTime {
		metric.MinTime = duration
	}
	if duration > metric.MaxTime {
		metric.MaxTime = duration
	}
}

// GetMetrics 获取所有性能指标
func (pm *PerformanceMonitor) GetMetrics() map[string]*Metric {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	
	result := make(map[string]*Metric)
	for k, v := range pm.metrics {
		result[k] = &Metric{
			Name:      v.Name,
			Count:     v.Count,
			TotalTime: v.TotalTime,
			MinTime:   v.MinTime,
			MaxTime:   v.MaxTime,
			LastTime:  v.LastTime,
		}
	}
	return result
}

// PrintMetrics 打印性能指标
func (pm *PerformanceMonitor) PrintMetrics() {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	
	fmt.Println("=== 性能监控报告 ===")
	for name, metric := range pm.metrics {
		avgTime := metric.TotalTime / time.Duration(metric.Count)
		fmt.Printf("操作: %s\n", name)
		fmt.Printf("  调用次数: %d\n", metric.Count)
		fmt.Printf("  总耗时: %v\n", metric.TotalTime)
		fmt.Printf("  平均耗时: %v\n", avgTime)
		fmt.Printf("  最小耗时: %v\n", metric.MinTime)
		fmt.Printf("  最大耗时: %v\n", metric.MaxTime)
		fmt.Printf("  最后调用: %v\n", metric.LastTime.Format("2006-01-02 15:04:05"))
		fmt.Println()
	}
}

// GetMemoryUsage 获取内存使用情况
func GetMemoryUsage() runtime.MemStats {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m
}

// PrintMemoryUsage 打印内存使用情况
func PrintMemoryUsage() {
	m := GetMemoryUsage()
	fmt.Println("=== 内存使用情况 ===")
	fmt.Printf("已分配内存: %d KB\n", bToKb(m.Alloc))
	fmt.Printf("总分配内存: %d KB\n", bToKb(m.TotalAlloc))
	fmt.Printf("系统内存: %d KB\n", bToKb(m.Sys))
	fmt.Printf("GC次数: %d\n", m.NumGC)
	fmt.Printf("Goroutine数量: %d\n", runtime.NumGoroutine())
}

func bToKb(b uint64) uint64 {
	return b / 1024
}

// 全局函数
func RecordPerformance(name string, duration time.Duration) {
	globalMonitor.Record(name, duration)
}

func PrintPerformanceMetrics() {
	globalMonitor.PrintMetrics()
}

func GetPerformanceMetrics() map[string]*Metric {
	return globalMonitor.GetMetrics()
}