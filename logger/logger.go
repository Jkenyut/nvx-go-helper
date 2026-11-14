// Package logger provides a centralized, configuration-driven structured logging
// setup using zerolog and high-performance diode ring-buffer for microservices.
// All environments emit structured JSON lines to stdout (ideal for Loki, ELK,
// CloudWatch, Datadog, and container runtimes), while adapting runtime defaults:
//
//   - production/prod           → info level, 10000 buffer size
//   - development/staging/other → debug level, 1000 buffer size, caller enabled
//
// Features:
//   - Consistent JSON output across all environments (12-Factor App)
//   - Automatic service name, environment, and port inclusion
//   - LOG_LEVEL environment variable override
//   - OpenTelemetry trace_id and span_id context propagation
//   - High-performance asynchronous ring-buffer writing via diode
//   - Bridge to Go standard library log/slog (*slog.Logger) for driver interoperability
package logger

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Jkenyut/nvx-go-helper/activity"
	"github.com/bytedance/sonic"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/diode"
	"go.opentelemetry.io/otel/trace"
)

var (
	closerMu          sync.Mutex
	globalDiodeCloser io.Closer
	globalLogger      atomic.Pointer[zerolog.Logger]
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

type options struct {
	serviceName string
	env         string
	port        int
	level       string
	bufferSize  int
	writer      io.Writer
}

// Option configures logger behavior.
type Option func(*options)

// WithServiceName sets the service name attribute included in every log entry.
func WithServiceName(name string) Option {
	return func(o *options) {
		o.serviceName = strings.TrimSpace(name)
	}
}

// WithEnv sets the environment name (e.g., "production", "development", "staging").
func WithEnv(env string) Option {
	return func(o *options) {
		o.env = strings.TrimSpace(env)
	}
}

// WithPort sets the application port attribute included in log entries.
func WithPort(port int) Option {
	return func(o *options) {
		o.port = port
	}
}

// WithLevel sets a custom log level (e.g., "debug", "info", "warn", "error").
func WithLevel(level string) Option {
	return func(o *options) {
		o.level = strings.TrimSpace(level)
	}
}

// WithBufferSize overrides the diode asynchronous ring-buffer size.
func WithBufferSize(size int) Option {
	return func(o *options) {
		if size > 0 {
			o.bufferSize = size
		}
	}
}

// WithWriter sets a custom underlying io.Writer (e.g., for testing or custom log sinks).
func WithWriter(w io.Writer) Option {
	return func(o *options) {
		o.writer = w
	}
}

// ActivityHook extracts contextual metadata from activity context
// (such as request_id, correlation_id, user_id, user_ip, ip_origin, and custom metadata)
// and OpenTelemetry trace/span IDs, automatically attaching them to every log event.
type ActivityHook struct{}

// Run executes the hook for each log event, attaching activity and tracing context.
func (h ActivityHook) Run(e *zerolog.Event, _ zerolog.Level, _ string) {
	ctx := e.GetCtx()
	if ctx == nil {
		return
	}

	// 1. OpenTelemetry trace and span IDs
	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.HasTraceID() {
		e.Str("trace_id", spanCtx.TraceID().String())
	}
	if spanCtx.HasSpanID() {
		e.Str("span_id", spanCtx.SpanID().String())
	}

	// 2. Standard activity context fields
	act := activity.FromContext(ctx)
	if act.RequestID != "" {
		e.Str("request_id", act.RequestID)
	}
	if act.CorrelationID != "" {
		e.Str("correlation_id", act.CorrelationID)
	}
	if act.UserID != "" {
		e.Str("user_id", act.UserID)
	}
	if act.UserIP != "" {
		e.Str("user_ip", act.UserIP)
	}
	if act.UserIPOrigin != "" {
		e.Str("user_ip_origin", act.UserIPOrigin)
	}

	// 3. Custom metadata map attached via activity.WithMetadata
	if meta, ok := activity.GetMetadata(ctx); ok && len(meta) > 0 {
		for k, v := range meta {
			e.Interface(k, v)
		}
	}
}

// Init initializes the global zerolog logger using functional options.
// If called without options, it defaults to development mode with JSON output to stdout.
func Init(opts ...Option) {
	o := options{
		serviceName: "unknown-service",
		env:         "development",
		port:        0,
		bufferSize:  1000,
	}
	for _, opt := range opts {
		opt(&o)
	}

	cleanEnv := strings.ToLower(strings.TrimSpace(o.env))
	if cleanEnv == "" {
		cleanEnv = "development"
	}
	isProd := cleanEnv == "production" || cleanEnv == "prod"
	if isProd && o.bufferSize == 1000 {
		o.bufferSize = 10000
	}

	writer := o.writer
	if writer == nil {
		writer = os.Stdout
	}

	if sizeStr := os.Getenv("LOG_BUFFER_SIZE"); sizeStr != "" {
		if s, err := strconv.Atoi(sizeStr); err == nil && s > 0 {
			o.bufferSize = s
		}
	}

	wr := diode.NewWriter(writer, o.bufferSize, 10*time.Millisecond, func(missed int) {
		_, _ = fmt.Fprintf(os.Stderr, "[logger] warning: diode dropped %d log messages\n", missed)
	})

	closerMu.Lock()
	zerolog.InterfaceMarshalFunc = sonic.Marshal
	zerolog.TimeFieldFormat = time.RFC3339

	// 1. Check custom level option first
	if o.level != "" {
		if lvl, err := zerolog.ParseLevel(o.level); err == nil {
			zerolog.SetGlobalLevel(lvl)
		}
	} else if levelStr := os.Getenv("LOG_LEVEL"); levelStr != "" {
		// 2. Environment variable override
		if lvl, err := zerolog.ParseLevel(levelStr); err == nil {
			zerolog.SetGlobalLevel(lvl)
		}
	} else {
		// 3. Fallback to environment default
		if isProd {
			zerolog.SetGlobalLevel(zerolog.InfoLevel)
		} else {
			zerolog.SetGlobalLevel(zerolog.DebugLevel)
		}
	}

	logContext := zerolog.New(wr).
		With().
		Timestamp().
		Str("service", o.serviceName).
		Str("env", cleanEnv)

	if o.port > 0 {
		logContext = logContext.Int("port", o.port)
	}

	if !isProd {
		logContext = logContext.Caller()
	}

	log := logContext.Logger().Hook(ActivityHook{})
	globalLogger.Store(&log)
	zerolog.DefaultContextLogger = &log

	oldCloser := globalDiodeCloser
	globalDiodeCloser = wr
	closerMu.Unlock()

	if oldCloser != nil {
		_ = oldCloser.Close()
	}
}

// InitFromConfig initializes the global zerolog logger using values from the
// provided configuration provider (maintained for backward compatibility with driver configs).
func InitFromConfig(cfg ConfigProvider) {
	env := cfg.Environment()
	serviceName := cfg.ServiceName()
	port := 0
	if c, ok := cfg.(Config); ok {
		port = c.Port
	} else if p, ok := cfg.(interface{ GetPort() int }); ok {
		port = p.GetPort()
	}

	Init(
		WithServiceName(serviceName),
		WithEnv(env),
		WithPort(port),
	)
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
	if l := globalLogger.Load(); l != nil {
		return l
	}
	initOnce.Do(func() {
		if globalLogger.Load() == nil {
			InitFromConfig(Config{})
		}
	})
	if l := globalLogger.Load(); l != nil {
		return l
	}
	InitFromConfig(Config{})
	return globalLogger.Load()
}

// Ctx returns a logger instance bound to the provided context.
// Any log event created from this logger (or with .Ctx(ctx)) will automatically
// execute the ActivityHook, extracting all activity metadata and OpenTelemetry spans.
func Ctx(ctx context.Context) *zerolog.Logger {
	if ctx == nil {
		return L()
	}
	l := L().With().Ctx(ctx).Logger()
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

func withCtx(e *zerolog.Event, ctx context.Context) *zerolog.Event {
	if e == nil || ctx == nil {
		return e
	}
	return e.Ctx(ctx)
}

// Debug returns a zerolog.Event for debug-level logging with context.
func Debug(ctx context.Context) *zerolog.Event { return withCtx(L().Debug(), ctx) }

// Info returns a zerolog.Event for info-level logging with context.
func Info(ctx context.Context) *zerolog.Event { return withCtx(L().Info(), ctx) }

// Warn returns a zerolog.Event for warning-level logging with context.
func Warn(ctx context.Context) *zerolog.Event { return withCtx(L().Warn(), ctx) }

// Error returns a zerolog.Event for error-level logging with context.
func Error(ctx context.Context) *zerolog.Event { return withCtx(L().Error(), ctx) }

// Fatal returns a zerolog.Event for fatal-level logging with context.
func Fatal(ctx context.Context) *zerolog.Event { return withCtx(L().Fatal(), ctx) }

// Panic returns a zerolog.Event for panic-level logging with context.
func Panic(ctx context.Context) *zerolog.Event { return withCtx(L().Panic(), ctx) }

// zerologSlogHandler adapts zerolog to slog.Handler
type zerologSlogHandler struct {
	logger *zerolog.Logger
	groups []string
}

func (h *zerologSlogHandler) resolveKey(key string) string {
	if len(h.groups) == 0 {
		return key
	}
	return strings.Join(h.groups, ".") + "." + key
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

func (h *zerologSlogHandler) Handle(ctx context.Context, r slog.Record) error {
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

	if ctx != nil {
		e = e.Ctx(ctx)
	}

	r.Attrs(func(a slog.Attr) bool {
		val := a.Value.Resolve().Any()
		key := h.resolveKey(a.Key)
		switch v := val.(type) {
		case error:
			e = e.AnErr(key, v)
		case json.RawMessage:
			e = e.RawJSON(key, v)
		case []byte:
			if sonic.Valid(v) {
				e = e.RawJSON(key, v)
			} else {
				e = e.Bytes(key, v)
			}
		default:
			e = e.Any(key, val)
		}
		return true
	})

	e.Msg(r.Message)
	return nil
}

func (h *zerologSlogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	sub := h.logger.With()
	for _, a := range attrs {
		val := a.Value.Resolve().Any()
		key := h.resolveKey(a.Key)
		switch v := val.(type) {
		case error:
			sub = sub.AnErr(key, v)
		case json.RawMessage:
			sub = sub.RawJSON(key, v)
		case []byte:
			if sonic.Valid(v) {
				sub = sub.RawJSON(key, v)
			} else {
				sub = sub.Bytes(key, v)
			}
		default:
			sub = sub.Any(key, val)
		}
	}
	l := sub.Logger()
	return &zerologSlogHandler{logger: &l, groups: h.groups}
}

func (h *zerologSlogHandler) WithGroup(name string) slog.Handler {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return h
	}
	newGroups := make([]string, len(h.groups)+1)
	copy(newGroups, h.groups)
	newGroups[len(h.groups)] = trimmed
	return &zerologSlogHandler{
		logger: h.logger,
		groups: newGroups,
	}
}
