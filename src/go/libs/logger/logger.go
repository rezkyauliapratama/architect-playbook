// src/go/libs/logger/logger.go
package logger

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/pkgerrors"
)

// Logger wraps zerolog.Logger for unified logging interface
type Logger struct {
	zl     zerolog.Logger
	writer io.Writer
}

var (
	defaultLogger *Logger
)

// Config contains logger configuration
type Config struct {
	LogLevel     string
	IsProduction bool
	Output       io.Writer
	ServiceName  string
	Version      string
	EnableCaller bool

	// 🚀 FUTURE: Async settings
	// EnableAsync      bool
	// AsyncBufferSize  int
	// AsyncFlushPeriod time.Duration
	// AsyncWorkers     int
}

// Initialize sets up the default logger
func Initialize(cfg Config) {
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack

	level, err := zerolog.ParseLevel(cfg.LogLevel)
	if err != nil {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)

	zerolog.TimeFieldFormat = time.RFC3339
	zerolog.TimestampFieldName = "timestamp"

	output := cfg.Output
	if output == nil {
		output = os.Stdout
	}

	writer := output

	// Create zerolog instance
	zl := zerolog.New(writer).With().Timestamp().Logger()

	// ✅ FIX: Add caller with skip frame to point to actual caller, not wrapper
	if cfg.EnableCaller {
		// CallerWithSkipFrameCount(2) skips:
		// Frame 0: runtime.Caller
		// Frame 1: zerolog caller
		// Frame 2: our wrapper method (Info, Error, etc.)
		// Frame 3: actual caller (service/handler) ← We want this!
		zl = zl.With().CallerWithSkipFrameCount(3).Logger()
	}

	// Add service metadata
	if cfg.ServiceName != "" {
		zl = zl.With().Str("service", cfg.ServiceName).Logger()
	}
	if cfg.Version != "" {
		zl = zl.With().Str("version", cfg.Version).Logger()
	}

	// Add environment tag
	environment := getEnvironment(cfg.IsProduction)
	zl = zl.With().Str("environment", environment).Logger()

	defaultLogger = &Logger{
		zl:     zl,
		writer: writer,
	}
}

// Get returns the global logger instance
func Get() *Logger {
	if defaultLogger == nil {
		Initialize(Config{
			LogLevel:     "info",
			IsProduction: true,
			EnableCaller: true,
		})
	}
	return defaultLogger
}

// ==========================================
// Context Builders (Chainable)
// ==========================================

// WithField adds a key-value pair to logger context
func (l *Logger) WithField(key string, value interface{}) *Logger {
	return &Logger{
		zl:     l.zl.With().Interface(key, value).Logger(),
		writer: l.writer,
	}
}

// WithFields adds multiple fields to logger context
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	ctx := l.zl.With()
	for k, v := range fields {
		ctx = ctx.Interface(k, v)
	}
	return &Logger{
		zl:     ctx.Logger(),
		writer: l.writer,
	}
}

// WithTransactionID adds transaction ID for payment tracing
func (l *Logger) WithTransactionID(txnID string) *Logger {
	return &Logger{
		zl:     l.zl.With().Str("transaction_id", txnID).Logger(),
		writer: l.writer,
	}
}

// WithTransferID adds transfer ID for BI-FAST transfer tracing (alias)
func (l *Logger) WithTransferID(transferID string) *Logger {
	return l.WithTransactionID(transferID)
}

// WithRequestID adds request ID
func (l *Logger) WithRequestID(requestID string) *Logger {
	return &Logger{
		zl:     l.zl.With().Str("request_id", requestID).Logger(),
		writer: l.writer,
	}
}

// WithCorrelationID adds correlation ID for distributed tracing
func (l *Logger) WithCorrelationID(correlationID string) *Logger {
	return &Logger{
		zl:     l.zl.With().Str("correlation_id", correlationID).Logger(),
		writer: l.writer,
	}
}

// WithUserID adds user ID for audit trail
func (l *Logger) WithUserID(userID string) *Logger {
	return &Logger{
		zl:     l.zl.With().Str("user_id", userID).Logger(),
		writer: l.writer,
	}
}

// WithContext extracts IDs from context.Context and adds them to logger
func (l *Logger) WithContext(ctx context.Context) *Logger {
	logger := l
	if requestID, ok := ctx.Value("request_id").(string); ok && requestID != "" {
		logger = logger.WithRequestID(requestID)
	}
	if correlationID, ok := ctx.Value("correlation_id").(string); ok && correlationID != "" {
		logger = logger.WithCorrelationID(correlationID)
	}
	if userID, ok := ctx.Value("user_id").(string); ok && userID != "" {
		logger = logger.WithUserID(userID)
	}
	if transactionID, ok := ctx.Value("transaction_id").(string); ok && transactionID != "" {
		logger = logger.WithTransactionID(transactionID)
	}
	return logger
}

