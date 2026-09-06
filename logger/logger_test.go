package logger

import (
	"context"
	"log/slog"
	"sync"
	"testing"

	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/trace"
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
