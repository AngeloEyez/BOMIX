package logger

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
	"time"
)

// Logger is a wrapper around slog.Logger that supports ring buffer storage
type Logger struct {
	*slog.Logger
	buffer       *Buffer
	eventCb      func(event string, data interface{})
	defaultAttrs []any
	mu           sync.RWMutex
}

// NewLogger creates a new logger with ring buffer support
func NewLogger(bufferCapacity int) *Logger {
	buffer := NewBuffer(bufferCapacity)

	// Create a text handler that writes to stdout
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})

	logger := &Logger{
		Logger: slog.New(handler),
		buffer: buffer,
	}

	return logger
}

// With creates a child Logger with additional attributes
func (l *Logger) With(args ...any) *Logger {
	l.mu.RLock()
	defer l.mu.RUnlock()
	newDefault := append(append([]any{}, l.defaultAttrs...), args...)
	return &Logger{
		Logger:       l.Logger.With(args...),
		buffer:       l.buffer,
		eventCb:      l.eventCb,
		defaultAttrs: newDefault,
	}
}

// SetEventCallback sets the callback function for emitting events
func (l *Logger) SetEventCallback(cb func(event string, data interface{})) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.eventCb = cb
}

// emitEvent emits an event if a callback is set
func (l *Logger) emitEvent(event string, data interface{}) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if l.eventCb != nil {
		l.eventCb(event, data)
	}
}

// addLogEntry adds a log entry to the buffer and emits an event
func (l *Logger) addLogEntry(level string, message string, attrs map[string]string) {
	entry := &LogEntry{
		Level:     level,
		Message:   message,
		Timestamp: time.Now(),
		Attrs:     attrs,
	}
	l.buffer.Add(entry)
	l.emitEvent("log:new", entry)
}

// extractAttrs converts slog key-value pairs to a string map
func extractAttrs(attrs ...any) map[string]string {
	if len(attrs) == 0 {
		return nil
	}
	m := make(map[string]string)
	for i := 0; i < len(attrs)-1; i += 2 {
		if k, ok := attrs[i].(string); ok {
			m[k] = fmt.Sprintf("%v", attrs[i+1])
		}
	}
	return m
}

// Debug logs a debug message
func (l *Logger) Debug(msg string, attrs ...any) {
	l.Logger.Debug(msg, attrs...)
	combined := append(append([]any{}, l.defaultAttrs...), attrs...)
	l.addLogEntry("DEBUG", msg, extractAttrs(combined...))
}

// Info logs an info message
func (l *Logger) Info(msg string, attrs ...any) {
	l.Logger.Info(msg, attrs...)
	combined := append(append([]any{}, l.defaultAttrs...), attrs...)
	l.addLogEntry("INFO", msg, extractAttrs(combined...))
}

// Warn logs a warning message
func (l *Logger) Warn(msg string, attrs ...any) {
	l.Logger.Warn(msg, attrs...)
	combined := append(append([]any{}, l.defaultAttrs...), attrs...)
	l.addLogEntry("WARN", msg, extractAttrs(combined...))
}

// Error logs an error message
func (l *Logger) Error(msg string, attrs ...any) {
	l.Logger.Error(msg, attrs...)
	combined := append(append([]any{}, l.defaultAttrs...), attrs...)
	l.addLogEntry("ERROR", msg, extractAttrs(combined...))
}

// GetLogs returns log entries filtered by level
func (l *Logger) GetLogs(level string, limit int) []*LogEntry {
	return l.buffer.GetEntries(level, limit)
}

// ClearLogs clears all log entries
func (l *Logger) ClearLogs() {
	l.buffer.Clear()
}

// LogEntryJSON is a JSON-serializable version of LogEntry
type LogEntryJSON struct {
	Level     string            `json:"level"`
	Message   string            `json:"message"`
	Timestamp string            `json:"timestamp"`
	Attrs     map[string]string `json:"attrs,omitempty"`
}

// ToJSON converts a LogEntry to a JSON-serializable format
func (e *LogEntry) ToJSON() *LogEntryJSON {
	return &LogEntryJSON{
		Level:     e.Level,
		Message:   e.Message,
		Timestamp: e.Timestamp.Format(time.RFC3339),
		Attrs:     e.Attrs,
	}
}

// Write implements io.Writer for slog compatibility
type logWriter struct{}

func (lw *logWriter) Write(p []byte) (n int, err error) {
	return os.Stdout.Write(p)
}

// NewWriter returns an io.Writer for logging
func NewWriter() io.Writer {
	return &logWriter{}
}
