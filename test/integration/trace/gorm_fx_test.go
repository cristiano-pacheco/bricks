//go:build integration

package trace_test

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/cristiano-pacheco/bricks/pkg/itestkit"
	brickstrace "github.com/cristiano-pacheco/bricks/pkg/otel/trace"
	"github.com/stretchr/testify/require"
	collectortrace "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
	"go.uber.org/fx"
	"google.golang.org/protobuf/proto"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type traceReceiver struct {
	mu       sync.Mutex
	requests []*collectortrace.ExportTraceServiceRequest
}

func (r *traceReceiver) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	data, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var message collectortrace.ExportTraceServiceRequest
	if err = proto.Unmarshal(data, &message); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	r.mu.Lock()
	r.requests = append(r.requests, &message)
	r.mu.Unlock()
	w.Header().Set("Content-Type", "application/x-protobuf")
	w.WriteHeader(http.StatusOK)
}

func (r *traceReceiver) spans() []*tracepb.Span {
	r.mu.Lock()
	defer r.mu.Unlock()

	var spans []*tracepb.Span
	for _, request := range r.requests {
		for _, resource := range request.GetResourceSpans() {
			for _, scope := range resource.GetScopeSpans() {
				spans = append(spans, scope.GetSpans()...)
			}
		}
	}
	return spans
}

func configureTracing(t *testing.T, endpoint, settings string) {
	t.Helper()
	dir := t.TempDir()
	yaml := fmt.Sprintf(`app:
  open-telemetry:
    app_name: gorm-integration
    trace_url: %q
    exporter_type: http
    insecure: true
    batch_timeout: 1h
    max_batch_size: 8
    max_queue_size: 16
    export_timeout: 200ms
    shutdown_timeout: 1s
%s
`, endpoint, settings)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "base.yaml"), []byte(yaml), 0600))
	t.Setenv("APP_CONFIG_DIR", dir)
	t.Setenv("APP_ENV", "local")
}

// queryService represents a consumer's public operation using its own GORM client.
type queryService struct {
	db *gorm.DB
}

type existingGORMPlugin struct{}

func (existingGORMPlugin) Name() string {
	return "otelgorm"
}

func (existingGORMPlugin) Initialize(*gorm.DB) error {
	return nil
}

func (s *queryService) Execute(ctx context.Context, value string) (string, error) {
	var result string
	err := s.db.WithContext(ctx).Raw("SELECT ?::text", value).Scan(&result).Error
	return result, err
}

