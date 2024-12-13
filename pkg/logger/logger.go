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
	LogFile       = "$HOME/.config/corectl/corectl.log"
	LogMaxSize    = 10
	LogMaxBackups = 5
	LogMaxAge     = 30
	LogCompress   = true

	DefaultTailLines = 20

	TimeFieldKey    = "ts"
	MessageFieldKey = "status"
)

type CECGLogger struct {
	logger *zap.Logger
}
type CECGLoggerEntry struct {
	logger *zap.Logger
	level  zapcore.Level
}

var (
	cecgLogger *CECGLogger
	logPath    string
)

func defaultInit() *CECGLogger {
	return Init(zapcore.InfoLevel.String())
}
func Init(logLevelFlag string) *CECGLogger {
	if cecgLogger != nil {
		return cecgLogger
	}

	consoleLogLevel, err := zapcore.ParseLevel(logLevelFlag)
	if err != nil {
		consoleLogLevel = zapcore.WarnLevel
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("Failed to get home directory: %v\n", err)
		os.Exit(1)
	}
	logPath = strings.Replace(LogFile, "$HOME", homeDir, 1)

	logDir := filepath.Dir(logPath)
	if err := os.MkdirAll(logDir, 0644); err != nil {
		fmt.Printf("fail to create folder path: %v\n", err)
		os.Exit(1)
	}

	fileWriter := zapcore.AddSync(&lumberjack.Logger{
		Filename:   logPath,
		MaxSize:    LogMaxSize,
		MaxBackups: LogMaxBackups,
		MaxAge:     LogMaxAge,
		Compress:   LogCompress,
	})

	consoleWriter := zapcore.Lock(zapcore.AddSync(os.Stdout))

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = TimeFieldKey
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.MessageKey = MessageFieldKey

	jsonEncoder := zapcore.NewJSONEncoder(zap.NewDevelopmentEncoderConfig())
	consoleEncoder := zapcore.NewConsoleEncoder(encoderConfig)

	// Combine outputs with configurable log level
	core := zapcore.NewTee(
		zapcore.NewCore(jsonEncoder, fileWriter, zapcore.InfoLevel),
		zapcore.NewCore(consoleEncoder, consoleWriter, consoleLogLevel),
	)
	cecgLogger = &CECGLogger{logger: zap.New(core)}

	Debug().With(zap.String("value", consoleLogLevel.String())).Msg("log level configured")
	return cecgLogger
}

// Debug logs a message at Debug level
func Debug() *CECGLoggerEntry {
	defaultInit()
	return &CECGLoggerEntry{logger: cecgLogger.logger, level: zapcore.DebugLevel}
}

// Info logs a message at Info level
func Info() *CECGLoggerEntry {
	defaultInit()
	return &CECGLoggerEntry{logger: cecgLogger.logger, level: zapcore.InfoLevel}
}

// Info logs a message at Info level
func Warn() *CECGLoggerEntry {
	defaultInit()
	return &CECGLoggerEntry{logger: cecgLogger.logger, level: zapcore.WarnLevel}
}

// Info logs a message at Info level
func Error() *CECGLoggerEntry {
	defaultInit()
	return &CECGLoggerEntry{logger: cecgLogger.logger, level: zapcore.ErrorLevel}
}

// Info logs a message at Info level
func Fatal() *CECGLoggerEntry {
	defaultInit()
	return &CECGLoggerEntry{logger: cecgLogger.logger, level: zapcore.FatalLevel}
}

// Info logs a message at Info level
func Panic() *CECGLoggerEntry {
	defaultInit()
	return &CECGLoggerEntry{logger: cecgLogger.logger, level: zapcore.PanicLevel}
}

func (e *CECGLoggerEntry) Msg(msg string) {
	e.logger.Log(e.level, msg)
}

func (e *CECGLoggerEntry) Msgf(msg string, args ...interface{}) {
	e.logger.Log(e.level, fmt.Sprintf(msg, args...))
}

// With creates a child logger with the given fields
func (e *CECGLoggerEntry) With(fields ...zap.Field) *CECGLoggerEntry {
	e.logger = e.logger.With(fields...)
	return e
}

// Sync flushes any buffered log entries
func Sync() {
	defaultInit()
	err := cecgLogger.logger.Sync()

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
