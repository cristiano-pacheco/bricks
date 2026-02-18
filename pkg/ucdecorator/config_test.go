package ucdecorator_test

import (
	"testing"

	"github.com/cristiano-pacheco/bricks/pkg/ucdecorator"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	t.Run("returns config with all decorators enabled", func(t *testing.T) {
		// Arrange & Act
		cfg := ucdecorator.DefaultConfig()

		// Assert
		require.True(t, cfg.Enabled)
		require.True(t, cfg.Logging)
		require.True(t, cfg.Metrics)
		require.True(t, cfg.Tracing)
		require.True(t, cfg.Translation)
	})
}
