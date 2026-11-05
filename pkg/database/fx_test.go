package database_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
	"gorm.io/gorm"

	"github.com/cristiano-pacheco/bricks/pkg/config"
	"github.com/cristiano-pacheco/bricks/pkg/database"
)

func TestModule_LoadsConfigAndProvidesDatabase(t *testing.T) {
	// Arrange
	dir := t.TempDir()
	require.NoError(
		t,
		os.WriteFile(
			filepath.Join(dir, "base.yaml"),
			[]byte("app:\n  database:\n    host: db.example\n    port: 5432\n    name: testdb\n    user: testuser\n"),
			0600,
		),
	)
	t.Setenv("APP_CONFIG_DIR", dir)
	t.Setenv("APP_ENV", "local")

	var got database.Config

	// Act
	app := fx.New(fx.NopLogger, database.Module, fx.Invoke(func(c config.Config[database.Config]) { got = c.Get() }))

	// Assert
	require.NoError(t, app.Err())
	require.Equal(t, "db.example", got.Host)
	require.Equal(t, uint(5432), got.Port)
	require.NoError(t, fx.ValidateApp(fx.NopLogger, database.Module, fx.Invoke(func(*gorm.DB) {})))
}

func TestModule_InvalidConfigFails(t *testing.T) {
	// Arrange
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "base.yaml"), []byte("app:\n  database:\n    host: ''\n"), 0600))
	t.Setenv("APP_CONFIG_DIR", dir)
	t.Setenv("APP_ENV", "local")

	// Act
	app := fx.New(fx.NopLogger, database.Module, fx.Invoke(func(*gorm.DB) {}))

	// Assert
	require.ErrorIs(t, app.Err(), database.ErrMissingHost)
}
