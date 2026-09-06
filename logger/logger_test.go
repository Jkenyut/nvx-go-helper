package logger

import (
	"context"
	"log/slog"
	"sync"
	"testing"

	"bytes"
	"github.com/Jkenyut/nvx-go-helper/activity"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/trace"
	"strings"
)

func TestInitFromConfig_Environments(t *testing.T) {
	tests := []struct {
		name      string
		cfg       Config
		wantLevel zerolog.Level
	}{
		{
			name: "development defaults to debug",
			cfg: Config{
				Env:         "development",
				NameService: "dev-svc",
				Port:        8080,
			},
			wantLevel: zerolog.DebugLevel,
		},
		{
			name: "production defaults to info",
			cfg: Config{
				Env:         "production",
				NameService: "prod-svc",
				Port:        8080,
			},
			wantLevel: zerolog.InfoLevel,
		},
		{
			name: "prod alias defaults to info",
			cfg: Config{
				Env:         "prod",
				NameService: "prod-svc",
				Port:        8080,
			},
			wantLevel: zerolog.InfoLevel,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			InitFromConfig(tt.cfg)
			defer func() { _ = Close() }()

			if zerolog.GlobalLevel() != tt.wantLevel {
				t.Errorf("GlobalLevel = %v, want %v", zerolog.GlobalLevel(), tt.wantLevel)
			}

			l := L()
			if l == nil {
				t.Fatal("L() returned nil logger")
			}
		})
	}
}

func TestLogger_ConcurrentAccess(t *testing.T) {
	InitFromConfig(Config{
		Env:         "development",
		NameService: "test-svc",
		Port:        8081,
	})
	defer func() { _ = Close() }()

	var wg sync.WaitGroup
	workers := 10
	iterations := 100

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				l := L()
				if l == nil {
					t.Errorf("worker %d got nil logger", workerID)
					return
				}
				Info(context.Background()).Int("worker", workerID).Int("iter", j).Msg("concurrent log test")
			}
		}(i)
	}

	wg.Wait()
}

func TestCtx_WithAndWithoutSpan(t *testing.T) {
	InitFromConfig(Config{
		Env:         "development",
		NameService: "test-span",
		Port:        8082,
	})
	defer func() { _ = Close() }()

	t.Run("nil context does not panic", func(t *testing.T) {
		l := Ctx(nil)
		if l == nil {
			t.Fatal("Ctx(nil) returned nil logger")
		}
	})

	t.Run("empty context", func(t *testing.T) {
		ctx := context.Background()
		l := Ctx(ctx)
		if l == nil {
			t.Fatal("Ctx returned nil logger")
		}
	})

	t.Run("context with trace and span IDs", func(t *testing.T) {
		traceID, err := trace.TraceIDFromHex("4bf92f3577b34da6a3ce929d0e0e4736")
		if err != nil {
			t.Fatalf("failed to parse traceID: %v", err)
		}
		spanID, err := trace.SpanIDFromHex("00f067aa0ba902b7")
		if err != nil {
			t.Fatalf("failed to parse spanID: %v", err)
		}

		spanCtx := trace.NewSpanContext(trace.SpanContextConfig{
			TraceID:    traceID,
			SpanID:     spanID,
			TraceFlags: trace.FlagsSampled,
		})
		ctx := trace.ContextWithSpanContext(context.Background(), spanCtx)

		l := Ctx(ctx)
		if l == nil {
			t.Fatal("Ctx returned nil logger")
		}
	})
}

func TestConvenienceFunctions(t *testing.T) {
	InitFromConfig(Config{
		Env:         "development",
		NameService: "test-levels",
		Port:        8083,
	})
	defer func() { _ = Close() }()

	ctx := context.Background()

	Debug(ctx).Msg("debug log")
	Info(ctx).Msg("info log")
	Warn(ctx).Msg("warn log")
	Error(ctx).Msg("error log")
}

func TestSlogBridge(t *testing.T) {
	InitFromConfig(Config{
		Env:         "development",
		NameService: "test-slog",
		Port:        8084,
	})
	defer func() { _ = Close() }()

	slogLogger := Slog()
	if slogLogger == nil {
		t.Fatal("Slog() returned nil")
	}

	slogLogger.Info("test info via slog bridge", "key", "value")
	slogLogger.Warn("test warn via slog bridge", slog.Int("count", 42))

	ctxLogger := SlogWithContext(context.Background())
	if ctxLogger == nil {
		t.Fatal("SlogWithContext() returned nil")
	}
	ctxLogger.Debug("test debug with ctx", "active", true)
}

func TestActivityHook(t *testing.T) {
	var buf bytes.Buffer
	log := zerolog.New(&buf).Hook(ActivityHook{})

	ctx := context.Background()
	ctx = activity.WithRequestID(ctx, "req-xyz-123")
	ctx = activity.WithTransactionID(ctx, "trx-abc-456")
	ctx = activity.WithUserID(ctx, "user-999")
	ctx = activity.WithUserIP(ctx, "192.168.1.50")
	ctx = activity.WithUserIPOrigin(ctx, "203.0.113.195")
	ctx = activity.WithMetadata(ctx, map[string]any{"custom_tenant": "tenant-alpha"})

	t.Run("zerolog with Ctx context", func(t *testing.T) {
		buf.Reset()
		log.Info().Ctx(ctx).Msg("test activity hook message")

		out := buf.String()
		for _, expected := range []string{
			"\"request_id\":\"req-xyz-123\"",
			"\"transaction_id\":\"trx-abc-456\"",
			"\"user_id\":\"user-999\"",
			"\"user_ip\":\"192.168.1.50\"",
			"\"user_ip_origin\":\"203.0.113.195\"",
			"\"custom_tenant\":\"tenant-alpha\"",
			"test activity hook message",
		} {
			if !strings.Contains(out, expected) {
				t.Errorf("expected output to contain %q, got: %s", expected, out)
			}
		}
	})

	t.Run("slog bridge with context", func(t *testing.T) {
		buf.Reset()
		slogHandler := &zerologSlogHandler{logger: &log}
		slogger := slog.New(slogHandler)
		slogger.InfoContext(ctx, "test slog activity message")

		out := buf.String()
		for _, expected := range []string{
			"\"request_id\":\"req-xyz-123\"",
			"\"transaction_id\":\"trx-abc-456\"",
			"\"user_id\":\"user-999\"",
			"\"user_ip\":\"192.168.1.50\"",
			"\"user_ip_origin\":\"203.0.113.195\"",
			"\"custom_tenant\":\"tenant-alpha\"",
			"test slog activity message",
		} {
			if !strings.Contains(out, expected) {
				t.Errorf("expected slog output to contain %q, got: %s", expected, out)
			}
		}
	})
}
