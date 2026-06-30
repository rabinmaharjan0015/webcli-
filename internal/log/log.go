package log

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

type Logger struct {
	mu      sync.Mutex
	level   Level
	format  string
	writer  io.Writer
	appName string
}

var defaultLogger = New(LevelInfo, "text", os.Stderr, "webcli")

func New(level Level, format string, writer io.Writer, appName string) *Logger {
	return &Logger{
		level:   level,
		format:  format,
		writer:  writer,
		appName: appName,
	}
}

func ParseLevel(s string) Level {
	switch strings.ToLower(s) {
	case "debug":
		return LevelDebug
	case "info":
		return LevelInfo
	case "warn", "warning":
		return LevelWarn
	case "error":
		return LevelError
	default:
		return LevelInfo
	}
}

func Init(levelStr, format string) {
	level := ParseLevel(levelStr)
	if format != "json" && format != "text" {
		format = "text"
	}
	defaultLogger.mu.Lock()
	defer defaultLogger.mu.Unlock()
	defaultLogger.level = level
	defaultLogger.format = format
}

func Debug(format string, args ...interface{}) {
	defaultLogger.log(LevelDebug, format, args...)
}

func Info(format string, args ...interface{}) {
	defaultLogger.log(LevelInfo, format, args...)
}

func Warn(format string, args ...interface{}) {
	defaultLogger.log(LevelWarn, format, args...)
}

func Error(format string, args ...interface{}) {
	defaultLogger.log(LevelError, format, args...)
}

func (l *Logger) log(level Level, format string, args ...interface{}) {
	if level < l.level {
		return
	}

	msg := fmt.Sprintf(format, args...)

	l.mu.Lock()
	defer l.mu.Unlock()

	switch l.format {
	case "json":
		entry := map[string]interface{}{
			"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
			"level":     level.String(),
			"app":       l.appName,
			"message":   msg,
		}
		data, _ := json.Marshal(entry)
		fmt.Fprintln(l.writer, string(data))
	default:
		timestamp := time.Now().Format("2006/01/02 15:04:05")
		fmt.Fprintf(l.writer, "%s %s %s\n", timestamp, level.String(), msg)
	}
}
