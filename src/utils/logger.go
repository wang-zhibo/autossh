package utils

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// LogLevel 定义日志级别
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
)

var levelNames = map[LogLevel]string{
	DEBUG: "DEBUG",
	INFO:  "INFO",
	WARN:  "WARN",
	ERROR: "ERROR",
}

// Logger 结构体
type Logger struct {
	mu       sync.Mutex
	file     *os.File
	logger   *log.Logger
	level    LogLevel
	filename string
	category string
}

var defaultLogger *Logger

func init() {
	logFile, _ := ParsePath("./app.log")
	defaultLogger = NewLogger(logFile, INFO)
}

// NewLogger 创建新的日志记录器
func NewLogger(filename string, level LogLevel) *Logger {
	logger := &Logger{
		filename: filename,
		level:    level,
	}
	
	if err := logger.openFile(); err != nil {
		// 如果无法打开文件，使用标准输出
		logger.logger = log.New(os.Stdout, "", log.LstdFlags)
	}
	
	return logger
}

// openFile 打开日志文件
func (l *Logger) openFile() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	// 确保目录存在
	dir := filepath.Dir(l.filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建日志目录失败: %w", err)
	}
	
	file, err := os.OpenFile(l.filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return fmt.Errorf("打开日志文件失败: %w", err)
	}
	
	l.file = file
	l.logger = log.New(io.MultiWriter(os.Stdout, file), "", log.LstdFlags)
	return nil
}

// log 内部日志方法
func (l *Logger) log(level LogLevel, category string, format string, args ...interface{}) {
	if level < l.level {
		return
	}
	
	l.mu.Lock()
	defer l.mu.Unlock()
	
	levelName := levelNames[level]
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	
	var message string
	if category != "" {
		message = fmt.Sprintf("[%s] [%s] [%s] %s", timestamp, levelName, category, fmt.Sprintf(format, args...))
	} else {
		message = fmt.Sprintf("[%s] [%s] %s", timestamp, levelName, fmt.Sprintf(format, args...))
	}
	
	if l.logger != nil {
		l.logger.Println(message)
	}
}

// SetLevel 设置日志级别
func (l *Logger) SetLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// Category 设置日志分类
func (l *Logger) Category(category string) *Logger {
	return &Logger{
		file:     l.file,
		logger:   l.logger,
		level:    l.level,
		filename: l.filename,
		category: category,
	}
}

// Debug 调试日志
func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(DEBUG, l.category, format, args...)
}

// Info 信息日志
func (l *Logger) Info(format string, args ...interface{}) {
	l.log(INFO, l.category, format, args...)
}

// Warn 警告日志
func (l *Logger) Warn(format string, args ...interface{}) {
	l.log(WARN, l.category, format, args...)
}

// Error 错误日志
func (l *Logger) Error(format string, args ...interface{}) {
	l.log(ERROR, l.category, format, args...)
}

// Close 关闭日志文件
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// 全局函数
func Info(a ...interface{}) {
	fmt.Println(a...)
}

func Error(a ...interface{}) {
	fmt.Println(a...)
}

func Warn(a ...interface{}) {
	fmt.Println(a...)
}

func Debug(a ...interface{}) {
	fmt.Println(a...)
}

func SetLevel(level int) {}
