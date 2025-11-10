package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"
)

// LogLevel represents the severity of a log message
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
	FATAL
)

// String returns the string representation of the log level
func (l LogLevel) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	case FATAL:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// Logger is the global logger instance
type Logger struct {
	writer      io.Writer
	mu          sync.Mutex
	logPath     string
	minLevel    LogLevel
	initialized bool
}

var (
	globalLogger *Logger
	once         sync.Once
)

// Initialize sets up the logger with log rotation
func Initialize() error {
	var initErr error
	once.Do(func() {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			initErr = fmt.Errorf("failed to get home directory: %w", err)
			return
		}

		logDir := filepath.Join(homeDir, ".oni", "logs")
		if err := os.MkdirAll(logDir, 0755); err != nil {
			initErr = fmt.Errorf("failed to create log directory: %w", err)
			return
		}

		logPath := filepath.Join(logDir, "oni.log")

		// Set up log rotation with lumberjack
		lumberjackLogger := &lumberjack.Logger{
			Filename:   logPath,
			MaxSize:    10,   // megabytes
			MaxBackups: 5,    // keep 5 old log files
			MaxAge:     30,   // days
			Compress:   true, // compress old log files
		}

		globalLogger = &Logger{
			writer:      lumberjackLogger,
			logPath:     logPath,
			minLevel:    DEBUG, // Log everything by default
			initialized: true,
		}
	})

	return initErr
}

// GetLogFilePath returns the path to the current log file
func GetLogFilePath() string {
	if globalLogger == nil || !globalLogger.initialized {
		return ""
	}
	return globalLogger.logPath
}

// SetMinLevel sets the minimum log level to record
func SetMinLevel(level LogLevel) {
	if globalLogger != nil {
		globalLogger.mu.Lock()
		globalLogger.minLevel = level
		globalLogger.mu.Unlock()
	}
}

// formatFields converts a map of fields to a string representation
func formatFields(fields map[string]interface{}) string {
	if len(fields) == 0 {
		return ""
	}

	var parts []string
	for k, v := range fields {
		parts = append(parts, fmt.Sprintf("%s=%v", k, v))
	}
	return fmt.Sprintf("{%s}", strings.Join(parts, ", "))
}

// getCallerInfo returns the file, line, and function name of the caller
func getCallerInfo(skip int) string {
	pc, file, line, ok := runtime.Caller(skip)
	if !ok {
		return "unknown:0:unknown"
	}

	// Get just the filename, not the full path
	filename := filepath.Base(file)

	// Get function name
	funcName := "unknown"
	fn := runtime.FuncForPC(pc)
	if fn != nil {
		fullName := fn.Name()
		// Extract just the function name without package path
		parts := strings.Split(fullName, "/")
		if len(parts) > 0 {
			lastPart := parts[len(parts)-1]
			funcParts := strings.Split(lastPart, ".")
			if len(funcParts) > 1 {
				funcName = funcParts[len(funcParts)-1]
			}
		}
	}

	return fmt.Sprintf("%s:%d:%s", filename, line, funcName)
}

// log writes a log message with the specified level
func log(level LogLevel, msg string, fields map[string]interface{}) {
	if globalLogger == nil || !globalLogger.initialized {
		// If logger is not initialized, silently return
		return
	}

	if level < globalLogger.minLevel {
		return
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	caller := getCallerInfo(3) // Skip log(), the level function, and the actual caller
	fieldsStr := formatFields(fields)

	var logLine string
	if fieldsStr != "" {
		logLine = fmt.Sprintf("[%s] [%s] [%s] %s %s\n", timestamp, level.String(), caller, msg, fieldsStr)
	} else {
		logLine = fmt.Sprintf("[%s] [%s] [%s] %s\n", timestamp, level.String(), caller, msg)
	}

	globalLogger.mu.Lock()
	defer globalLogger.mu.Unlock()

	_, _ = globalLogger.writer.Write([]byte(logLine))
}

// Debug logs a debug message
func Debug(msg string, fields map[string]interface{}) {
	log(DEBUG, msg, fields)
}

// Info logs an info message
func Info(msg string, fields map[string]interface{}) {
	log(INFO, msg, fields)
}

// Warn logs a warning message
func Warn(msg string, fields map[string]interface{}) {
	log(WARN, msg, fields)
}

// Error logs an error message
func Error(msg string, err error, fields map[string]interface{}) {
	if fields == nil {
		fields = make(map[string]interface{})
	}
	if err != nil {
		fields["error"] = err.Error()
	}
	log(ERROR, msg, fields)
}

// Fatal logs a fatal error message
func Fatal(msg string, err error, fields map[string]interface{}) {
	if fields == nil {
		fields = make(map[string]interface{})
	}
	if err != nil {
		fields["error"] = err.Error()
	}
	log(FATAL, msg, fields)
}
