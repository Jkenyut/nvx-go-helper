// Package logger provides a centralized, configuration-driven structured logging
// setup using zerolog and high-performance diode ring-buffer for microservices.
// It automatically adapts to the runtime environment:
//
//   - production/prod  → JSON output to stdout (ideal for Loki, ELK, CloudWatch)
//   - development/staging/other → Pretty colored console output to stderr
//
// Features:
//   - Environment-aware output format
//   - Automatic service name, environment, and port inclusion
//   - LOG_LEVEL environment variable override
//   - OpenTelemetry trace_id and span_id context propagation
//   - High-performance asynchronous ring-buffer writing via diode
//   - Bridge to Go standard library log/slog (*slog.Logger) for driver interoperability
package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/diode"
	"go.opentelemetry.io/otel/trace"
)

var (
	closerMu          sync.Mutex
	globalDiodeCloser io.Closer
	initOnce          sync.Once
)

// Config holds the logger configuration parameters.
type Config struct {
	NameService string `json:"name_service" yaml:"name_service"`
	Env         string `json:"env" yaml:"env"`
	Port        int    `json:"port" yaml:"port"`
}

// Environment returns the normalized lowercase environment string.
// Defaults to "development" if empty or whitespace-only.
func (c Config) Environment() string {
	if env := strings.ToLower(strings.TrimSpace(c.Env)); env != "" {
		return env
	}
	return "development"
}

// ServiceName returns the trimmed name of the service.
// Defaults to "unknown-service" if empty or whitespace-only.
func (c Config) ServiceName() string {
	if name := strings.TrimSpace(c.NameService); name != "" {
		return name
	}
	return "unknown-service"
}

// ConfigProvider represents any configuration struct capable of providing
// service name and environment (e.g. Config or driver Listener config).
type ConfigProvider interface {
	ServiceName() string
	Environment() string
}

// InitFromConfig initializes the global zerolog logger using values from the
// provided configuration provider.
func InitFromConfig(cfg ConfigProvider) {
	env := cfg.Environment()
	serviceName := cfg.ServiceName()
	port := 0
	if c, ok := cfg.(Config); ok {
		port = c.Port
	} else if p, ok := cfg.(interface{ GetPort() int }); ok {
		port = p.GetPort()
	}

	var writer io.Writer
	cleanEnv := strings.ToLower(strings.TrimSpace(env))
	isProd := cleanEnv == "production" || cleanEnv == "prod"

	if isProd {
		writer = io.MultiWriter(os.Stdout)
	} else {
		writer = zerolog.ConsoleWriter{
			Out:     os.Stderr,
			NoColor: false,
		}
	}

	zerolog.TimeFieldFormat = time.RFC3339

	if levelStr := os.Getenv("LOG_LEVEL"); levelStr != "" {
		if level, err := zerolog.ParseLevel(levelStr); err == nil {
			zerolog.SetGlobalLevel(level)
		}
	} else {
		if isProd {
			zerolog.SetGlobalLevel(zerolog.InfoLevel)
		} else {
			zerolog.SetGlobalLevel(zerolog.DebugLevel)
		}
	}

	diodeSize := 1000
	if isProd {
		diodeSize = 10000
	}
	if sizeStr := os.Getenv("LOG_BUFFER_SIZE"); sizeStr != "" {
		if s, err := strconv.Atoi(sizeStr); err == nil && s > 0 {
			diodeSize = s
		}
	}

	wr := diode.NewWriter(writer, diodeSize, 10*time.Millisecond, func(missed int) {
		_, _ = fmt.Fprintf(os.Stderr, "[logger] warning: diode dropped %d log messages\n", missed)
	})

	closerMu.Lock()
	if globalDiodeCloser != nil {
		_ = globalDiodeCloser.Close()
	}
	globalDiodeCloser = wr
	closerMu.Unlock()

	logContext := zerolog.New(wr).
		With().
		Timestamp().
		Str("service", serviceName).
		Str("env", env)

	if port > 0 {
		logContext = logContext.Int("port", port)
	}

	if !isProd {
		logContext = logContext.Caller()
	}

	log := logContext.Logger()
	zerolog.DefaultContextLogger = &log
}

