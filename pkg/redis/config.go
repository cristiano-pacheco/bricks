package redis

import (
	"errors"
	"strings"
	"time"
)

// ClientType represents the type of Redis client
type ClientType string

const (
	// ClientTypeSingleNode represents a single-node Redis instance
	ClientTypeSingleNode ClientType = "single_node"
	// ClientTypeCluster represents a Redis cluster
	ClientTypeCluster ClientType = "cluster"
	// ClientTypeSentinel represents a Redis Sentinel setup
	ClientTypeSentinel ClientType = "sentinel"
	// ClientTypeFailover represents a Redis Failover setup
	ClientTypeFailover ClientType = "failover"
)

// Config holds the configuration for Redis connection
type Config struct {
	// Connection parameters
	URL      string     // Redis URL (e.g., redis://localhost:6379)
	Password string     // Redis password
	DB       int        // Database number (0-15 for single node)
	Type     ClientType // Client type (single_node, cluster, sentinel, failover)

	// Cluster-specific parameters
	ClusterAddrs []string // Cluster node addresses (for cluster mode)

	// Sentinel-specific parameters
	SentinelAddrs    []string // Sentinel addresses (for sentinel mode)
	MasterName       string   // Master name (for sentinel/failover mode)
	SentinelPassword string   // Sentinel password (for sentinel mode)
	SentinelUsername string   // Sentinel username (for sentinel mode)
	RouteByLatency   bool     // Route read commands by latency (sentinel mode)
	RouteRandomly    bool     // Route read commands randomly (sentinel mode)
	ReplicaOnly      bool     // Use replica nodes only (sentinel mode)

	// Connection pool settings
	MaxRetries            int           // Maximum number of retries before giving up
	MinRetryBackoff       time.Duration // Minimum backoff between each retry
	MaxRetryBackoff       time.Duration // Maximum backoff between each retry
	DialTimeout           time.Duration // Dial timeout for establishing new connections
	ReadTimeout           time.Duration // Timeout for socket reads
	WriteTimeout          time.Duration // Timeout for socket writes
	ContextTimeoutEnabled bool          // Enable timeouts from context deadline
	PoolFIFO              bool          // Use FIFO mode for pool
	PoolSize              int           // Maximum number of socket connections
	PoolTimeout           time.Duration // Amount of time client waits for connection if all connections busy
	MinIdleConns          int           // Minimum number of idle connections
	MaxIdleConns          int           // Maximum number of idle connections
	ConnMaxIdleTime       time.Duration // Amount of time after which client closes idle connections
	ConnMaxLifetime       time.Duration // Connection age at which client retires the connection

	// TLS configuration
	EnableTLS     bool   // Enable TLS
	TLSSkipVerify bool   // Skip TLS certificate verification
	TLSServerName string // TLS server name

	// Advanced parameters
	MaxRedirects     int    // Maximum number of redirects (cluster mode)
	ReadOnly         bool   // Enable read-only mode (cluster mode)
	UnstableMode     bool   // Allow connections to unstable cluster nodes
	Protocol         int    // Redis protocol version (2 or 3)
	DisableIndentity bool   // Disable set-info on connect
	IdentitySuffix   string // Add suffix to client name for debugging

	// Application-level settings
	Namespace      string        // Key namespace prefix
	EnableMetrics  bool          // Enable metrics collection
	CommandTimeout time.Duration // Default timeout for commands
}

