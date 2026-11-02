//go:build integration

package trace_test

import (
	"context"
	"encoding/hex"
	"errors"
	"net"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cristiano-pacheco/bricks/pkg/itestkit"
	brickstrace "github.com/cristiano-pacheco/bricks/pkg/otel/trace"
	bricksredis "github.com/cristiano-pacheco/bricks/pkg/redis"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
)

type redisService struct{ client redis.UniversalClient }

func (s *redisService) Execute(ctx context.Context) error {
	return s.client.Set(ctx, "private-key", "private-value", 0).Err()
}

//nolint:paralleltest // Configuration and the tracing provider are process-wide.
func TestRedisInstrumentation_PublicService(t *testing.T) {
	kit := itestkit.New(itestkit.DefaultConfig())
	require.NoError(t, kit.StartRedis())
	t.Cleanup(kit.StopRedis)
	client := kit.Redis()
	service := &redisService{client: client}
	receiver := &traceReceiver{}
	server := httptest.NewServer(receiver)
	defer server.Close()
	for _, tc := range []struct {
		name, settings   string
		tracing, metrics bool
	}{
		{name: "default"},
		{name: "database-only", settings: "  database:\n    tracing:\n      enabled: true\n"},
		{name: "metrics-only", settings: "  redis:\n    metrics:\n      enabled: true\n", metrics: true},
		{name: "tracing", settings: "  redis:\n    tracing:\n      enabled: true\n", tracing: true},
		{name: "both", settings: "  redis:\n    tracing:\n      enabled: true\n    metrics:\n      enabled: true\n", tracing: true, metrics: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			configureTracing(t, strings.TrimPrefix(server.URL, "http://"), "    trace_enabled: true\n    sample_rate: 1.0\n"+tc.settings)
			registry := prometheus.NewRegistry()
			app := fx.New(brickstrace.Module, brickstrace.RedisModule(), bricksredis.PoolMetricsModule(),
				fx.Provide(func() redis.UniversalClient { return client }, func() prometheus.Registerer { return registry }), fx.NopLogger)
			require.NoError(t, app.Err())
			require.NoError(t, app.Start(context.Background()))
			before := len(receiver.spans())
			ctx, parent := brickstrace.Span(context.Background(), "redis-service")
			traceID, parentID := parent.SpanContext().TraceID().String(), parent.SpanContext().SpanID().String()
			require.NoError(t, service.Execute(ctx))
			parent.End()
			families, err := registry.Gather()
			require.NoError(t, err)
			found := false
			for _, family := range families {
				if family.GetName() == "redis_pool_hit_total" {
					found = true
				}
			}
			require.Equal(t, tc.metrics, found)
			require.NoError(t, app.Stop(context.Background()))
			spans := receiver.spans()[before:]
			expected := 1
			if tc.tracing {
				expected++
			}
			require.Len(t, spans, expected)
			for _, span := range spans {
				if span.GetName() == "redis-service" {
					continue
				}
				require.Equal(t, traceID, hex.EncodeToString(span.GetTraceId()))
				require.Equal(t, parentID, hex.EncodeToString(span.GetParentSpanId()))
				require.NotContains(t, span.String(), "private-key")
				require.NotContains(t, span.String(), "private-value")
			}
		})
	}
}