// ==========================================
// Simple Logging Methods
// ==========================================

// Trace logs at trace level
func (l *Logger) Trace(msg string) {
	l.zl.Trace().Msg(msg)
}

// Debug logs at debug level
func (l *Logger) Debug(msg string) {
	l.zl.Debug().Msg(msg)
}

// Info logs at info level
func (l *Logger) Info(msg string) {
	l.zl.Info().Msg(msg)
}

// Warn logs at warn level
func (l *Logger) Warn(msg string) {
	l.zl.Warn().Msg(msg)
}

// Error logs at error level with error object
func (l *Logger) Error(msg string, err error) {
	l.zl.Error().Err(err).Msg(msg)
}

// Fatal logs at fatal level and exits
func (l *Logger) Fatal(msg string, err error) {
	l.zl.Fatal().Err(err).Msg(msg)
}

// ==========================================
// Context Logging Methods (Recommended)
// ==========================================

// TraceContext logs with additional context fields
func (l *Logger) TraceContext(msg string, ctx map[string]interface{}) {
	event := l.zl.Trace()
	for k, v := range ctx {
		event = event.Interface(k, v)
	}
	event.Msg(msg)
}

// DebugContext logs with additional context fields
func (l *Logger) DebugContext(msg string, ctx map[string]interface{}) {
	event := l.zl.Debug()
	for k, v := range ctx {
		event = event.Interface(k, v)
	}
	event.Msg(msg)
}

// InfoContext logs with additional context fields
func (l *Logger) InfoContext(msg string, ctx map[string]interface{}) {
	event := l.zl.Info()
	for k, v := range ctx {
		event = event.Interface(k, v)
	}
	event.Msg(msg)
}

// WarnContext logs with additional context fields
func (l *Logger) WarnContext(msg string, ctx map[string]interface{}) {
	event := l.zl.Warn()
	for k, v := range ctx {
		event = event.Interface(k, v)
	}
	event.Msg(msg)
}

// ErrorContext logs with additional context fields and error
func (l *Logger) ErrorContext(msg string, err error, ctx map[string]interface{}) {
	event := l.zl.Error().Err(err)
	for k, v := range ctx {
		event = event.Interface(k, v)
	}
	event.Msg(msg)
}

// FatalContext logs with additional context fields and exits
func (l *Logger) FatalContext(msg string, err error, ctx map[string]interface{}) {
	event := l.zl.Fatal().Err(err)
	for k, v := range ctx {
		event = event.Interface(k, v)
	}
	event.Msg(msg)
}

// ==========================================
// Helper Methods
// ==========================================

// LogOperation logs an operation with duration and performance classification
func (l *Logger) LogOperation(operation string, duration time.Duration, ctx map[string]interface{}) {
	if ctx == nil {
		ctx = make(map[string]interface{})
	}

	ctx["operation"] = operation
	ctx["duration_ms"] = duration.Milliseconds()

	// Auto-classify performance for APM alerting
	if duration > 3*time.Second {
		ctx["performance"] = "critical"
		l.ErrorContext("Operation critical slowness", nil, ctx)
	} else if duration > 1*time.Second {
		ctx["performance"] = "slow"
		l.WarnContext("Operation slow", ctx)
	} else if duration > 500*time.Millisecond {
		ctx["performance"] = "acceptable"
		l.InfoContext("Operation completed", ctx)
	} else {
		ctx["performance"] = "fast"
		l.DebugContext("Operation completed", ctx)
	}
}

// LogHTTPRequest logs HTTP request details
func (l *Logger) LogHTTPRequest(method, path string, statusCode int, duration time.Duration, ctx map[string]interface{}) {
	if ctx == nil {
		ctx = make(map[string]interface{})
	}

	ctx["http_method"] = method
	ctx["http_path"] = path
	ctx["http_status"] = statusCode
	ctx["duration_ms"] = duration.Milliseconds()

	// HTTP status-based log levels
	if statusCode >= 500 {
		l.ErrorContext("HTTP request server error", nil, ctx)
	} else if statusCode >= 400 {
		l.WarnContext("HTTP request client error", ctx)
	} else {
		l.InfoContext("HTTP request completed", ctx)
	}
}

// GetZerolog returns underlying zerolog.Logger for advanced usage
func (l *Logger) GetZerolog() zerolog.Logger {
	return l.zl
}

// ==========================================
// Utility Functions
// ==========================================

func getEnvironment(isProduction bool) string {
	env := os.Getenv("ENVIRONMENT")
	if env != "" {
		return env
	}

	if isProduction {
		return "production"
	}
	return "development"
}
