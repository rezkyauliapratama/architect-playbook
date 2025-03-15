// pkg/logger/logger.go
package logger

import (
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/pkgerrors"
)

// Logger wraps zerolog.Logger to provide a unified logging interface
type Logger struct {
	zl zerolog.Logger
}

var (
	// defaultLogger is the global logger instance
	defaultLogger *Logger
)

// Config contains configuration options for the logger
type Config struct {
	// LogLevel sets the minimum level logs will be written
	LogLevel string
	// IsProduction determines output format (pretty console vs JSON)
	IsProduction bool
	// Output sets the destination for the logs (defaults to os.Stdout)
	Output io.Writer
}

// Initialize sets up the default logger with the provided configuration
func Initialize(cfg Config) {
	// Set up error stack traces
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack

	// Parse log level, defaulting to info if invalid
	level, err := zerolog.ParseLevel(cfg.LogLevel)
	if err != nil {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)

	// Set time format to ISO8601
	zerolog.TimeFieldFormat = time.RFC3339

	// Default output to stdout if none provided
	output := cfg.Output
	if output == nil {
		output = os.Stdout
	}

	var zl zerolog.Logger
	if cfg.IsProduction {
		// JSON structured logging for production
		zl = zerolog.New(output).With().Timestamp().Caller().Logger()
	} else {
		// Pretty console logging for development
		consoleWriter := zerolog.ConsoleWriter{
			Out:        output,
			TimeFormat: "15:04:05",
		}
		zl = zerolog.New(consoleWriter).With().Timestamp().Caller().Logger()
	}

	// Initialize the default logger
	defaultLogger = &Logger{zl: zl}
}

// Get returns the global logger instance
func Get() *Logger {
	if defaultLogger == nil {
		// Auto-initialize with sensible defaults if not already initialized
		Initialize(Config{
			LogLevel:     "info",
			IsProduction: false,
		})
	}
	return defaultLogger
}

// With adds key-value pairs to the logger context
func (l *Logger) With(key string, value interface{}) *Logger {
	return &Logger{zl: l.zl.With().Interface(key, value).Logger()}
}

// WithFields adds multiple fields to the logger context
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	context := l.zl.With()
	for k, v := range fields {
		context = context.Interface(k, v)
	}
	return &Logger{zl: context.Logger()}
}

// WithTransferID adds a transfer ID field to the logger
func (l *Logger) WithTransferID(transferID string) *Logger {
	return &Logger{zl: l.zl.With().Str("transfer_id", transferID).Logger()}
}

// Trace logs a message at trace level
func (l *Logger) Trace(msg string) {
	l.zl.Trace().Msg(msg)
}

// Debug logs a message at debug level
func (l *Logger) Debug(msg string) {
	l.zl.Debug().Msg(msg)
}

// Info logs a message at info level
func (l *Logger) Info(msg string) {
	l.zl.Info().Msg(msg)
}

// Warn logs a message at warn level
func (l *Logger) Warn(msg string) {
	l.zl.Warn().Msg(msg)
}

// Error logs a message at error level
func (l *Logger) Error(msg string, err error) {
	l.zl.Error().Err(err).Msg(msg)
}

// Fatal logs a message at fatal level and then calls os.Exit(1)
func (l *Logger) Fatal(msg string, err error) {
	l.zl.Fatal().Err(err).Msg(msg)
}

// TraceContext logs a message with context fields at trace level
func (l *Logger) TraceContext(msg string, ctx map[string]interface{}) {
	event := l.zl.Trace()
	for k, v := range ctx {
		event = event.Interface(k, v)
	}
	event.Msg(msg)
}

// DebugContext logs a message with context fields at debug level
func (l *Logger) DebugContext(msg string, ctx map[string]interface{}) {
	event := l.zl.Debug()
	for k, v := range ctx {
		event = event.Interface(k, v)
	}
	event.Msg(msg)
}

// InfoContext logs a message with context fields at info level
func (l *Logger) InfoContext(msg string, ctx map[string]interface{}) {
	event := l.zl.Info()
	for k, v := range ctx {
		event = event.Interface(k, v)
	}
	event.Msg(msg)
}

// WarnContext logs a message with context fields at warn level
func (l *Logger) WarnContext(msg string, ctx map[string]interface{}) {
	event := l.zl.Warn()
	for k, v := range ctx {
		event = event.Interface(k, v)
	}
	event.Msg(msg)
}

// ErrorContext logs a message with context fields at error level
func (l *Logger) ErrorContext(msg string, err error, ctx map[string]interface{}) {
	event := l.zl.Error().Err(err)
	for k, v := range ctx {
		event = event.Interface(k, v)
	}
	event.Msg(msg)
}