//nolint:paralleltest // Configuration and tracing use process-wide state.
func TestRedisInstrumentation_ClientStartup(t *testing.T) {
	kit := itestkit.New(itestkit.DefaultConfig())
	require.NoError(t, kit.StartRedis())
	t.Cleanup(kit.StopRedis)
	client := kit.Redis()
	receiver := &traceReceiver{}
	server := httptest.NewServer(receiver)
	defer server.Close()
	configureTracing(t, strings.TrimPrefix(server.URL, "http://"), "    trace_enabled: true\n    sample_rate: 1.0\n  redis:\n    tracing:\n      enabled: true\n")
	for range 2 {
		before := len(receiver.spans())
		app := fx.New(brickstrace.Module, brickstrace.RedisModule(), fx.Provide(func(lc fx.Lifecycle) redis.UniversalClient {
			lc.Append(fx.Hook{OnStart: func(ctx context.Context) error {
				ctx, span := brickstrace.Span(ctx, "startup")
				defer span.End()
				return client.Set(ctx, "secret-key", "secret-value", 0).Err()
			}})
			return client
		}), fx.NopLogger)
		require.NoError(t, app.Err())
		require.NoError(t, app.Start(context.Background()))
		require.NoError(t, app.Stop(context.Background()))
		spans := receiver.spans()[before:]
		require.Len(t, spans, 2)
		var parent, child string
		for _, span := range spans {
			require.NotContains(t, span.String(), "secret-key")
			require.NotContains(t, span.String(), "secret-value")
			if span.GetName() == "startup" {
				parent = hex.EncodeToString(span.GetSpanId())
			} else {
				child = hex.EncodeToString(span.GetParentSpanId())
			}
		}
		require.NotEmpty(t, parent)
		require.Equal(t, parent, child)
	}
}

//nolint:paralleltest // Configuration and tracing use process-wide state.
func TestRedisInstrumentation_ReversedModuleOrder(t *testing.T) {
	kit := itestkit.New(itestkit.DefaultConfig())
	require.NoError(t, kit.StartRedis())
	t.Cleanup(kit.StopRedis)
	client := kit.Redis()
	for range 2 {
		receiver := &traceReceiver{}
		server := httptest.NewServer(receiver)
		configureTracing(t, strings.TrimPrefix(server.URL, "http://"),
			"    trace_enabled: true\n    sample_rate: 1.0\n  redis:\n    tracing:\n      enabled: true\n")
		app := fx.New(brickstrace.RedisModule(), brickstrace.Module,
			fx.Provide(func() redis.UniversalClient { return client }), fx.NopLogger)
		require.NoError(t, app.Err())
		require.NoError(t, app.Start(context.Background()))
		ctx, parent := brickstrace.Span(context.Background(), "reversed-order")
		parentID := parent.SpanContext().SpanID().String()
		require.NoError(t, (&redisService{client: client}).Execute(ctx))
		parent.End()
		require.NoError(t, app.Stop(context.Background()))
		server.Close()
		spans := receiver.spans()
		require.Len(t, spans, 2)
		for _, span := range spans {
			if span.GetName() != "reversed-order" {
				require.Equal(t, parentID, hex.EncodeToString(span.GetParentSpanId()))
			}
		}
	}
}

//nolint:paralleltest // Configuration and tracing use process-wide state.
func TestRedisInstrumentation_CommandPrivacy(t *testing.T) {
	kit := itestkit.New(itestkit.DefaultConfig())
	require.NoError(t, kit.StartRedis())
	t.Cleanup(kit.StopRedis)
	client := kit.Redis()
	receiver := &traceReceiver{}
	server := httptest.NewServer(receiver)
	defer server.Close()
	configureTracing(t, strings.TrimPrefix(server.URL, "http://"),
		"    trace_enabled: true\n    sample_rate: 1.0\n  redis:\n    tracing:\n      enabled: true\n")
	app := fx.New(brickstrace.RedisModule(), brickstrace.Module,
		fx.Provide(func() redis.UniversalClient { return client }), fx.NopLogger)
	require.NoError(t, app.Err())
	require.NoError(t, app.Start(context.Background()))
	ctx, parent := brickstrace.Span(context.Background(), "privacy")
	for _, name := range []string{"COMMAND", "CLUSTER"} {
		_ = client.Do(ctx, name, "sensitive-subcommand").Err()
		pipe := client.Pipeline()
		pipe.Do(ctx, name, "sensitive-pipeline-subcommand")
		pipe.Set(ctx, "private-pipeline-key", "private-pipeline-value", 0)
		_, _ = pipe.Exec(ctx)
	}
	require.NoError(t, client.Set(ctx, "safe-key", "safe-value", 0).Err())
	parent.End()
	require.NoError(t, app.Stop(context.Background()))
	spans := receiver.spans()
	var sawSet bool
	for _, span := range spans {
		if span.GetName() == "set" {
			sawSet = true
		}
		require.NotContains(t, strings.ToLower(span.GetName()), "command")
		require.NotContains(t, strings.ToLower(span.GetName()), "cluster")
		require.NotContains(t, span.String(), "sensitive-subcommand")
		require.NotContains(t, span.String(), "sensitive-pipeline-subcommand")
		require.NotContains(t, span.String(), "private-pipeline-key")
		require.NotContains(t, span.String(), "private-pipeline-value")
	}
	require.True(t, sawSet)
}

