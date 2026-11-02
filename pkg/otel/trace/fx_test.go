package trace_test

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	brickstrace "github.com/cristiano-pacheco/bricks/pkg/otel/trace"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	oteltrace "go.opentelemetry.io/otel/trace"
	collectortrace "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
	"go.uber.org/fx"
	"google.golang.org/protobuf/proto"
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
    app_name: lifecycle-test
    app_version: "1.2"
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
	if strings.Contains(settings, "max_queue_size:") {
		yaml = strings.Replace(yaml, "    max_queue_size: 16\n", "", 1)
	}

	if strings.Contains(settings, "exporter_type:") {
		yaml = strings.Replace(yaml, "    exporter_type: http\n", "", 1)
	}

	require.NoError(t, os.WriteFile(filepath.Join(dir, "base.yaml"), []byte(yaml), 0600))
	t.Setenv("APP_CONFIG_DIR", dir)
	t.Setenv("APP_ENV", "local")
}

func startTracing(t *testing.T) *fx.App {
	t.Helper()

	app := fx.New(brickstrace.Module, fx.NopLogger)
	require.NoError(t, app.Err())

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	require.NoError(t, app.Start(ctx))

	return app
}

func stopTracing(t *testing.T, app *fx.App) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	require.NoError(t, app.Stop(ctx))
}

//nolint:paralleltest // Fx tracing and configuration use process-global state and environment variables.
func TestModule_YAML_ExportPropagationAndSequentialApplications(t *testing.T) {
	receiver := &traceReceiver{}

	server := httptest.NewServer(receiver)
	defer server.Close()

	endpoint := strings.TrimPrefix(server.URL, "http://")
	for i := range 2 {
		configureTracing(t, endpoint, "    trace_enabled: true\n    sample_rate: 1.0")
		app := startTracing(t)
		carrier := propagation.MapCarrier{"traceparent": "00-0123456789abcdef0123456789abcdef-0123456789abcdef-01"}
		parent := otel.GetTextMapPropagator().Extract(context.Background(), carrier)
		ctx, span := brickstrace.Span(parent, fmt.Sprintf("request-%d", i))
		outbound := propagation.MapCarrier{}
		otel.GetTextMapPropagator().Inject(ctx, outbound)
		require.Contains(t, outbound.Get("traceparent"), "0123456789abcdef0123456789abcdef")
		require.True(t, span.SpanContext().IsSampled())
		span.End()
		stopTracing(t, app)
	}

	receiver.mu.Lock()
	require.NotEmpty(t, receiver.requests)
	attributes := receiver.requests[0].GetResourceSpans()[0].GetResource().GetAttributes()
	receiver.mu.Unlock()

	identity := map[string]string{}
	for _, attribute := range attributes {
		identity[attribute.GetKey()] = attribute.GetValue().GetStringValue()
	}

	require.Equal(t, "lifecycle-test", identity["service.name"])
	require.Equal(t, "1.2", identity["service.version"])

	spans := receiver.spans()
	require.Len(t, spans, 2)

	for i, span := range spans {
		require.Equal(t, fmt.Sprintf("request-%d", i), span.GetName())
		require.Equal(t, "0123456789abcdef0123456789abcdef", hex.EncodeToString(span.GetTraceId()))
		require.Equal(t, "0123456789abcdef", hex.EncodeToString(span.GetParentSpanId()))
	}
}

//nolint:paralleltest // Fx tracing and configuration use process-global state and environment variables.
func TestModule_YAML_SamplingAndDisabled(t *testing.T) {
	receiver := &traceReceiver{}

	server := httptest.NewServer(receiver)
	defer server.Close()

	endpoint := strings.TrimPrefix(server.URL, "http://")

	for _, tc := range []struct {
		name, settings string
		sampled        bool
	}{
		{"zero", "    trace_enabled: true\n    sample_rate: 0.0", false},
		{"full", "    trace_enabled: true\n    sample_rate: 1.0", true},
		{"disabled", "    trace_enabled: false\n    sample_rate: 1.0", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			configureTracing(t, endpoint, tc.settings)
			app := startTracing(t)
			_, span := brickstrace.Span(context.Background(), tc.name)
			require.Equal(t, tc.sampled, span.SpanContext().IsSampled())
			span.End()
			stopTracing(t, app)
		})
	}

	spans := receiver.spans()
	require.Len(t, spans, 1)
	require.Equal(t, "full", spans[0].GetName())
}

