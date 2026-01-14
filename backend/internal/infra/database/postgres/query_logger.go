package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/tracelog"
	"github.com/rs/zerolog"
)

// QueryLogger implements pgx.QueryTracer for logging database queries
type QueryLogger struct {
	logger zerolog.Logger
}

// NewQueryLogger creates a new query logger
func NewQueryLogger(logger zerolog.Logger) *QueryLogger {
	return &QueryLogger{
		logger: logger,
	}
}

// TraceQueryStart is called at the beginning of Query, QueryRow, and Exec calls
func (ql *QueryLogger) TraceQueryStart(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	// Store start time in context
	return context.WithValue(ctx, "query_start", time.Now())
}

// TraceQueryEnd is called at the end of Query, QueryRow, and Exec calls
func (ql *QueryLogger) TraceQueryEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryEndData) {
	// Get start time from context
	start, ok := ctx.Value("query_start").(time.Time)
	if !ok {
		start = time.Now()
	}

	duration := time.Since(start)

	// Get request ID from context if available
	requestID := ""
	if rid := ctx.Value("request_id"); rid != nil {
		if id, ok := rid.(string); ok {
			requestID = id
		}
	}

	event := ql.logger.Debug()

	// Add request ID if available
	if requestID != "" {
		event = event.Str("request_id", requestID)
	}

	event = event.
		Str("sql", data.SQL).
		Int64("duration_ms", duration.Milliseconds()).
		Str("command_tag", data.CommandTag.String())

	// Log error if exists
	if data.Err != nil {
		event = ql.logger.Error().
			Str("sql", data.SQL).
			Err(data.Err)
	}

	// Warn on slow queries (> 100ms)
	if duration > 100*time.Millisecond {
		event = ql.logger.Warn().
			Str("sql", data.SQL).
			Int64("duration_ms", duration.Milliseconds()).
			Str("command_tag", data.CommandTag.String())
		event.Msg("⚠️  Slow query detected")
		return
	}

	event.Msg("Query executed")
}

// PgxZerologAdapter adapts zerolog.Logger to pgx's Logger interface
type PgxZerologAdapter struct {
	logger zerolog.Logger
}

// NewPgxZerologAdapter creates a new adapter
func NewPgxZerologAdapter(logger zerolog.Logger) *PgxZerologAdapter {
	return &PgxZerologAdapter{logger: logger}
}

// Log implements pgx Logger interface
func (l *PgxZerologAdapter) Log(ctx context.Context, level tracelog.LogLevel, msg string, data map[string]interface{}) {
	var event *zerolog.Event

	switch level {
	case tracelog.LogLevelTrace:
		event = l.logger.Trace()
	case tracelog.LogLevelDebug:
		event = l.logger.Debug()
	case tracelog.LogLevelInfo:
		event = l.logger.Info()
	case tracelog.LogLevelWarn:
		event = l.logger.Warn()
	case tracelog.LogLevelError:
		event = l.logger.Error()
	default:
		event = l.logger.Info()
	}

	// Add all data fields
	for key, value := range data {
		event = event.Interface(key, value)
	}

	event.Msg(msg)
}