//nolint:paralleltest // Configuration uses process-wide state.
func TestRedisInstrumentation_RejectsRing(t *testing.T) {
	configureTracing(t, "localhost:4318", "  redis:\n    tracing:\n      enabled: true\n")
	var dials atomic.Int32
	ring := redis.NewRing(&redis.RingOptions{Addrs: map[string]string{"first": "localhost:6379"}, Dialer: func(context.Context, string, string) (net.Conn, error) {
		dials.Add(1)
		return nil, errors.New("unexpected dial")
	}})
	defer func() { _ = ring.Close() }()
	app := fx.New(brickstrace.RedisModule(), brickstrace.Module,
		fx.Provide(func() redis.UniversalClient { return ring }), fx.NopLogger)
	require.ErrorContains(t, app.Err(), "ring clients are not supported")
	require.Zero(t, dials.Load())
}

//nolint:paralleltest // Configuration uses process-wide state.
func TestRedisInstrumentation_RejectsCluster(t *testing.T) {
	configureTracing(t, "localhost:4318", "  redis:\n    tracing:\n      enabled: true\n")
	var dials atomic.Int32
	cluster := redis.NewClusterClient(&redis.ClusterOptions{Addrs: []string{"127.0.0.1:1"}, Dialer: func(context.Context, string, string) (net.Conn, error) {
		dials.Add(1)
		return nil, errors.New("unexpected dial")
	}})
	defer func() { _ = cluster.Close() }()
	app := fx.New(brickstrace.RedisModule(), brickstrace.Module,
		fx.Provide(func() redis.UniversalClient { return cluster }), fx.NopLogger)
	require.ErrorContains(t, app.Err(), "cluster clients are not supported")
	require.Zero(t, dials.Load())
}

//nolint:paralleltest // Configuration uses process-wide state.
func TestRedisInstrumentation_BricksClusterRejectedBeforeDial(t *testing.T) {
	for _, tc := range []struct {
		name   string
		module fx.Option
	}{
		{name: "universal", module: bricksredis.Module},
		{name: "wrapped", module: fx.Options(bricksredis.ClientModule, fx.Provide(func(c *bricksredis.Client) redis.UniversalClient { return c.UniversalClient() }))},
	} {
		t.Run(tc.name, func(t *testing.T) {
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			require.NoError(t, err)
			defer listener.Close()
			configureTracing(t, "localhost:4318", "  redis:\n    type: cluster\n    clusteraddrs:\n      - "+listener.Addr().String()+"\n    tracing:\n      enabled: true\n")
			app := fx.New(brickstrace.RedisModule(), brickstrace.Module, tc.module,
				fx.Provide(func() bricksredis.Config {
					return bricksredis.Config{Type: bricksredis.ClientTypeCluster, ClusterAddrs: []string{listener.Addr().String()}}
				}), fx.NopLogger)
			require.ErrorContains(t, app.Err(), "cluster clients are not supported")
			tcp := listener.(*net.TCPListener)
			require.NoError(t, tcp.SetDeadline(time.Now().Add(200*time.Millisecond)))
			conn, acceptErr := tcp.Accept()
			if conn != nil {
				_ = conn.Close()
			}
			require.Error(t, acceptErr)
			require.True(t, errors.Is(acceptErr, os.ErrDeadlineExceeded))
		})
	}
}
