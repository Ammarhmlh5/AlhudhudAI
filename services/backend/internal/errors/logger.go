package errors

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime"
	"strings"
)

// Logger provides error logging with stack traces
type Logger struct {
	logger *slog.Logger
}

// NewLogger creates a new error logger
func NewLogger(logger *slog.Logger) *Logger {
	return &Logger{
		logger: logger,
	}
}

// LogError logs an error with full details and stack trace
func (l *Logger) LogError(ctx context.Context, err error, message string, args ...interface{}) {
	var appErr *AppError
	if !errors.As(err, &appErr) {
		// Wrap unknown error
		appErr = Wrap(err, CodeInternalError, CategorySystem, message)
	}

	// Add stack trace if not present
	if appErr.StackTrace == "" {
		appErr.StackTrace = captureStackTrace(3)
	}

	// Build log attributes
	attrs := []slog.Attr{
		slog.String("error_code", appErr.Code),
		slog.String("error_category", appErr.Category),
		slog.String("error_message", appErr.Message),
	}

	// Add details if present
	if len(appErr.Details) > 0 {
		for key, value := range appErr.Details {
			attrs = append(attrs, slog.Any(key, value))
		}
	}

	// Add stack trace
	attrs = append(attrs, slog.String("stack_trace", appErr.StackTrace))

	// Add underlying error if present
	if appErr.Err != nil {
		attrs = append(attrs, slog.String("underlying_error", appErr.Err.Error()))
	}

	// Add additional arguments
	for i := 0; i < len(args); i += 2 {
		if i+1 < len(args) {
			key := fmt.Sprintf("%v", args[i])
			value := args[i+1]
			attrs = append(attrs, slog.Any(key, value))
		}
	}

	// Log based on criticality
	if IsCritical(appErr) {
		l.logger.LogAttrs(ctx, slog.LevelError, message, attrs...)
	} else {
		l.logger.LogAttrs(ctx, slog.LevelWarn, message, attrs...)
	}
}

// LogErrorWithContext logs an error with additional context
func (l *Logger) LogErrorWithContext(ctx context.Context, err error, message string, contextData map[string]interface{}) {
	var appErr *AppError
	if !errors.As(err, &appErr) {
		appErr = Wrap(err, CodeInternalError, CategorySystem, message)
	}

	// Add context data to error details
	for key, value := range contextData {
		appErr.WithDetails(key, value)
	}

	l.LogError(ctx, appErr, message)
}

// captureStackTrace captures the current stack trace
func captureStackTrace(skip int) string {
	const maxDepth = 32
	var pcs [maxDepth]uintptr
	n := runtime.Callers(skip, pcs[:])

	var builder strings.Builder
	frames := runtime.CallersFrames(pcs[:n])

	for {
		frame, more := frames.Next()
		
		// Skip runtime frames
		if !strings.Contains(frame.File, "runtime/") {
			builder.WriteString(fmt.Sprintf("%s\n\t%s:%d\n", frame.Function, frame.File, frame.Line))
		}

		if !more {
			break
		}
	}

	return builder.String()
}

// WithStackTrace adds stack trace to an error
func WithStackTrace(err error) error {
	var appErr *AppError
	if !errors.As(err, &appErr) {
		appErr = Wrap(err, CodeInternalError, CategorySystem, err.Error())
	}

	if appErr.StackTrace == "" {
		appErr.StackTrace = captureStackTrace(2)
	}

	return appErr
}

// LogAndReturn logs an error and returns it
func (l *Logger) LogAndReturn(ctx context.Context, err error, message string, args ...interface{}) error {
	l.LogError(ctx, err, message, args...)
	return err
}

// LogCritical logs a critical error that requires immediate attention
func (l *Logger) LogCritical(ctx context.Context, err error, message string, args ...interface{}) {
	var appErr *AppError
	if !errors.As(err, &appErr) {
		appErr = Wrap(err, CodeInternalError, CategorySystem, message)
	}

	// Add stack trace
	if appErr.StackTrace == "" {
		appErr.StackTrace = captureStackTrace(3)
	}

	// Build log attributes
	attrs := []slog.Attr{
		slog.String("severity", "CRITICAL"),
		slog.String("error_code", appErr.Code),
		slog.String("error_category", appErr.Category),
		slog.String("error_message", appErr.Message),
		slog.String("stack_trace", appErr.StackTrace),
	}

	// Add details
	if len(appErr.Details) > 0 {
		for key, value := range appErr.Details {
			attrs = append(attrs, slog.Any(key, value))
		}
	}

	// Add additional arguments
	for i := 0; i < len(args); i += 2 {
		if i+1 < len(args) {
			key := fmt.Sprintf("%v", args[i])
			value := args[i+1]
			attrs = append(attrs, slog.Any(key, value))
		}
	}

	l.logger.LogAttrs(ctx, slog.LevelError, message, attrs...)
}