// Validate validates the Redis configuration
func (c Config) Validate() error {
	// Validate client type
	if strings.TrimSpace(string(c.Type)) == "" {
		return ErrEmptyClientType
	}

	clientType := strings.TrimSpace(strings.ToLower(string(c.Type)))
	if clientType != string(ClientTypeSingleNode) &&
		clientType != string(ClientTypeCluster) &&
		clientType != string(ClientTypeSentinel) &&
		clientType != string(ClientTypeFailover) {
		return &ConfigError{
			Field: "Type",
			Value: c.Type,
			Err:   ErrInvalidClientType,
		}
	}

	// Validate URL for single node and failover
	if c.Type == ClientTypeSingleNode || c.Type == ClientTypeFailover {
		if strings.TrimSpace(c.URL) == "" {
			return &ConfigError{
				Field: "URL",
				Value: c.URL,
				Err:   ErrMissingURL,
			}
		}
	}

	// Validate cluster addresses
	if c.Type == ClientTypeCluster {
		if len(c.ClusterAddrs) == 0 && strings.TrimSpace(c.URL) == "" {
			return &ConfigError{
				Field: "ClusterAddrs",
				Value: c.ClusterAddrs,
				Err:   errors.New("cluster addresses or URL is required for cluster mode"),
			}
		}
	}

	// Validate sentinel configuration
	if c.Type == ClientTypeSentinel {
		if len(c.SentinelAddrs) == 0 {
			return &ConfigError{
				Field: "SentinelAddrs",
				Value: c.SentinelAddrs,
				Err:   errors.New("sentinel addresses are required for sentinel mode"),
			}
		}
		if strings.TrimSpace(c.MasterName) == "" {
			return &ConfigError{
				Field: "MasterName",
				Value: c.MasterName,
				Err:   errors.New("master name is required for sentinel mode"),
			}
		}
	}

	// Validate failover configuration
	if c.Type == ClientTypeFailover {
		if strings.TrimSpace(c.MasterName) == "" {
			return &ConfigError{
				Field: "MasterName",
				Value: c.MasterName,
				Err:   errors.New("master name is required for failover mode"),
			}
		}
	}

	// Validate DB number (only applicable for single node)
	if c.Type == ClientTypeSingleNode {
		if c.DB < 0 || c.DB > 15 {
			return &ConfigError{
				Field: "DB",
				Value: c.DB,
				Err:   ErrInvalidDB,
			}
		}
	}

	// Validate pool settings
	if c.PoolSize < 0 {
		return &ConfigError{
			Field: "PoolSize",
			Value: c.PoolSize,
			Err:   errors.New("pool size cannot be negative"),
		}
	}

	if c.MinIdleConns < 0 {
		return &ConfigError{
			Field: "MinIdleConns",
			Value: c.MinIdleConns,
			Err:   errors.New("min idle connections cannot be negative"),
		}
	}

	if c.MaxIdleConns < 0 {
		return &ConfigError{
			Field: "MaxIdleConns",
			Value: c.MaxIdleConns,
			Err:   errors.New("max idle connections cannot be negative"),
		}
	}

	if c.MaxRetries < 0 {
		return &ConfigError{
			Field: "MaxRetries",
			Value: c.MaxRetries,
			Err:   errors.New("max retries cannot be negative"),
		}
	}

	return nil
}

// SetDefaults sets default values for unset configuration fields
func (c *Config) SetDefaults() {
	if c.Type == "" {
		c.Type = ClientTypeSingleNode
	}

	if c.DialTimeout == 0 {
		c.DialTimeout = 5 * time.Second
	}

	if c.ReadTimeout == 0 {
		c.ReadTimeout = 3 * time.Second
	}

	if c.WriteTimeout == 0 {
		c.WriteTimeout = 3 * time.Second
	}

	if c.PoolSize == 0 {
		c.PoolSize = 10
	}

	if c.MinIdleConns == 0 {
		c.MinIdleConns = 2
	}

	if c.MaxRetries == 0 {
		c.MaxRetries = 3
	}

	if c.MinRetryBackoff == 0 {
		c.MinRetryBackoff = 8 * time.Millisecond
	}

	if c.MaxRetryBackoff == 0 {
		c.MaxRetryBackoff = 512 * time.Millisecond
	}

	if c.PoolTimeout == 0 {
		c.PoolTimeout = 4 * time.Second
	}

	if c.ConnMaxIdleTime == 0 {
		c.ConnMaxIdleTime = 30 * time.Minute
	}

	if c.ConnMaxLifetime == 0 {
		c.ConnMaxLifetime = 1 * time.Hour
	}

	if c.CommandTimeout == 0 {
		c.CommandTimeout = 5 * time.Second
	}
}