//nolint:paralleltest // Fx tracing and configuration use process-global state and environment variables.
func TestModule_YAML_InvalidConfiguration(t *testing.T) {
	for _, settings := range []string{
		"    trace_enabled: true\n    sample_rate: 1.1",
		"    trace_enabled: true\n    sample_rate: -0.1",
		"    trace_enabled: true\n    sample_rate: 1.0\n    max_queue_size: -1",
		"    trace_enabled: true\n    sample_rate: 1.0\n    exporter_type: invalid",
	} {
		configureTracing(t, "localhost:4318", settings)

		app := fx.New(brickstrace.Module, fx.NopLogger)
		if app.Err() == nil {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			err := app.Start(ctx)

			cancel()
			require.Error(t, err, settings)
		}
	}
}

//nolint:paralleltest // Fx tracing and configuration use process-global state and environment variables.
func TestModule_BackendOutageAndFailedStartupCleanup(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	endpoint := listener.Addr().String()
	require.NoError(t, listener.Close())
	configureTracing(t, endpoint, "    trace_enabled: true\n    sample_rate: 1.0")
	app := startTracing(t)
	_, span := brickstrace.Span(context.Background(), "outage")
	span.End()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_ = app.Stop(ctx) // Export failure is allowed; shutdown must remain bounded.
	require.NoError(t, ctx.Err())

	failure := errors.New("later startup hook failed")
	app = fx.New(brickstrace.Module, fx.NopLogger, fx.Invoke(func(lifecycle fx.Lifecycle) {
		lifecycle.Append(fx.Hook{OnStart: func(context.Context) error { return failure }})
	}))
	require.NoError(t, app.Err())
	require.ErrorIs(t, app.Start(context.Background()), failure)

	receiver := &traceReceiver{}

	server := httptest.NewServer(receiver)
	defer server.Close()

	configureTracing(t, strings.TrimPrefix(server.URL, "http://"), "    trace_enabled: true\n    sample_rate: 1.0")
	app = startTracing(t)
	_, child := brickstrace.Span(
		oteltrace.ContextWithSpanContext(context.Background(), oteltrace.SpanContext{}),
		"restart",
	)
	child.End()
	stopTracing(t, app)
}

//nolint:paralleltest // Fx tracing and configuration use process-global state and environment variables.
func TestModule_YAML_SamplingRespectsParent(t *testing.T) {
	receiver := &traceReceiver{}

	server := httptest.NewServer(receiver)
	defer server.Close()

	endpoint := strings.TrimPrefix(server.URL, "http://")

	for _, tc := range []struct {
		name    string
		rate    string
		parent  string
		sampled bool
	}{
		{"sampled-parent", "0.0", "01", true},
		{"unsampled-parent", "1.0", "00", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			configureTracing(t, endpoint, "    trace_enabled: true\n    sample_rate: "+tc.rate)
			app := startTracing(t)
			carrier := propagation.MapCarrier{
				"traceparent": "00-0123456789abcdef0123456789abcdef-0123456789abcdef-" + tc.parent,
			}
			ctx := otel.GetTextMapPropagator().Extract(context.Background(), carrier)
			_, span := brickstrace.Span(ctx, tc.name)
			require.Equal(t, tc.sampled, span.SpanContext().IsSampled())
			span.End()
			stopTracing(t, app)
		})
	}

	spans := receiver.spans()
	require.Len(t, spans, 1)
	require.Equal(t, "sampled-parent", spans[0].GetName())
}
