package chi_test

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	bricksconfig "github.com/cristiano-pacheco/bricks/pkg/config"
	"github.com/cristiano-pacheco/bricks/pkg/http/server/chi"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
)

func TestServer_Start_PortZeroReportsEffectiveAddressAndShutsDown(t *testing.T) {
	server, err := chi.New(testConfig("127.0.0.1:0", false))
	require.NoError(t, err)

	server.Router().Get("/registered", func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = writer.Write([]byte("registered"))
	})
	server.SetupRoutes()

	startErrors := make(chan error, 1)
	go func() {
		startErrors <- server.Start()
	}()
	defer func() {
		_ = server.Shutdown(context.Background())
	}()

	address := waitForEffectiveAddress(t, server.Addr)
	require.Equal(t, "127.0.0.1", hostOf(t, address))

	status, body := waitForResponse(t, "http://"+address+"/registered")
	require.Equal(t, http.StatusOK, status)
	require.Equal(t, "registered", body)

	require.NoError(t, server.Shutdown(context.Background()))
	require.ErrorIs(t, <-startErrors, http.ErrServerClosed)
}

func TestServer_MetricsCanBeDisabledWithoutCreatingAListener(t *testing.T) {
	server, err := chi.New(testConfig("127.0.0.1:0", false))
	require.NoError(t, err)
	require.Empty(t, server.MetricsAddr())
}

func TestServer_RequestLoggingCanBeDisabled(t *testing.T) {
	server, err := chi.New(testConfig("127.0.0.1:0", false))
	require.NoError(t, err)
	server.Router().Get("/request", func(writer http.ResponseWriter, request *http.Request) {
		require.Nil(t, middleware.GetLogEntry(request))
		writer.WriteHeader(http.StatusNoContent)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/request", nil)
	server.Router().ServeHTTP(recorder, request)
	require.Equal(t, http.StatusNoContent, recorder.Code)
}

func TestServer_MetricsPortZeroReportsEffectiveAddress(t *testing.T) {
	cfg := testConfig("127.0.0.1:0", true)
	cfg.MetricsAddress = "127.0.0.1:0"
	server, err := chi.New(cfg)
	require.NoError(t, err)
	server.SetupRoutes()

	startErrors := make(chan error, 1)
	go func() {
		startErrors <- server.Start()
	}()
	defer func() {
		_ = server.Shutdown(context.Background())
	}()

	mainAddress := waitForEffectiveAddress(t, server.Addr)
	metricsAddress := waitForEffectiveAddress(t, server.MetricsAddr)
	require.NotEqual(t, mainAddress, metricsAddress)

	status, _ := waitForResponse(t, "http://"+metricsAddress+"/metrics")
	require.Equal(t, http.StatusOK, status)

	require.NoError(t, server.Shutdown(context.Background()))
	require.ErrorIs(t, <-startErrors, http.ErrServerClosed)
}

func TestServer_Start_ReturnsErrorForOccupiedAddress(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer func() {
		_ = listener.Close()
	}()

	server, err := chi.New(testConfig(listener.Addr().String(), false))
	require.NoError(t, err)
	require.ErrorIs(t, server.Start(), chi.ErrListen)
}

func TestServer_FxStart_ReturnsErrorForOccupiedAddress(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer func() {
		_ = listener.Close()
	}()

	loaded := loadConfig(t, listener.Addr().String(), false, false, false)
	app, _ := newLifecycleApp(loaded, nil, nil)

	require.ErrorIs(t, app.Start(context.Background()), chi.ErrListen)
	_ = app.Stop(context.Background())
}

func TestServer_FxStartRegistersRoutesBeforeServing(t *testing.T) {
	loaded := loadConfig(t, "127.0.0.1:0", false, false, false)
	route := routeFunc(func(server *chi.Server) {
		server.Router().Get("/from-fx-route", func(writer http.ResponseWriter, _ *http.Request) {
			_, _ = writer.Write([]byte("from-fx-route"))
		})
	})
	app, server := newLifecycleApp(loaded, route, nil)

	require.NoError(t, app.Start(context.Background()))
	address := waitForEffectiveAddress(t, server.Addr)
	status, body := waitForResponse(t, "http://"+address+"/from-fx-route")
	require.Equal(t, http.StatusOK, status)
	require.Equal(t, "from-fx-route", body)
	require.NoError(t, app.Stop(context.Background()))
}

func TestServer_FxStartHonorsRouteLogging(t *testing.T) {
	tests := []struct {
		name    string
		enabled bool
	}{
		{name: "disabled", enabled: false},
		{name: "enabled", enabled: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var logs strings.Builder
			logger := slog.New(slog.NewTextHandler(&logs, nil))
			loaded := loadConfig(t, "127.0.0.1:0", false, false, tt.enabled)
			app, server := newLifecycleApp(loaded, nil, logger)

			require.NoError(t, app.Start(context.Background()))
			require.NotEmpty(t, server.Addr())
			require.NoError(t, app.Stop(context.Background()))

			if tt.enabled {
				require.Contains(t, logs.String(), "HTTP Server")
				return
			}
			require.Empty(t, logs.String())
		})
	}
}

func TestConfigValidate_AcceptsPortZeroAndRejectsMalformedAddress(t *testing.T) {
	require.NoError(t, (chi.Config{Port: 0}).Validate())

	server, err := chi.New(chi.Config{Address: "127.0.0.1"})
	require.Error(t, err)
	require.Nil(t, server)
	require.ErrorIs(t, err, chi.ErrInvalidConfig)
	require.ErrorIs(t, err, chi.ErrInvalidAddress)
}

type routeFunc func(*chi.Server)

func (route routeFunc) Setup(server *chi.Server) {
	route(server)
}

func testConfig(address string, metrics bool) chi.Config {
	return chi.Config{
		Address:              address,
		EnableMetrics:        metrics,
		EnableRequestLogging: false,
		EnableRouteLogging:   false,
	}
}

func loadConfig(
	t *testing.T,
	address string,
	metrics, requestLogging, routeLogging bool,
) bricksconfig.Config[chi.Config] {
	t.Helper()
	configDir := t.TempDir()
	content := fmt.Sprintf(`app:
  http:
    address: %q
    enablemetrics: %t
    enablerequestlogging: %t
    enableroutelogging: %t
`, address, metrics, requestLogging, routeLogging)
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "base.yaml"), []byte(content), 0o600))
	t.Setenv("APP_CONFIG_DIR", configDir)
	t.Setenv("APP_ENV", "test")

	loaded, err := bricksconfig.New[chi.Config](bricksconfig.WithPath("app.http"))
	require.NoError(t, err)
	return loaded
}

