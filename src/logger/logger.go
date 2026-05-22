package logger

import (
	"compress/gzip"
	"fmt"
	"io"
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

func normalizeOptions(options Options) Options {
	if options.Path == "" {
		options.Path = logPath
	}
	if options.Level == "" {
		options.Level = "INFO"
	}
	if options.MaxDay <= 0 {
		options.MaxDay = 7
	}
	return options
}

func Open(options Options) {
	options = normalizeOptions(options)
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
		rotatelogs.WithHandler(rotatelogs.HandlerFunc(func(e rotatelogs.Event) {
			if e.Type() != rotatelogs.FileRotatedEventType {
				return
			}

			rotated, ok := e.(*rotatelogs.FileRotatedEvent)
			if !ok {
				return
			}
			if err := compressLogFile(rotated.PreviousFile()); err != nil {
				stdlog.Printf("压缩日志失败: %v", err)
			}
			if err := cleanupCompressedLogs(dir, options.MaxDay); err != nil {
				stdlog.Printf("清理压缩日志失败: %v", err)
			}
		})),
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

	if err := compressExistingLogs(dir, fileSink.CurrentFileName()); err != nil {
		stdlog.Printf("压缩已有日志失败: %v", err)
	}
	if err := cleanupCompressedLogs(dir, options.MaxDay); err != nil {
		stdlog.Printf("清理压缩日志失败: %v", err)
	}
}

func ApplyOptions(options Options) {
	options = normalizeOptions(options)
	minLevel = parseLevel(options.Level)

	dir := options.Path
	currentFile := ""
	if fileSink != nil {
		currentFile = fileSink.CurrentFileName()
	}
	if err := compressExistingLogs(dir, currentFile); err != nil {
		stdlog.Printf("压缩已有日志失败: %v", err)
	}
	if err := cleanupCompressedLogs(dir, options.MaxDay); err != nil {
		stdlog.Printf("清理压缩日志失败: %v", err)
	}
}

// Close 关闭logger
func Close() {
	if fileSink != nil {
		_ = fileSink.Close()
		fileSink = nil
	}
}

func compressLogFile(path string) error {
	if path == "" || strings.HasSuffix(path, ".gz") {
		return nil
	}

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	gzPath := path + ".gz"
	if _, err := os.Stat(gzPath); err == nil {
		return os.Remove(path)
	} else if !os.IsNotExist(err) {
		return err
	}

	in, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer in.Close()

	tmpPath := gzPath + ".tmp"
	out, err := os.Create(tmpPath)
	if err != nil {
		return err
	}

	gz := gzip.NewWriter(out)
	_, copyErr := io.Copy(gz, in)
	closeErr := gz.Close()
	fileCloseErr := out.Close()
	if copyErr != nil {
		_ = os.Remove(tmpPath)
		return copyErr
	}
	if closeErr != nil {
		_ = os.Remove(tmpPath)
		return closeErr
	}
	if fileCloseErr != nil {
		_ = os.Remove(tmpPath)
		return fileCloseErr
	}

	if err := os.Rename(tmpPath, gzPath); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}

	if err := os.Chtimes(gzPath, info.ModTime(), info.ModTime()); err != nil {
		return err
	}

	return os.Remove(path)
}

func compressExistingLogs(dir, currentFile string) error {
	if currentFile != "" {
		absCurrentFile, err := filepath.Abs(currentFile)
		if err != nil {
			return err
		}
		currentFile = absCurrentFile
	}

	files, err := filepath.Glob(filepath.Join(dir, "*.log"))
	if err != nil {
		return err
	}

	for _, file := range files {
		base := filepath.Base(file)
		if base == "current.log" {
			continue
		}

		absFile, err := filepath.Abs(file)
		if err != nil {
			return err
		}
		if currentFile != "" && absFile == currentFile {
			continue
		}

		if err := compressLogFile(file); err != nil {
			return err
		}
	}

	return nil
}

func cleanupCompressedLogs(dir string, maxDay int) error {
	files, err := filepath.Glob(filepath.Join(dir, "*.log.gz"))
	if err != nil {
		return err
	}

	for _, file := range files {
		expired, err := compressedLogExpired(file, maxDay)
		if err != nil {
			return err
		}
		if expired {
			if err := os.Remove(file); err != nil && !os.IsNotExist(err) {
				return err
			}
		}
	}

	return nil
}

func compressedLogExpired(path string, maxDay int) (bool, error) {
	cutoff := time.Now().Add(-time.Duration(maxDay) * 24 * time.Hour)
	base := filepath.Base(path)
	datePart := strings.TrimSuffix(base, ".log.gz")
	if t, err := time.ParseInLocation("2006-01-02", datePart, time.Local); err == nil {
		return t.Before(cutoff), nil
	}

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return info.ModTime().Before(cutoff), nil
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
