package logger

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/prasetyowira/shorter/constant"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

var logger *zap.Logger

// LoggerContext represents the context for log entries
type LoggerContext struct {
	RequestID string
	Context   string
	Filename  string
}

// LoggerInfo contains structured logging information
type LoggerInfo struct {
	ContextFunction string
	Error           *CustomError
	Data            map[string]interface{}
}

// CustomError represents a structured error for logging
type CustomError struct {
	Code    string
	Message string
	Type    string
}

// Initialize sets up the logger
func Initialize(isProduction bool) {
	// Default level
	logLevel := zap.NewAtomicLevelAt(zapcore.DebugLevel)
	if isProduction {
		logLevel = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	}

	// Create encoder config
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        constant.LogTimeKey,
		LevelKey:       constant.LogLevelKey,
		NameKey:        constant.LogNameKey,
		CallerKey:      constant.LogCallerKey,
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     constant.LogMessageKey,
		StacktraceKey:  constant.LogStacktraceKey,
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// Create config
	var config zap.Config
	if isProduction {
		config = zap.Config{
			Level:       logLevel,
			Development: false,
			Sampling: &zap.SamplingConfig{
				Initial:    100,
				Thereafter: 100,
			},
			Encoding:         constant.LogEncodingJSON,
			EncoderConfig:    encoderConfig,
			OutputPaths:      []string{constant.LogOutputStdout},
			ErrorOutputPaths: []string{constant.LogOutputStderr},
		}
	} else {
		config = zap.Config{
			Level:            logLevel,
			Development:      true,
			Encoding:         constant.LogEncodingConsole,
			EncoderConfig:    encoderConfig,
			OutputPaths:      []string{constant.LogOutputStdout},
			ErrorOutputPaths: []string{constant.LogOutputStderr},
		}
	}

	// Build the logger
	var err error
	logger, err = config.Build()
	if err != nil {
		// If we can't initialize the logger, we're in serious trouble
		// Fall back to stderr and exit
		os.Stderr.WriteString("failed to initialize logger: " + err.Error() + "\n")
		os.Exit(1)
	}

	// Defer syncing logs on shutdown
	// Intentionally not calling defer logger.Sync() here as it would never get called
	// The application should call Close() on shutdown
}

// Close ensures logger syncs before shutdown
func Close() {
	if logger != nil {
		_ = logger.Sync()
	}
}

// GetLoggerContext retrieves context information from the context
func GetLoggerContext(ctx context.Context) LoggerContext {
	var requestID string
	if id := ctx.Value("request_id"); id != nil {
		requestID = id.(string)
	} else {
		requestID = uuid.New().String()
	}

	// Get caller file information
	_, file, _, ok := runtime.Caller(2)
	var filename string
	if ok {
		filename = filepath.Base(file)
	} else {
		filename = "unknown"
	}

	return LoggerContext{
		RequestID: requestID,
		Filename:  filename,
	}
}

// createFields creates zap fields with proper structure
func createFields(ctx context.Context, info LoggerInfo) []zap.Field {
	fields := []zap.Field{}

	// Add request ID if available
	if requestID := getRequestID(ctx); requestID != "" {
		fields = append(fields, zap.String(constant.LogRequestIDKey, requestID))
	}

	// Add context/function info
	if info.ContextFunction != "" {
		fields = append(fields, zap.String(constant.LogFunctionKey, info.ContextFunction))
	}

	// Add error details if available
	if info.Error != nil {
		fields = append(fields, zap.String(constant.LogErrorCodeKey, info.Error.Code))
		fields = append(fields, zap.String(constant.LogErrorTypeKey, info.Error.Type))
		fields = append(fields, zap.String(constant.LogErrorMessageKey, info.Error.Message))
	}

	// Add additional data
	if info.Data != nil {
		for k, v := range info.Data {
			fields = append(fields, zap.Any(k, v))
		}
	}

	return fields
}

// Debug logs a debug message
func Debug(msg string, info LoggerInfo) {
	if logger == nil {
		return
	}
	logger.Debug(msg, createFields(nil, info)...)
}

// Info logs an info message
func Info(msg string, info LoggerInfo) {
	if logger == nil {
		return
	}
	logger.Info(msg, createFields(nil, info)...)
}

// Warn logs a warning message
func Warn(msg string, info LoggerInfo) {
	if logger == nil {
		return
	}
	logger.Warn(msg, createFields(nil, info)...)
}

// Error logs an error message
func Error(msg string, info LoggerInfo) {
	if logger == nil {
		return
	}
	logger.Error(msg, createFields(nil, info)...)
}

// Fatal logs a fatal message and exits
func Fatal(msg string, info LoggerInfo) {
	if logger == nil {
		os.Exit(1)
	}
	logger.Fatal(msg, createFields(nil, info)...)
}

// CtxDebug logs a debug message with context
func CtxDebug(ctx context.Context, msg string, info LoggerInfo) {
	if logger == nil {
		return
	}
	logger.Debug(msg, createFields(ctx, info)...)
}

// CtxInfo logs an info message with context
func CtxInfo(ctx context.Context, msg string, info LoggerInfo) {
	if logger == nil {
		return
	}
	logger.Info(msg, createFields(ctx, info)...)
}

// CtxWarn logs a warning message with context
func CtxWarn(ctx context.Context, msg string, info LoggerInfo) {
	if logger == nil {
		return
	}
	logger.Warn(msg, createFields(ctx, info)...)
}

// CtxError logs an error message with context
func CtxError(ctx context.Context, msg string, info LoggerInfo) {
	if logger == nil {
		return
	}
	logger.Error(msg, createFields(ctx, info)...)
}

// CtxFatal logs a fatal message with context and exits
func CtxFatal(ctx context.Context, msg string, info LoggerInfo) {
	if logger == nil {
		os.Exit(1)
	}
	logger.Fatal(msg, createFields(ctx, info)...)
}

// NewRequestContext creates a new context for a request
func NewRequestContext() context.Context {
	return context.Background()
}

// WithRequestID adds a request ID to the context
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, constant.RequestIDKey, requestID)
}

// getRequestID gets the request ID from the context
func getRequestID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}

	if reqID, ok := ctx.Value(constant.RequestIDKey).(string); ok {
		return reqID
	}

	return ""
}

// FormatMetadata formats map data into key=value • key=value format
func FormatMetadata(data map[string]interface{}) string {
	if len(data) == 0 {
		return ""
	}

	var parts []string
	for k, v := range data {
		parts = append(parts, fmt.Sprintf("%s=%v", k, v))
	}
	return strings.Join(parts, " • ")
}
