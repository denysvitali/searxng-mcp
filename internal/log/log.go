package log

import (
	"os"

	"github.com/sirupsen/logrus"
)

var logger *logrus.Logger

// Init initializes the global logger with the specified level
func Init(level string) {
	logger = logrus.New()
	logger.SetOutput(os.Stdout)
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})

	switch level {
	case "debug":
		logger.SetLevel(logrus.DebugLevel)
	case "info":
		logger.SetLevel(logrus.InfoLevel)
	case "warn":
		logger.SetLevel(logrus.WarnLevel)
	case "error":
		logger.SetLevel(logrus.ErrorLevel)
	default:
		logger.SetLevel(logrus.InfoLevel)
	}
}

// Get returns the global logger instance
func Get() *logrus.Logger {
	if logger == nil {
		Init("info")
	}
	return logger
}

// WithField returns a logger entry with a single field
func WithField(key string, value interface{}) *logrus.Entry {
	return Get().WithField(key, value)
}

// WithFields returns a logger entry with multiple fields
func WithFields(fields logrus.Fields) *logrus.Entry {
	return Get().WithFields(fields)
}

// Debug logs a message at DebugLevel
func Debug(args ...interface{}) {
	Get().Debug(args...)
}

// Info logs a message at InfoLevel
func Info(args ...interface{}) {
	Get().Info(args...)
}

// Warn logs a message at WarnLevel
func Warn(args ...interface{}) {
	Get().Warn(args...)
}

// Error logs a message at ErrorLevel
func Error(args ...interface{}) {
	Get().Error(args...)
}

// Fatal logs a message at FatalLevel and exits
func Fatal(args ...interface{}) {
	Get().Fatal(args...)
}
