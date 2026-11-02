package metrics_test

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cristiano-pacheco/bricks/pkg/http/server/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
)

func writeConfig(t *testing.T, enabled bool, port int) {
	t.Helper()
	dir := t.TempDir()
	data := fmt.Sprintf("app:\n  metrics:\n    enabled: %t\n    address: 127.0.0.1\n    port: %d\n", enabled, port)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "base.yaml"), []byte(data), 0600))
	t.Setenv("APP_CONFIG_DIR", dir)
	t.Setenv("APP_ENV", "local")
}

func application(registry *prometheus.Registry, server **metrics.Server, extra ...fx.Option) *fx.App {
	options := []fx.Option{
		metrics.Module,
		fx.Provide(func() prometheus.Gatherer {
			return registry
		}),
		fx.Populate(server),
		fx.NopLogger,
	}

	return fx.New(append(options, extra...)...)
}

//nolint:paralleltest // Configuration uses process-wide environment variables.
func TestMetricsLifecycle_CustomRegistryAndSequentialInstances(t *testing.T) {
	writeConfig(t, true, 0)

	for i := range 2 {
		registry := prometheus.NewRegistry()
		counter := prometheus.NewCounter(prometheus.CounterOpts{Name: "listener_custom_total", Help: "Custom counter."})
		require.NoError(t, registry.Register(counter))
		counter.Add(float64(i + 1))

		var server *metrics.Server

		app := application(registry, &server)
		require.NoError(t, app.Start(context.Background()))

		if server == nil {
			t.Fatal("metrics server was not populated")
		}

		address := server.Addr()
		_, port, err := net.SplitHostPort(address)
		require.NoError(t, err)
		require.NotEqual(t, "0", port)

		client := &http.Client{Timeout: time.Second}
		response, err := client.Get("http://" + address + "/metrics")
		require.NoError(t, err)
		body, err := io.ReadAll(response.Body)
		require.NoError(t, err)
		require.NoError(t, response.Body.Close())
		require.Equal(t, http.StatusOK, response.StatusCode)
		require.Contains(t, string(body), fmt.Sprintf("listener_custom_total %d", i+1))
		require.NotContains(t, string(body), "go_goroutines")
		require.NoError(t, app.Stop(context.Background()))

		connection, err := net.DialTimeout("tcp", address, time.Second)
		if err == nil {
			connection.Close()
		}

		require.Error(t, err)
	}
}

//nolint:paralleltest // Configuration uses process-wide environment variables.
func TestMetricsLifecycle_Disabled(t *testing.T) {
	writeConfig(t, false, 0)

	var server *metrics.Server

	app := application(prometheus.NewRegistry(), &server)
	require.NoError(t, app.Start(context.Background()))

	if server == nil {
		t.Fatal("metrics server was not populated")
	}

	require.Empty(t, server.Addr())
	require.NoError(t, app.Stop(context.Background()))
}

//nolint:paralleltest // Configuration uses process-wide environment variables.
func TestMetricsLifecycle_OccupiedPort(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port
	writeConfig(t, true, port)

	var server *metrics.Server

	app := application(prometheus.NewRegistry(), &server)
	err = app.Start(context.Background())
	require.ErrorContains(t, err, "bind metrics listener")
}

//nolint:paralleltest // Configuration uses process-wide environment variables.
func TestMetricsLifecycle_PartialStartupReleasesPort(t *testing.T) {
	writeConfig(t, true, 0)

	var server *metrics.Server

	app := application(prometheus.NewRegistry(), &server, fx.Invoke(func(lifecycle fx.Lifecycle, _ *metrics.Server) {
		lifecycle.Append(fx.Hook{OnStart: func(context.Context) error {
			return fmt.Errorf("later startup failure")
		}})
	}))
	err := app.Start(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), "later startup failure")

	if server == nil {
		t.Fatal("metrics server was not populated")
	}

	listener, err := net.Listen("tcp", server.Addr())
	require.NoError(t, err)
	require.NoError(t, listener.Close())
}

func TestMetricsServer_CanceledStartupDoesNotBind(t *testing.T) {
	t.Parallel()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	address := listener.Addr().String()
	port := listener.Addr().(*net.TCPAddr).Port
	require.NoError(t, listener.Close())

	server, err := metrics.New(
		metrics.Config{Enabled: true, Address: "127.0.0.1", Port: uint(port)},
		prometheus.NewRegistry(),
	)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	require.ErrorIs(t, server.StartContext(ctx), context.Canceled)
	require.Empty(t, server.Addr())

	listener, err = net.Listen("tcp", address)
	require.NoError(t, err)
	require.NoError(t, listener.Close())
}

func TestMetricsServer_ConcurrentStartAndShutdownReleasesListener(t *testing.T) {
	t.Parallel()

	for range 100 {
		server, err := metrics.New(metrics.Config{Enabled: true, Address: "127.0.0.1"}, prometheus.NewRegistry())
		require.NoError(t, err)

		startDone := make(chan error, 1)
		go func() {
			startDone <- server.StartContext(context.Background())
		}()

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		err = server.Shutdown(ctx)

		cancel()
		require.NoError(t, err)

		startErr := <-startDone
		if startErr != nil {
			require.ErrorIs(t, startErr, http.ErrServerClosed)
		}

		if address := server.Addr(); address != "" {
			listener, listenErr := net.Listen("tcp", address)
			require.NoError(t, listenErr)
			require.NoError(t, listener.Close())
		}
	}
}
