package redis_test

import (
	"testing"
	"time"

	"github.com/cristiano-pacheco/bricks/pkg/redis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConfigValidation tests configuration validation
func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  redis.Config
		wantErr bool
	}{
		{
			name: "valid single node config",
			config: redis.Config{
				URL:  "redis://localhost:6379",
				Type: redis.ClientTypeSingleNode,
			},
			wantErr: false,
		},
		{
			name: "missing URL",
			config: redis.Config{
				Type: redis.ClientTypeSingleNode,
			},
			wantErr: true,
		},
		{
			name: "invalid client type",
			config: redis.Config{
				URL:  "redis://localhost:6379",
				Type: "invalid",
			},
			wantErr: true,
		},
		{
			name: "invalid DB number",
			config: redis.Config{
				URL:  "redis://localhost:6379",
				Type: redis.ClientTypeSingleNode,
				DB:   99,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			tt.config.SetDefaults()

			// Act
			err := tt.config.Validate()

			// Assert
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestConfigDefaults tests default values
func TestConfigDefaults(t *testing.T) {
	// Arrange
	cfg := redis.Config{
		URL:  "redis://localhost:6379",
		Type: redis.ClientTypeSingleNode,
	}

	// Act
	cfg.SetDefaults()

	// Assert
	assert.Equal(t, 5*time.Second, cfg.DialTimeout)
	assert.Equal(t, 10, cfg.PoolSize)
	assert.Equal(t, 3, cfg.MaxRetries)
}
