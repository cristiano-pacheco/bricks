package redis_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	"github.com/cristiano-pacheco/bricks/pkg/redis"
)

func TestModules_LoadConfigAndProvideClients(t *testing.T) {
	for _, tc := range []struct {
		name    string
		module  fx.Option
		consume any
	}{
		{"universal", redis.Module, func(redis.UniversalClient) {}},
		{"wrapped", redis.ClientModule, func(*redis.Client) {}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			dir := t.TempDir()
			require.NoError(
				t,
				os.WriteFile(
					filepath.Join(dir, "base.yaml"),
					[]byte("app:\n  redis:\n    url: redis://example:6379\n    poolsize: 42\n"),
					0600,
				),
			)
			t.Setenv("APP_CONFIG_DIR", dir)
			t.Setenv("APP_ENV", "local")

			var got redis.Config
			// Act
			app := fx.New(fx.NopLogger, tc.module, fx.Invoke(func(c redis.Config) { got = c }))
			// Assert
			require.NoError(t, app.Err())
			require.Equal(t, "redis://example:6379", got.URL)
			require.Equal(t, 42, got.PoolSize)
			require.NoError(t, fx.ValidateApp(fx.NopLogger, tc.module, fx.Invoke(tc.consume)))
		})
	}
}

func TestModule_InvalidConfigFails(t *testing.T) {
	// Arrange
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "base.yaml"), []byte("app:\n  redis:\n    url: ''\n"), 0600))
	t.Setenv("APP_CONFIG_DIR", dir)
	t.Setenv("APP_ENV", "local")
	// Act
	app := fx.New(fx.NopLogger, redis.Module, fx.Invoke(func(redis.UniversalClient) {}))
	// Assert
	require.ErrorIs(t, app.Err(), redis.ErrMissingURL)
}
