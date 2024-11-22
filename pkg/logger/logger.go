package logger

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	LogFile       = "$HOME/.config/corectl/log.json"
	LogMaxSize    = 10
	LogMaxBackups = 5
	LogMaxAge     = 30
	LogCompress   = true

	DefaultTailLines = 20

	TimeFieldKey    = "ts"
	MessageFieldKey = "status"
)

var (
	Log     *zap.Logger
	LogPath string
	// AtomicLevel is used to set the terminal log level
	// Note: we log everything to the log file, but only print
	//        to the terminal at the level specified here
	AtomicLevel zap.AtomicLevel
)

func Init() error {
	if Log != nil {
		return nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("Failed to get home directory: %v\n", err)
		os.Exit(1)
	}
	LogPath = strings.Replace(LogFile, "$HOME", homeDir, 1)

	logDir := filepath.Dir(LogPath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return err
	}

	AtomicLevel = zap.NewAtomicLevel()
	AtomicLevel.SetLevel(zap.InfoLevel)

	logFile := &lumberjack.Logger{
		Filename:   LogPath,
		MaxSize:    LogMaxSize,
		MaxBackups: LogMaxBackups,
		MaxAge:     LogMaxAge,
		Compress:   LogCompress,
	}

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = TimeFieldKey
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.MessageKey = MessageFieldKey

	fileCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.AddSync(logFile),
		zapcore.DebugLevel,
	)

	consoleCore := zapcore.NewCore(
		zapcore.NewConsoleEncoder(encoderConfig),
		zapcore.AddSync(os.Stdout),
		AtomicLevel,
	)

	core := zapcore.NewTee(fileCore, consoleCore)
	Log = zap.New(core)
	return nil
}

// Debug logs a message at Debug level
func Debug(msg string, fields ...zap.Field) {
	Log.Debug(msg, fields...)
}

// Info logs a message at Info level
func Info(msg string, fields ...zap.Field) {
	Log.Info(msg, fields...)
}

// Warn logs a message at Warn level
func Warn(msg string, fields ...zap.Field) {
	Log.Warn(msg, fields...)
}

// Error logs a message at Error level
func Error(msg string, fields ...zap.Field) {
	Log.Error(msg, fields...)
}

// Fatal logs a message at Fatal level and then calls os.Exit(1)
func Fatal(msg string, fields ...zap.Field) {
	Log.Fatal(msg, fields...)
}

// Fatal logs a message at Fatal level and then calls os.Exit(1)
func Panic(msg string, fields ...zap.Field) {
	Log.Panic(msg, fields...)
}

// With creates a child logger with the given fields
func With(fields ...zap.Field) *zap.Logger {
	return Log.With(fields...)
}

// Sync flushes any buffered log entries
func Sync() {
	err := Log.Sync()

	// Ignore sync errors from stdout/stderr
	// https://github.com/uber-go/zap/issues/880
	// Also Skip sync errors in GH Actions
	if os.Getenv("GITHUB_ACTIONS") == "true" || !errors.Is(err, syscall.ENOTTY) {
		return
	}

	if err != nil && !errors.Is(err, syscall.ENOTTY) {
		panic(fmt.Sprintf("failed to sync logger: %v", err))
	}
}
