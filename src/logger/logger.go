package logger

import (
	"fmt"
	stdlog "log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/mattn/go-colorable"
)

// ---- Levels ----

type Level int

const (
	level_debug Level = iota
	level_info
	level_warn
	level_error
	level_fatal
)

var levelNames = [...]string{"DEBUG", "INFO", "WARN", "ERROR", "FATAL"}

// ANSI colors (only for terminal)
const (
	cReset   = "\x1b[0m"
	cBlue    = "\x1b[34m"
	cGreen   = "\x1b[32m"
	cYellow  = "\x1b[33m"
	cRed     = "\x1b[31m"
	cMagenta = "\x1b[35m"
)

const (
	logPath = "log"
)

type Options struct {
	Path   string
	Level  string
	MaxDay int
}

func parseLevel(s string) Level {
	switch strings.ToUpper(s) {
	case "DEBUG":
		return level_debug
	case "INFO":
		return level_info
	case "WARN", "WARNING":
		return level_warn
	case "ERROR":
		return level_error
	case "FATAL":
		return level_fatal
	default:
		return level_info
	}
}

var (
	fileSink   *rotatelogs.RotateLogs
	fileLogger *stdlog.Logger
	termLogger *stdlog.Logger
	minLevel   = level_info
)

func OpenDefault() {
	Open(Options{
		Path:   logPath,
		Level:  "INFO",
		MaxDay: 7,
	})
}

func Open(options Options) {
	if options.Path == "" {
		options.Path = logPath
	}
	if options.Level == "" {
		options.Level = "INFO"
	}
	if options.MaxDay <= 0 {
		options.MaxDay = 7
	}

	minLevel = parseLevel(options.Level)
	dir := options.Path
	if err := os.MkdirAll(dir, 0o755); err != nil {
		panic(fmt.Errorf("创建日志目录失败: %w", err))
	}

	// 文件名：年-月-日.log, 符号链接 current.log到当前log文件
	pattern := filepath.Join(dir, "%Y-%m-%d.log")
	rl, err := rotatelogs.New(
		pattern,
		rotatelogs.WithLinkName(filepath.Join(dir, "current.log")),
		rotatelogs.WithRotationTime(24*time.Hour),
		rotatelogs.WithMaxAge(time.Duration(options.MaxDay)*24*time.Hour),
		rotatelogs.WithRotationCount(0),
	)
	if err != nil {
		panic(fmt.Errorf("初始化日志失败: %w", err))
	}
	fileSink = rl

	// 文件logger，不带颜色
	fileLogger = stdlog.New(fileSink, "", stdlog.LstdFlags)

	// 终端logger，带颜色输出
	termOut := colorable.NewColorableStdout()
	termLogger = stdlog.New(termOut, "", stdlog.LstdFlags)
}

// Close flushes and closes the rotating writer.
func Close() {
	if fileSink != nil {
		_ = fileSink.Close()
		fileSink = nil
	}
}
func callerInfo(skip int) string {
	pc, file, line, ok := runtime.Caller(skip)
	if !ok {
		return "[unknown:0 unknown] "
	}

	funcName := "unknown"
	if fn := runtime.FuncForPC(pc); fn != nil {
		funcName = shortFuncName(fn.Name())
	}

	return fmt.Sprintf("[%s:%d %s] ", filepath.Base(file), line, funcName)
}

func shortFuncName(name string) string {
	if idx := strings.LastIndex(name, "/"); idx >= 0 {
		name = name[idx+1:]
	}
	return name
}

func Debug(format string, v ...any) { logf(level_debug, format, v...) }
func Info(format string, v ...any)  { logf(level_info, format, v...) }
func Warn(format string, v ...any)  { logf(level_warn, format, v...) }
func Error(format string, v ...any) { logf(level_error, format, v...) }
func Fatal(format string, v ...any) {
	logf(level_fatal, format, v...)
	// 用空内容的Fatal结束进程
	stdlog.Fatalf("")
}

func logf(lv Level, format string, v ...any) {
	if lv < minLevel {
		return
	}
	caller := callerInfo(3)

	prefix := "[" + levelNames[lv] + "] " + caller

	if fileLogger != nil {
		fileLogger.Printf(prefix+format, v...)
	}

	color := colorFor(lv)
	if termLogger != nil {
		termLogger.Printf(color+prefix+cReset+format+cReset, v...)
		return
	}
	stdlog.Printf(prefix+format, v...)
}

func colorFor(lv Level) string {
	switch lv {
	case level_debug:
		return cBlue
	case level_info:
		return cGreen
	case level_warn:
		return cYellow
	case level_error:
		return cRed
	case level_fatal:
		return cMagenta
	default:
		return cReset
	}
}
