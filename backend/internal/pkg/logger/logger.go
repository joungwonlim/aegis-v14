package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Config holds logger configuration
type Config struct {
	Level          string // debug, info, warn, error
	Format         string // json, pretty
	FileEnabled    bool
	FilePath       string // logs directory path
	RotationSize   int    // MB
	RetentionDays  int
	ServiceName    string
	ServiceVersion string
}

// Init initializes the global logger
func Init(cfg Config) error {
	// Set log level
	level, err := zerolog.ParseLevel(cfg.Level)
	if err != nil {
		return fmt.Errorf("invalid log level: %w", err)
	}
	zerolog.SetGlobalLevel(level)

	// Configure time format
	zerolog.TimeFieldFormat = time.RFC3339

	// Create writers
	var writers []io.Writer

	// Console writer
	if cfg.Format == "pretty" {
		consoleWriter := zerolog.ConsoleWriter{
			Out:        os.Stderr,
			TimeFormat: "15:04:05",
			NoColor:    false,
		}
		writers = append(writers, consoleWriter)
	} else {
		writers = append(writers, os.Stderr)
	}

	// File writer (if enabled)
	if cfg.FileEnabled {
		if err := os.MkdirAll(cfg.FilePath, 0755); err != nil {
			return fmt.Errorf("failed to create log directory: %w", err)
		}

		// Main app log
		appLogFile := &lumberjack.Logger{
			Filename:   filepath.Join(cfg.FilePath, "app.log"),
			MaxSize:    cfg.RotationSize, // MB
			MaxAge:     cfg.RetentionDays, // days
			MaxBackups: 10,
			Compress:   true,
		}
		writers = append(writers, appLogFile)

		// Error log (ERROR and above only)
		errorLogFile := &lumberjack.Logger{
			Filename:   filepath.Join(cfg.FilePath, "error.log"),
			MaxSize:    cfg.RotationSize,
			MaxAge:     cfg.RetentionDays,
			MaxBackups: 10,
			Compress:   true,
		}
		errorWriter := zerolog.LevelWriterAdapter{
			Writer: errorLogFile,
		}
		writers = append(writers, &errorWriter)
	}

	// Multi writer
	multi := zerolog.MultiLevelWriter(writers...)

	// Create logger with context
	logger := zerolog.New(multi).With().
		Timestamp().
		Str("service", cfg.ServiceName).
		Str("version", cfg.ServiceVersion).
		Logger()

	// Set as global logger
	log.Logger = logger

	// Log initialization
	log.Info().
		Str("level", cfg.Level).
		Str("format", cfg.Format).
		Bool("file_enabled", cfg.FileEnabled).
		Msg("Logger initialized")

	return nil
}

// NewQueryLogger creates a logger for database queries
func NewQueryLogger(logPath string, rotationSize int, retentionDays int) zerolog.Logger {
	if logPath == "" {
		// If file logging disabled, use default logger
		return log.Logger
	}

	// Ensure directory exists
	if err := os.MkdirAll(logPath, 0755); err != nil {
		log.Warn().Err(err).Msg("Failed to create query log directory, using default logger")
		return log.Logger
	}

	queryLogFile := &lumberjack.Logger{
		Filename:   filepath.Join(logPath, "query.log"),
		MaxSize:    rotationSize,
		MaxAge:     retentionDays,
		MaxBackups: 5,
		Compress:   true,
	}

	return zerolog.New(queryLogFile).With().
		Timestamp().
		Str("type", "query").
		Logger()
}

// NewAccessLogger creates a logger for HTTP access logs
func NewAccessLogger(logPath string, rotationSize int, retentionDays int) zerolog.Logger {
	if logPath == "" {
		return log.Logger
	}

	if err := os.MkdirAll(logPath, 0755); err != nil {
		log.Warn().Err(err).Msg("Failed to create access log directory, using default logger")
		return log.Logger
	}

	accessLogFile := &lumberjack.Logger{
		Filename:   filepath.Join(logPath, "access.log"),
		MaxSize:    rotationSize,
		MaxAge:     retentionDays,
		MaxBackups: 10,
		Compress:   true,
	}

	return zerolog.New(accessLogFile).With().
		Timestamp().
		Str("type", "access").
		Logger()
}

// GetLogger returns the global logger
func GetLogger() *zerolog.Logger {
	return &log.Logger
}
