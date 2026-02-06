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

const (
	defaultDialTimeout     = 5 * time.Second
	defaultReadTimeout     = 3 * time.Second
	defaultWriteTimeout    = 3 * time.Second
	defaultPoolSize        = 10
	defaultMinIdleConns    = 2
	defaultMaxRetries      = 3
	defaultMinRetryBackoff = 8 * time.Millisecond
	defaultMaxRetryBackoff = 512 * time.Millisecond
	defaultPoolTimeout     = 4 * time.Second
	defaultConnMaxIdleTime = 30 * time.Minute
	defaultConnMaxLifetime = 1 * time.Hour
	defaultCommandTimeout  = 5 * time.Second
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
	validators := []func() error{
		c.validateClientType,
		c.validateURL,
		c.validateCluster,
		c.validateSentinel,
		c.validateFailover,
		c.validateDB,
		c.validatePoolSettings,
	}

	for _, validate := range validators {
		if err := validate(); err != nil {
			return err
		}
	}

	return nil
}

// SetDefaults sets default values for unset configuration fields
func (c *Config) SetDefaults() {
	if c.Type == "" {
		c.Type = ClientTypeSingleNode
	}

	setDefaultDuration(&c.DialTimeout, defaultDialTimeout)
	setDefaultDuration(&c.ReadTimeout, defaultReadTimeout)
	setDefaultDuration(&c.WriteTimeout, defaultWriteTimeout)
	setDefaultInt(&c.PoolSize, defaultPoolSize)
	setDefaultInt(&c.MinIdleConns, defaultMinIdleConns)
	setDefaultInt(&c.MaxRetries, defaultMaxRetries)
	setDefaultDuration(&c.MinRetryBackoff, defaultMinRetryBackoff)
	setDefaultDuration(&c.MaxRetryBackoff, defaultMaxRetryBackoff)
	setDefaultDuration(&c.PoolTimeout, defaultPoolTimeout)
	setDefaultDuration(&c.ConnMaxIdleTime, defaultConnMaxIdleTime)
	setDefaultDuration(&c.ConnMaxLifetime, defaultConnMaxLifetime)
	setDefaultDuration(&c.CommandTimeout, defaultCommandTimeout)
}

func (c Config) validateClientType() error {
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

	return nil
}

func (c Config) validateURL() error {
	if c.Type != ClientTypeSingleNode && c.Type != ClientTypeFailover {
		return nil
	}
	if strings.TrimSpace(c.URL) == "" {
		return &ConfigError{
			Field: "URL",
			Value: c.URL,
			Err:   ErrMissingURL,
		}
	}
	return nil
}

func (c Config) validateCluster() error {
	if c.Type != ClientTypeCluster {
		return nil
	}
	if len(c.ClusterAddrs) == 0 && strings.TrimSpace(c.URL) == "" {
		return &ConfigError{
			Field: "ClusterAddrs",
			Value: c.ClusterAddrs,
			Err:   errors.New("cluster addresses or URL is required for cluster mode"),
		}
	}
	return nil
}

func (c Config) validateSentinel() error {
	if c.Type != ClientTypeSentinel {
		return nil
	}
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
	return nil
}

func (c Config) validateFailover() error {
	if c.Type != ClientTypeFailover {
		return nil
	}
	if strings.TrimSpace(c.MasterName) == "" {
		return &ConfigError{
			Field: "MasterName",
			Value: c.MasterName,
			Err:   errors.New("master name is required for failover mode"),
		}
	}
	return nil
}

func (c Config) validateDB() error {
	if c.Type != ClientTypeSingleNode {
		return nil
	}
	if c.DB < 0 || c.DB > 15 {
		return &ConfigError{
			Field: "DB",
			Value: c.DB,
			Err:   ErrInvalidDB,
		}
	}
	return nil
}

func (c Config) validatePoolSettings() error {
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

func setDefaultDuration(value *time.Duration, defaultValue time.Duration) {
	if *value == 0 {
		*value = defaultValue
	}
}

func setDefaultInt(value *int, defaultValue int) {
	if *value == 0 {
		*value = defaultValue
	}
}
