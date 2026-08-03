package utils

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

// LogLevel 定义日志级别。
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

var globalLogLevel atomic.Int32

func init() {
	globalLogLevel.Store(int32(INFO))
}

type loggerState struct {
	mu        sync.Mutex
	file      *os.File
	writer    *bufio.Writer
	logger    *log.Logger
	level     LogLevel
	stop      chan struct{}
	done      chan struct{}
	closeOnce sync.Once
	closeErr  error
}

// Logger 是一个显式创建、显式关闭的文件日志记录器。它不会在包加载时创建文件。
// 由 Category 返回的记录器共享同一文件和生命周期。
type Logger struct {
	state    *loggerState
	filename string
	category string
}

// NewLogger 创建新的日志记录器。调用方完成后应调用 Close。
func NewLogger(filename string, level LogLevel) *Logger {
	state := &loggerState{
		level: level,
		stop:  make(chan struct{}),
		done:  make(chan struct{}),
	}
	logger := &Logger{state: state, filename: filename}

	if err := logger.openFile(); err != nil {
		// 文件日志不可用时仍保留标准输出，避免日志本身影响主流程。
		state.logger = log.New(os.Stdout, "", 0)
		close(state.done)
		return logger
	}

	go state.flushRoutine()
	return logger
}

func (l *Logger) openFile() error {
	if l == nil || l.state == nil {
		return fmt.Errorf("日志记录器未初始化")
	}

	dir := filepath.Dir(l.filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建日志目录失败: %w", err)
	}

	file, err := os.OpenFile(l.filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return fmt.Errorf("打开日志文件失败: %w", err)
	}

	l.state.file = file
	l.state.writer = bufio.NewWriterSize(file, 4096)
	l.state.logger = log.New(io.MultiWriter(os.Stdout, l.state.writer), "", 0)
	return nil
}

func (state *loggerState) flushRoutine() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	defer close(state.done)

	for {
		select {
		case <-state.stop:
			return
		case <-ticker.C:
			state.flush()
		}
	}
}

func (state *loggerState) flush() {
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.writer != nil {
		_ = state.writer.Flush()
	}
}

func (l *Logger) log(level LogLevel, format string, args ...interface{}) {
	if l == nil || l.state == nil {
		return
	}

	state := l.state
	state.mu.Lock()
	defer state.mu.Unlock()
	if level < state.level || state.logger == nil || state.file == nil && state.writer != nil {
		return
	}

	message := fmt.Sprintf(format, args...)
	if l.category != "" {
		message = fmt.Sprintf("[%s] [%s] %s", levelNames[level], l.category, message)
	} else {
		message = fmt.Sprintf("[%s] %s", levelNames[level], message)
	}
	state.logger.Println(message)
	if level >= ERROR && state.writer != nil {
		_ = state.writer.Flush()
	}
}

// SetLevel 设置此记录器的最低输出级别。
func (l *Logger) SetLevel(level LogLevel) {
	if l == nil || l.state == nil {
		return
	}
	l.state.mu.Lock()
	l.state.level = level
	l.state.mu.Unlock()
}

// Category 返回共享底层日志资源、带指定分类的新记录器。
func (l *Logger) Category(category string) *Logger {
	if l == nil {
		return nil
	}
	return &Logger{state: l.state, filename: l.filename, category: category}
}

func (l *Logger) Debug(format string, args ...interface{}) { l.log(DEBUG, format, args...) }
func (l *Logger) Info(format string, args ...interface{})  { l.log(INFO, format, args...) }
func (l *Logger) Warn(format string, args ...interface{})  { l.log(WARN, format, args...) }
func (l *Logger) Error(format string, args ...interface{}) { l.log(ERROR, format, args...) }

// Close 停止刷新协程并刷新、关闭文件。重复调用安全。
func (l *Logger) Close() error {
	if l == nil || l.state == nil {
		return nil
	}

	state := l.state
	state.closeOnce.Do(func() {
		if state.writer != nil {
			close(state.stop)
			<-state.done
		}

		state.mu.Lock()
		defer state.mu.Unlock()
		if state.writer != nil {
			if err := state.writer.Flush(); err != nil {
				state.closeErr = err
			}
		}
		if state.file != nil {
			if err := state.file.Close(); err != nil && state.closeErr == nil {
				state.closeErr = err
			}
			state.file = nil
		}
	})
	return state.closeErr
}

func currentLogLevel() LogLevel {
	return LogLevel(globalLogLevel.Load())
}

func shouldLog(level LogLevel) bool {
	return level >= currentLogLevel()
}

// SetLevel 设置全局打印函数的最低输出级别。
func SetLevel(level int) {
	if level < int(DEBUG) || level > int(ERROR) {
		return
	}
	globalLogLevel.Store(int32(level))
}

func Info(a ...interface{}) {
	if shouldLog(INFO) {
		fmt.Println(a...)
	}
}

func Error(a ...interface{}) {
	if shouldLog(ERROR) {
		fmt.Println(a...)
	}
}

func Warn(a ...interface{}) {
	if shouldLog(WARN) {
		fmt.Println(a...)
	}
}

func Debug(a ...interface{}) {
	if shouldLog(DEBUG) {
		fmt.Println(a...)
	}
}
