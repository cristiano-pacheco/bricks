package chi_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cristiano-pacheco/bricks/pkg/http/server/chi"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
)

func TestLegacyMetricsLifecycle_ReadTimeout(t *testing.T) {
	publicListener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	metricsListener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	publicPort := publicListener.Addr().(*net.TCPAddr).Port
	metricsPort := metricsListener.Addr().(*net.TCPAddr).Port

	require.NoError(t, publicListener.Close())
	require.NoError(t, metricsListener.Close())

	dir := t.TempDir()
	data := fmt.Sprintf(
		"app:\n  http:\n    port: %d\n    metricsport: %d\n    enablemetrics: true\n    readtimeout: 100ms\n",
		publicPort,
		metricsPort,
	)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "base.yaml"), []byte(data), 0600))
	t.Setenv("APP_CONFIG_DIR", dir)
	t.Setenv("APP_ENV", "local")

	app := fx.New(chi.Module, fx.NopLogger)
	require.NoError(t, app.Start(context.Background()))
	t.Cleanup(func() {
		require.NoError(t, app.Stop(context.Background()))
	})

	address := fmt.Sprintf("127.0.0.1:%d", metricsPort)
	client := &http.Client{Timeout: time.Second}
	response, err := client.Get("http://" + address + "/metrics")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode)
	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	require.Contains(t, string(body), "promhttp_metric_handler_requests_in_flight")
	require.NoError(t, response.Body.Close())

	connection, err := net.DialTimeout("tcp", address, time.Second)
	require.NoError(t, err)

	defer connection.Close()

	require.NoError(t, connection.SetDeadline(time.Now().Add(2*time.Second)))
	_, err = io.WriteString(connection, "GET /metrics HTTP/1.1\r\nHost: localhost\r\n")
	require.NoError(t, err)

	started := time.Now()

	_, err = io.ReadAll(connection)

	if _, ok := errors.AsType[net.Error](err); ok {
		t.Fatal("metrics server did not enforce the configured read timeout")
	}

	require.Less(t, time.Since(started), time.Second)
}

func TestLegacyMetricsLifecycle_OccupiedMetricsPortReleasesPublicListener(t *testing.T) {
	occupied, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	defer occupied.Close()

	public, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	publicAddress := public.Addr().String()
	require.NoError(t, public.Close())

	dir := t.TempDir()
	data := fmt.Sprintf("app:\n  http:\n    address: %q\n    metricsaddress: %q\n    enablemetrics: true\n",
		publicAddress, occupied.Addr().String())
	require.NoError(t, os.WriteFile(filepath.Join(dir, "base.yaml"), []byte(data), 0600))
	t.Setenv("APP_CONFIG_DIR", dir)
	t.Setenv("APP_ENV", "local")

	app := fx.New(chi.Module, fx.NopLogger)
	require.ErrorIs(t, app.Start(context.Background()), chi.ErrListen)

	listener, err := net.Listen("tcp", publicAddress)
	require.NoError(t, err)
	require.NoError(t, listener.Close())
}