// Close flushes all remaining logs in the buffer and stops the background
// logging goroutine. This should be called during graceful shutdown.
func Close() error {
	closerMu.Lock()
	defer closerMu.Unlock()
	if globalDiodeCloser != nil {
		err := globalDiodeCloser.Close()
		globalDiodeCloser = nil
		return err
	}
	return nil
}

// L returns the global zerolog logger instance.
func L() *zerolog.Logger {
	initOnce.Do(func() {
		if zerolog.DefaultContextLogger == nil {
			InitFromConfig(Config{})
		}
	})
	return zerolog.DefaultContextLogger
}

// Ctx returns a logger instance with OpenTelemetry trace_id and span_id from context.
func Ctx(ctx context.Context) *zerolog.Logger {
	l := L().With().Logger()
	if ctx != nil {
		spanCtx := trace.SpanContextFromContext(ctx)
		if spanCtx.HasTraceID() {
			l = l.With().Str("trace_id", spanCtx.TraceID().String()).Logger()
		}
		if spanCtx.HasSpanID() {
			l = l.With().Str("span_id", spanCtx.SpanID().String()).Logger()
		}
	}
	return &l
}

// Slog returns an stdlib *slog.Logger backed by the global Zerolog instance.
// Pass this directly to nvx-go-driver clients (e.g. postgres, redis, rabbitmq).
func Slog() *slog.Logger {
	return slog.New(&zerologSlogHandler{logger: L()})
}

// SlogWithContext returns an stdlib *slog.Logger with OpenTelemetry trace_id and span_id injected.
func SlogWithContext(ctx context.Context) *slog.Logger {
	return slog.New(&zerologSlogHandler{logger: Ctx(ctx)})
}

// Debug returns a zerolog.Event for debug-level logging with context.
func Debug(ctx context.Context) *zerolog.Event { return Ctx(ctx).Debug() }

// Info returns a zerolog.Event for info-level logging with context.
func Info(ctx context.Context) *zerolog.Event { return Ctx(ctx).Info() }

// Warn returns a zerolog.Event for warning-level logging with context.
func Warn(ctx context.Context) *zerolog.Event { return Ctx(ctx).Warn() }

// Error returns a zerolog.Event for error-level logging with context.
func Error(ctx context.Context) *zerolog.Event { return Ctx(ctx).Error() }

// Fatal returns a zerolog.Event for fatal-level logging with context.
func Fatal(ctx context.Context) *zerolog.Event { return Ctx(ctx).Fatal() }

// Panic returns a zerolog.Event for panic-level logging with context.
func Panic(ctx context.Context) *zerolog.Event { return Ctx(ctx).Panic() }

// zerologSlogHandler adapts zerolog to slog.Handler
type zerologSlogHandler struct {
	logger *zerolog.Logger
}

func (h *zerologSlogHandler) Enabled(_ context.Context, level slog.Level) bool {
	switch level {
	case slog.LevelDebug:
		return h.logger.GetLevel() <= zerolog.DebugLevel
	case slog.LevelInfo:
		return h.logger.GetLevel() <= zerolog.InfoLevel
	case slog.LevelWarn:
		return h.logger.GetLevel() <= zerolog.WarnLevel
	case slog.LevelError:
		return h.logger.GetLevel() <= zerolog.ErrorLevel
	default:
		return true
	}
}

func (h *zerologSlogHandler) Handle(_ context.Context, r slog.Record) error {
	var e *zerolog.Event
	switch r.Level {
	case slog.LevelDebug:
		e = h.logger.Debug()
	case slog.LevelInfo:
		e = h.logger.Info()
	case slog.LevelWarn:
		e = h.logger.Warn()
	case slog.LevelError:
		e = h.logger.Error()
	default:
		e = h.logger.Info()
	}

	r.Attrs(func(a slog.Attr) bool {
		e = e.Any(a.Key, a.Value.Any())
		return true
	})

	e.Msg(r.Message)
	return nil
}

func (h *zerologSlogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	sub := h.logger.With()
	for _, a := range attrs {
		sub = sub.Any(a.Key, a.Value.Any())
	}
	l := sub.Logger()
	return &zerologSlogHandler{logger: &l}
}

func (h *zerologSlogHandler) WithGroup(_ string) slog.Handler {
	return h
}
