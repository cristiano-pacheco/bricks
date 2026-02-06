package redis_test

import (
	"testing"
	"time"

	"github.com/cristiano-pacheco/bricks/pkg/redis"
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
			tt.config.SetDefaults()
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestConfigDefaults tests default values
func TestConfigDefaults(t *testing.T) {
	cfg := redis.Config{
		URL:  "redis://localhost:6379",
		Type: redis.ClientTypeSingleNode,
	}

	cfg.SetDefaults()

	if cfg.DialTimeout != 5*time.Second {
		t.Errorf("Expected DialTimeout to be 5s, got %v", cfg.DialTimeout)
	}

	if cfg.PoolSize != 10 {
		t.Errorf("Expected PoolSize to be 10, got %d", cfg.PoolSize)
	}

	if cfg.MaxRetries != 3 {
		t.Errorf("Expected MaxRetries to be 3, got %d", cfg.MaxRetries)
	}
}