//nolint:paralleltest // Tracing and configuration use process-global state.
func TestGORMTracing_PublicService(t *testing.T) {
	kit := itestkit.New(itestkit.DefaultConfig())
	require.NoError(t, kit.StartPostgres())
	t.Cleanup(kit.StopPostgres)

	sqlDB, err := kit.DB().DB()
	require.NoError(t, err)

	for _, tc := range []struct {
		name      string
		settings  string
		wantQuery bool
	}{
		{name: "default-disabled"},
		{name: "database-without-tracing", settings: "  database: {}\n"},
		{name: "tracing-without-enable", settings: "  database:\n    tracing: {}\n"},
		{name: "metrics-only", settings: "  database:\n    metrics:\n      enabled: true\n"},
		{
			name: "both-enabled", wantQuery: true,
			settings: "  database:\n    tracing:\n      enabled: true\n    metrics:\n      enabled: true\n",
		},
		{name: "enabled", settings: "  database:\n    tracing:\n      enabled: true\n", wantQuery: true},
		{name: "redis-only", settings: "  redis:\n    redis_trace_enabled: true\n"},
		{
			name:      "gorm-with-redis-disabled",
			settings:  "  database:\n    tracing:\n      enabled: true\n  redis:\n    redis_trace_enabled: false\n",
			wantQuery: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			receiver := &traceReceiver{}
			server := httptest.NewServer(receiver)
			defer server.Close()

			dir := t.TempDir()
			yaml := fmt.Sprintf(`app:
  open-telemetry:
    app_name: gorm-integration
    trace_url: %q
    exporter_type: http
    insecure: true
    trace_enabled: true
    sample_rate: 1.0
    batch_timeout: 1h
    max_batch_size: 8
    max_queue_size: 16
    export_timeout: 1s
    shutdown_timeout: 3s
%s`, strings.TrimPrefix(server.URL, "http://"), tc.settings)
			require.NoError(t, os.WriteFile(filepath.Join(dir, "base.yaml"), []byte(yaml), 0600))
			t.Setenv("APP_CONFIG_DIR", dir)
			t.Setenv("APP_ENV", "local")

			db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{})
			require.NoError(t, err)
			service := &queryService{db: db}
			app := fx.New(brickstrace.GORMModule(), fx.Supply(db), fx.NopLogger)
			require.NoError(t, app.Err())
			require.NoError(t, app.Start(context.Background()))

			ctx, parent := brickstrace.Span(context.Background(), "public-service")
			result, queryErr := service.Execute(ctx, "query-secret-value")
			parentID := parent.SpanContext().SpanID().String()
			traceID := parent.SpanContext().TraceID().String()
			parent.End()
			require.NoError(t, app.Stop(context.Background()))
			require.NoError(t, queryErr)
			require.Equal(t, "query-secret-value", result)

			spans := receiver.spans()
			wantCount := 1
			if tc.wantQuery {
				wantCount++
			}
			require.Len(t, spans, wantCount)
			for _, span := range spans {
				if span.GetName() == "public-service" {
					continue
				}
				require.Equal(t, traceID, hex.EncodeToString(span.GetTraceId()))
				require.Equal(t, parentID, hex.EncodeToString(span.GetParentSpanId()))
				require.NotContains(t, span.String(), "query-secret-value")
				require.NotContains(t, strings.ToLower(span.String()), "select")
			}
		})
	}

	t.Run("sequential-apps-reuse-client", func(t *testing.T) {
		receiver := &traceReceiver{}
		server := httptest.NewServer(receiver)
		defer server.Close()
		configureTracing(t, strings.TrimPrefix(server.URL, "http://"),
			"    trace_enabled: true\n    sample_rate: 1.0\n  database:\n    tracing:\n      enabled: true")

		db, openErr := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{})
		require.NoError(t, openErr)
		service := &queryService{db: db}
		for i := range 2 {
			app := fx.New(brickstrace.GORMModule(), fx.Supply(db), fx.NopLogger)
			require.NoError(t, app.Err())
			require.NoError(t, app.Start(context.Background()))

			ctx, parent := brickstrace.Span(context.Background(), "sequential-service")
			result, queryErr := service.Execute(ctx, "sequential-value")
			parentID := parent.SpanContext().SpanID().String()
			traceID := parent.SpanContext().TraceID().String()
			parent.End()
			require.NoError(t, app.Stop(context.Background()))
			require.NoError(t, queryErr)
			require.Equal(t, "sequential-value", result)

			spans := receiver.spans()
			require.Len(t, spans, 2*(i+1))
			children := 0
			for _, span := range spans {
				if hex.EncodeToString(span.GetTraceId()) == traceID &&
					hex.EncodeToString(span.GetParentSpanId()) == parentID {
					children++
				}
			}
			require.Equal(t, 1, children)

			result, queryErr = service.Execute(context.Background(), "after-stop")
			require.NoError(t, queryErr)
			require.Equal(t, "after-stop", result)
		}
	})

	t.Run("disabled-does-not-resolve-unused-client", func(t *testing.T) {
		configureTracing(t, "localhost:4318", "    trace_enabled: false")
		resolved := false
		app := fx.New(brickstrace.GORMModule(), fx.Provide(func() (*gorm.DB, error) {
			resolved = true
			return nil, errors.New("unused client provider")
		}), fx.NopLogger)
		require.NoError(t, app.Err())
		require.NoError(t, app.Start(context.Background()))
		require.NoError(t, app.Stop(context.Background()))
		require.False(t, resolved)
	})

	t.Run("enabled-resolves-client-provider-error", func(t *testing.T) {
		configureTracing(t, "localhost:4318", "    trace_enabled: false\n  database:\n    tracing:\n      enabled: true")
		providerErr := errors.New("client provider failed")
		app := fx.New(brickstrace.GORMModule(), fx.Provide(func() (*gorm.DB, error) {
			return nil, providerErr
		}), fx.NopLogger)
		require.ErrorIs(t, app.Err(), providerErr)
		require.False(t, brickstrace.IsInitialized())
	})

	t.Run("invalid-enable-value-rejects-app", func(t *testing.T) {
		configureTracing(t, "localhost:4318",
			"    trace_enabled: false\n  database:\n    tracing:\n      enabled: not-a-bool")
		app := fx.New(brickstrace.GORMModule(), fx.NopLogger)
		require.Error(t, app.Err())
		require.False(t, brickstrace.IsInitialized())
	})

	t.Run("client-provider-start-query-is-traced", func(t *testing.T) {
		receiver := &traceReceiver{}
		server := httptest.NewServer(receiver)
		defer server.Close()
		configureTracing(t, strings.TrimPrefix(server.URL, "http://"),
			"    trace_enabled: true\n    sample_rate: 1.0\n  database:\n    tracing:\n      enabled: true")

		var parentID, traceID string
		app := fx.New(brickstrace.GORMModule(), fx.Provide(func(lifecycle fx.Lifecycle) (*gorm.DB, error) {
			db, openErr := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{})
			if openErr != nil {
				return nil, openErr
			}
			service := &queryService{db: db}
			lifecycle.Append(fx.Hook{OnStart: func(ctx context.Context) error {
				ctx, parent := brickstrace.Span(ctx, "provider-start")
				parentID = parent.SpanContext().SpanID().String()
				traceID = parent.SpanContext().TraceID().String()
				defer parent.End()
				_, queryErr := service.Execute(ctx, "startup-value")
				return queryErr
			}})
			return db, nil
		}), fx.NopLogger)
		require.NoError(t, app.Err())
		require.NoError(t, app.Start(context.Background()))
		require.NoError(t, app.Stop(context.Background()))
		spans := receiver.spans()
		require.Len(t, spans, 2)
		children := 0
		for _, span := range spans {
			if hex.EncodeToString(span.GetTraceId()) == traceID &&
				hex.EncodeToString(span.GetParentSpanId()) == parentID {
				children++
			}
		}
		require.Equal(t, 1, children)
	})

	t.Run("existing-plugin-rejects-startup", func(t *testing.T) {
		receiver := &traceReceiver{}
		server := httptest.NewServer(receiver)
		defer server.Close()
		configureTracing(t, strings.TrimPrefix(server.URL, "http://"),
			"    trace_enabled: true\n    sample_rate: 1.0\n  database:\n    tracing:\n      enabled: true")

		db, openErr := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{})
		require.NoError(t, openErr)
		require.NoError(t, db.Use(existingGORMPlugin{}))
		app := fx.New(brickstrace.GORMModule(), fx.Supply(db), fx.NopLogger)
		require.NoError(t, app.Err())
		require.Error(t, app.Start(context.Background()))
		require.False(t, brickstrace.IsInitialized())
	})

	t.Run("existing-callback-preserved-on-failed-start", func(t *testing.T) {
		configureTracing(t, "localhost:4318",
			"    trace_enabled: true\n    sample_rate: 1.0\n  database:\n    tracing:\n      enabled: true")
		db, openErr := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{})
		require.NoError(t, openErr)

		calls := 0
		require.NoError(t, db.Callback().Row().Before("gorm:row").Register("otel:before:row", func(*gorm.DB) {
			calls++
		}))
		app := fx.New(brickstrace.GORMModule(), fx.Supply(db), fx.NopLogger)
		require.NoError(t, app.Err())
		require.Error(t, app.Start(context.Background()))
		require.False(t, brickstrace.IsInitialized())

		service := &queryService{db: db}
		result, queryErr := service.Execute(context.Background(), "after-failed-start")
		require.NoError(t, queryErr)
		require.Equal(t, "after-failed-start", result)
		require.Equal(t, 1, calls)
	})

	t.Run("missing-client-rejects-startup", func(t *testing.T) {
		configureTracing(t, "localhost:4318",
			"    trace_enabled: true\n    sample_rate: 1.0\n  database:\n    tracing:\n      enabled: true")
		app := fx.New(brickstrace.GORMModule(), fx.NopLogger)
		if app.Err() == nil {
			require.Error(t, app.Start(context.Background()))
		}
		require.False(t, brickstrace.IsInitialized())
	})
}