func newLifecycleApp(
	loaded bricksconfig.Config[chi.Config],
	route chi.Route,
	logger *slog.Logger,
) (*fx.App, *chi.Server) {
	var server *chi.Server
	options := []fx.Option{
		fx.Provide(func() bricksconfig.Config[chi.Config] {
			return loaded
		}),
		fx.Provide(chi.NewWithLifecycle),
		fx.Invoke(func(value *chi.Server) {
			server = value
		}),
	}
	if route != nil {
		options = append(options, fx.Provide(fx.Annotate(
			func() chi.Route { return route },
			fx.ResultTags(`group:"routes"`),
		)))
	}
	if logger != nil {
		options = append(options, fx.Supply(logger))
	}
	return fx.New(options...), server
}

func waitForEffectiveAddress(t *testing.T, address func() string) string {
	t.Helper()
	var effective string
	require.Eventually(t, func() bool {
		effective = address()
		_, port, err := net.SplitHostPort(effective)
		return err == nil && port != "0"
	}, time.Second, 10*time.Millisecond)
	return effective
}

func waitForResponse(t *testing.T, url string) (int, string) {
	t.Helper()
	var status int
	var body string
	client := &http.Client{Timeout: time.Second}
	require.Eventually(t, func() bool {
		response, err := client.Get(url)
		if err != nil {
			return false
		}
		defer response.Body.Close()
		contents, err := io.ReadAll(response.Body)
		if err != nil {
			return false
		}
		status = response.StatusCode
		body = string(contents)
		return true
	}, time.Second, 10*time.Millisecond)
	return status, body
}

func hostOf(t *testing.T, address string) string {
	t.Helper()
	host, _, err := net.SplitHostPort(address)
	require.NoError(t, err)
	return host
}
