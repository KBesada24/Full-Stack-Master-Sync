package utils

import (
	"encoding/json"
	"fmt"
	"log"
	"runtime"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

// LogLevel represents the severity level of a log entry
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
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
	default:
		return "UNKNOWN"
	}
}

// LogEntry represents a structured log entry
type LogEntry struct {
	Timestamp time.Time              `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	TraceID   string                 `json:"trace_id,omitempty"`
	Source    string                 `json:"source,omitempty"`
	Context   map[string]interface{} `json:"context,omitempty"`
	Error     string                 `json:"error,omitempty"`
	File      string                 `json:"file,omitempty"`
	Line      int                    `json:"line,omitempty"`
}

// Logger represents a structured logger
type Logger struct {
	level  LogLevel
	format string // "json" or "text"
}

// NewLogger creates a new logger instance
func NewLogger(level, format string) *Logger {
	logLevel := parseLogLevel(level)
	if format != "json" && format != "text" {
		format = "json"
	}

	return &Logger{
		level:  logLevel,
		format: format,
	}
}

// parseLogLevel parses string log level to LogLevel enum
func parseLogLevel(level string) LogLevel {
	switch strings.ToUpper(level) {
	case "DEBUG":
		return DEBUG
	case "INFO":
		return INFO
	case "WARN", "WARNING":
		return WARN
	case "ERROR":
		return ERROR
	default:
		return INFO
	}
}

// Debug logs a debug message
func (l *Logger) Debug(message string, context ...map[string]interface{}) {
	l.log(DEBUG, message, "", context...)
}

// Info logs an info message
func (l *Logger) Info(message string, context ...map[string]interface{}) {
	l.log(INFO, message, "", context...)
}

// Warn logs a warning message
func (l *Logger) Warn(message string, context ...map[string]interface{}) {
	l.log(WARN, message, "", context...)
}

// Error logs an error message
func (l *Logger) Error(message string, err error, context ...map[string]interface{}) {
	errorMsg := ""
	if err != nil {
		errorMsg = err.Error()
	}
	l.log(ERROR, message, errorMsg, context...)
}

// log performs the actual logging
func (l *Logger) log(level LogLevel, message, errorMsg string, context ...map[string]interface{}) {
	// Skip if log level is below configured level
	if level < l.level {
		return
	}

	// Get caller information
	_, file, line, ok := runtime.Caller(2)
	if ok {
		// Extract just the filename from the full path
		parts := strings.Split(file, "/")
		if len(parts) > 0 {
			file = parts[len(parts)-1]
		}
	}

	// Merge context maps
	mergedContext := make(map[string]interface{})
	for _, ctx := range context {
		for k, v := range ctx {
			mergedContext[k] = v
		}
	}

	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     level.String(),
		Message:   message,
		Context:   mergedContext,
		Error:     errorMsg,
		File:      file,
		Line:      line,
	}

	l.output(entry)
}

// output writes the log entry to stdout
func (l *Logger) output(entry LogEntry) {
	if l.format == "json" {
		l.outputJSON(entry)
	} else {
		l.outputText(entry)
	}
}

// outputJSON outputs log entry in JSON format
func (l *Logger) outputJSON(entry LogEntry) {
	jsonData, err := json.Marshal(entry)
	if err != nil {
		log.Printf("Error marshaling log entry: %v", err)
		return
	}
	fmt.Println(string(jsonData))
}

// outputText outputs log entry in human-readable text format
func (l *Logger) outputText(entry LogEntry) {
	timestamp := entry.Timestamp.Format("2006-01-02 15:04:05")

	var output strings.Builder
	output.WriteString(fmt.Sprintf("[%s] %s: %s", timestamp, entry.Level, entry.Message))

	if entry.TraceID != "" {
		output.WriteString(fmt.Sprintf(" [trace_id=%s]", entry.TraceID))
	}

	if entry.Source != "" {
		output.WriteString(fmt.Sprintf(" [source=%s]", entry.Source))
	}

	if entry.File != "" && entry.Line > 0 {
		output.WriteString(fmt.Sprintf(" [%s:%d]", entry.File, entry.Line))
	}

	if entry.Error != "" {
		output.WriteString(fmt.Sprintf(" [error=%s]", entry.Error))
	}

	if len(entry.Context) > 0 {
		contextStr, _ := json.Marshal(entry.Context)
		output.WriteString(fmt.Sprintf(" [context=%s]", string(contextStr)))
	}

	fmt.Println(output.String())
}

// WithTraceID adds trace ID to log entry
func (l *Logger) WithTraceID(traceID string) *LoggerWithContext {
	return &LoggerWithContext{
		logger:  l,
		traceID: traceID,
	}
}

// WithSource adds source information to log entry
func (l *Logger) WithSource(source string) *LoggerWithContext {
	return &LoggerWithContext{
		logger: l,
		source: source,
	}
}

// WithSource adds source information to log entry (for LoggerWithContext)
func (lwc *LoggerWithContext) WithSource(source string) *LoggerWithContext {
	return &LoggerWithContext{
		logger:  lwc.logger,
		traceID: lwc.traceID,
		source:  source,
		context: lwc.context,
	}
}

// WithTraceID adds trace ID to log entry (for LoggerWithContext)
func (lwc *LoggerWithContext) WithTraceID(traceID string) *LoggerWithContext {
	return &LoggerWithContext{
		logger:  lwc.logger,
		traceID: traceID,
		source:  lwc.source,
		context: lwc.context,
	}
}

// LoggerWithContext represents a logger with additional context
type LoggerWithContext struct {
	logger  *Logger
	traceID string
	source  string
	context map[string]interface{}
}

// WithContext adds context to the logger
func (lwc *LoggerWithContext) WithContext(context map[string]interface{}) *LoggerWithContext {
	newContext := make(map[string]interface{})

	// Copy existing context
	for k, v := range lwc.context {
		newContext[k] = v
	}

	// Add new context
	for k, v := range context {
		newContext[k] = v
	}

	return &LoggerWithContext{
		logger:  lwc.logger,
		traceID: lwc.traceID,
		source:  lwc.source,
		context: newContext,
	}
}

// Debug logs a debug message with context
func (lwc *LoggerWithContext) Debug(message string, context ...map[string]interface{}) {
	lwc.logWithContext(DEBUG, message, "", context...)
}

// Info logs an info message with context
func (lwc *LoggerWithContext) Info(message string, context ...map[string]interface{}) {
	lwc.logWithContext(INFO, message, "", context...)
}

// Warn logs a warning message with context
func (lwc *LoggerWithContext) Warn(message string, context ...map[string]interface{}) {
	lwc.logWithContext(WARN, message, "", context...)
}

// Error logs an error message with context
func (lwc *LoggerWithContext) Error(message string, err error, context ...map[string]interface{}) {
	errorMsg := ""
	if err != nil {
		errorMsg = err.Error()
	}
	lwc.logWithContext(ERROR, message, errorMsg, context...)
}

// logWithContext performs logging with additional context
func (lwc *LoggerWithContext) logWithContext(level LogLevel, message, errorMsg string, context ...map[string]interface{}) {
	// Skip if log level is below configured level
	if level < lwc.logger.level {
		return
	}

	// Get caller information
	_, file, line, ok := runtime.Caller(2)
	if ok {
		parts := strings.Split(file, "/")
		if len(parts) > 0 {
			file = parts[len(parts)-1]
		}
	}

	// Merge all context
	mergedContext := make(map[string]interface{})

	// Add logger context
	for k, v := range lwc.context {
		mergedContext[k] = v
	}

	// Add method context
	for _, ctx := range context {
		for k, v := range ctx {
			mergedContext[k] = v
		}
	}

	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     level.String(),
		Message:   message,
		TraceID:   lwc.traceID,
		Source:    lwc.source,
		Context:   mergedContext,
		Error:     errorMsg,
		File:      file,
		Line:      line,
	}

	lwc.logger.output(entry)
}

// Global logger instance
var globalLogger *Logger

// InitLogger initializes the global logger
func InitLogger(level, format string) {
	globalLogger = NewLogger(level, format)
}

// GetLogger returns the global logger instance
func GetLogger() *Logger {
	if globalLogger == nil {
		globalLogger = NewLogger("info", "json")
	}
	return globalLogger
}

// LogRequest logs HTTP request information
func LogRequest(c *fiber.Ctx, logger *Logger) {
	traceID := GetTraceID(c)

	context := map[string]interface{}{
		"method":     c.Method(),
		"path":       c.Path(),
		"ip":         c.IP(),
		"user_agent": c.Get("User-Agent"),
	}

	logger.WithTraceID(traceID).WithSource("http").Info("Request received", context)
}

// LogResponse logs HTTP response information
func LogResponse(c *fiber.Ctx, logger *Logger, statusCode int, duration time.Duration) {
	traceID := GetTraceID(c)

	context := map[string]interface{}{
		"method":      c.Method(),
		"path":        c.Path(),
		"status_code": statusCode,
		"duration_ms": duration.Milliseconds(),
	}

	logLevel := "info"
	if statusCode >= 400 && statusCode < 500 {
		logLevel = "warn"
	} else if statusCode >= 500 {
		logLevel = "error"
	}

	loggerWithContext := logger.WithTraceID(traceID).WithSource("http")

	switch logLevel {
	case "warn":
		loggerWithContext.Warn("Request completed", context)
	case "error":
		loggerWithContext.Error("Request failed", nil, context)
	default:
		loggerWithContext.Info("Request completed", context)
	}
}
