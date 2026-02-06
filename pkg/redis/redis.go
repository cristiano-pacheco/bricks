package redis

import (
	"context"
	"crypto/tls"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// UniversalClient is an interface that represents both redis.Client and redis.ClusterClient
type UniversalClient = redis.UniversalClient

// Client wraps the Redis client with additional functionality
type Client struct {
	client    redis.UniversalClient
	config    Config
	opts      options
	namespace string
	metrics   *metricsCollector
	isClosed  bool
}

// NewClient creates a new Redis client with the given configuration and options
func NewClient(ctx context.Context, cfg Config, opts ...Option) (*Client, error) {
	// Set defaults
	cfg.SetDefaults()

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidConfig, err)
	}

	// Apply options
	options := defaultOptions()
	for _, opt := range opts {
		opt(&options)
	}

	// Create the appropriate client based on type
	var client redis.UniversalClient
	var err error

	switch cfg.Type {
	case ClientTypeSingleNode:
		client, err = createSingleNodeClient(cfg)
	case ClientTypeCluster:
		client, err = createClusterClient(cfg)
	case ClientTypeSentinel:
		client, err = createSentinelClient(cfg)
	case ClientTypeFailover:
		client, err = createFailoverClient(cfg)
	default:
		return nil, &ConfigError{
			Field: "Type",
			Value: cfg.Type,
			Err:   ErrInvalidClientType,
		}
	}

	if err != nil {
		return nil, &ConnectionError{
			URL:        cfg.URL,
			ClientType: cfg.Type,
			Attempt:    1,
			Err:        err,
		}
	}

	// Create client wrapper
	c := &Client{
		client:    client,
		config:    cfg,
		opts:      options,
		namespace: cfg.Namespace,
	}

	// Initialize metrics collector if enabled
	if cfg.EnableMetrics {
		c.metrics = newMetricsCollector()
	}

	// Ping the server with retries
	if err := c.pingWithRetry(ctx); err != nil {
		_ = client.Close()
		return nil, err
	}

	// Call post-connect hook if provided
	if options.OnConnect != nil {
		if err := options.OnConnect(ctx, c); err != nil {
			_ = client.Close()
			return nil, fmt.Errorf("post-connect hook failed: %w", err)
		}
	}

	return c, nil
}

// createSingleNodeClient creates a single-node Redis client
func createSingleNodeClient(cfg Config) (redis.UniversalClient, error) {
	opts, err := redis.ParseURL(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse redis URL: %w", err)
	}

	// Override with config values
	opts.Password = cfg.Password
	opts.DB = cfg.DB
	opts.MaxRetries = cfg.MaxRetries
	opts.MinRetryBackoff = cfg.MinRetryBackoff
	opts.MaxRetryBackoff = cfg.MaxRetryBackoff
	opts.DialTimeout = cfg.DialTimeout
	opts.ReadTimeout = cfg.ReadTimeout
	opts.WriteTimeout = cfg.WriteTimeout
	opts.ContextTimeoutEnabled = cfg.ContextTimeoutEnabled
	opts.PoolFIFO = cfg.PoolFIFO
	opts.PoolSize = cfg.PoolSize
	opts.PoolTimeout = cfg.PoolTimeout
	opts.MinIdleConns = cfg.MinIdleConns
	opts.MaxIdleConns = cfg.MaxIdleConns
	opts.ConnMaxIdleTime = cfg.ConnMaxIdleTime
	opts.ConnMaxLifetime = cfg.ConnMaxLifetime
	opts.Protocol = cfg.Protocol
	opts.DisableIndentity = cfg.DisableIndentity
	opts.IdentitySuffix = cfg.IdentitySuffix

	// Configure TLS
	if cfg.EnableTLS {
		opts.TLSConfig = &tls.Config{
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: cfg.TLSSkipVerify,
			ServerName:         cfg.TLSServerName,
		}
	} else if cfg.Password != "" {
		// Auto-enable TLS for password-protected connections
		opts.TLSConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
	}

	return redis.NewClient(opts), nil
}

// createClusterClient creates a Redis cluster client
func createClusterClient(cfg Config) (redis.UniversalClient, error) {
	var opts *redis.ClusterOptions

	// Parse URL if provided, otherwise use cluster addresses
	if cfg.URL != "" {
		parsedOpts, err := redis.ParseClusterURL(cfg.URL)
		if err != nil {
			return nil, fmt.Errorf("failed to parse redis cluster URL: %w", err)
		}
		opts = parsedOpts
	} else {
		opts = &redis.ClusterOptions{
			Addrs: cfg.ClusterAddrs,
		}
	}

	// Override with config values
	opts.Password = cfg.Password
	opts.MaxRetries = cfg.MaxRetries
	opts.MinRetryBackoff = cfg.MinRetryBackoff
	opts.MaxRetryBackoff = cfg.MaxRetryBackoff
	opts.DialTimeout = cfg.DialTimeout
	opts.ReadTimeout = cfg.ReadTimeout
	opts.WriteTimeout = cfg.WriteTimeout
	opts.ContextTimeoutEnabled = cfg.ContextTimeoutEnabled
	opts.PoolFIFO = cfg.PoolFIFO
	opts.PoolSize = cfg.PoolSize
	opts.PoolTimeout = cfg.PoolTimeout
	opts.MinIdleConns = cfg.MinIdleConns
	opts.MaxIdleConns = cfg.MaxIdleConns
	opts.ConnMaxIdleTime = cfg.ConnMaxIdleTime
	opts.ConnMaxLifetime = cfg.ConnMaxLifetime
	opts.MaxRedirects = cfg.MaxRedirects
	opts.ReadOnly = cfg.ReadOnly
	opts.Protocol = cfg.Protocol
	opts.DisableIndentity = cfg.DisableIndentity
	opts.IdentitySuffix = cfg.IdentitySuffix

	// Configure TLS
	if cfg.EnableTLS {
		opts.TLSConfig = &tls.Config{
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: cfg.TLSSkipVerify,
			ServerName:         cfg.TLSServerName,
		}
	} else if cfg.Password != "" {
		opts.TLSConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
	}

	return redis.NewClusterClient(opts), nil
}

// createSentinelClient creates a Redis sentinel client
func createSentinelClient(cfg Config) (redis.UniversalClient, error) {
	opts := &redis.FailoverOptions{
		MasterName:       cfg.MasterName,
		SentinelAddrs:    cfg.SentinelAddrs,
		SentinelPassword: cfg.SentinelPassword,
		SentinelUsername: cfg.SentinelUsername,
		Password:         cfg.Password,
		DB:               cfg.DB,
		RouteByLatency:   cfg.RouteByLatency,
		RouteRandomly:    cfg.RouteRandomly,
		ReplicaOnly:      cfg.ReplicaOnly,

		MaxRetries:            cfg.MaxRetries,
		MinRetryBackoff:       cfg.MinRetryBackoff,
		MaxRetryBackoff:       cfg.MaxRetryBackoff,
		DialTimeout:           cfg.DialTimeout,
		ReadTimeout:           cfg.ReadTimeout,
		WriteTimeout:          cfg.WriteTimeout,
		ContextTimeoutEnabled: cfg.ContextTimeoutEnabled,
		PoolFIFO:              cfg.PoolFIFO,
		PoolSize:              cfg.PoolSize,
		PoolTimeout:           cfg.PoolTimeout,
		MinIdleConns:          cfg.MinIdleConns,
		MaxIdleConns:          cfg.MaxIdleConns,
		ConnMaxIdleTime:       cfg.ConnMaxIdleTime,
		ConnMaxLifetime:       cfg.ConnMaxLifetime,
		Protocol:              cfg.Protocol,
		DisableIndentity:      cfg.DisableIndentity,
		IdentitySuffix:        cfg.IdentitySuffix,
	}

	// Configure TLS
	if cfg.EnableTLS {
		opts.TLSConfig = &tls.Config{
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: cfg.TLSSkipVerify,
			ServerName:         cfg.TLSServerName,
		}
	} else if cfg.Password != "" {
		opts.TLSConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
	}

	return redis.NewFailoverClient(opts), nil
}

// createFailoverClient creates a Redis failover client (alias for sentinel)
func createFailoverClient(cfg Config) (redis.UniversalClient, error) {
	opts, err := redis.ParseURL(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse redis URL: %w", err)
	}

	failoverOpts := &redis.FailoverOptions{
		MasterName:            cfg.MasterName,
		Password:              cfg.Password,
		DB:                    cfg.DB,
		MaxRetries:            cfg.MaxRetries,
		MinRetryBackoff:       cfg.MinRetryBackoff,
		MaxRetryBackoff:       cfg.MaxRetryBackoff,
		DialTimeout:           cfg.DialTimeout,
		ReadTimeout:           cfg.ReadTimeout,
		WriteTimeout:          cfg.WriteTimeout,
		ContextTimeoutEnabled: cfg.ContextTimeoutEnabled,
		PoolFIFO:              cfg.PoolFIFO,
		PoolSize:              cfg.PoolSize,
		PoolTimeout:           cfg.PoolTimeout,
		MinIdleConns:          cfg.MinIdleConns,
		MaxIdleConns:          cfg.MaxIdleConns,
		ConnMaxIdleTime:       cfg.ConnMaxIdleTime,
		ConnMaxLifetime:       cfg.ConnMaxLifetime,
		Protocol:              cfg.Protocol,
		DisableIndentity:      cfg.DisableIndentity,
		IdentitySuffix:        cfg.IdentitySuffix,
	}

	// Copy address from parsed URL
	if opts.Addr != "" {
		failoverOpts.SentinelAddrs = []string{opts.Addr}
	}

	// Configure TLS
	if cfg.EnableTLS {
		failoverOpts.TLSConfig = &tls.Config{
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: cfg.TLSSkipVerify,
			ServerName:         cfg.TLSServerName,
		}
	} else if cfg.Password != "" {
		failoverOpts.TLSConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
	}

	return redis.NewFailoverClient(failoverOpts), nil
}

// pingWithRetry attempts to ping the Redis server with retries
func (c *Client) pingWithRetry(ctx context.Context) error {
	var lastErr error
	maxRetries := c.config.MaxRetries
	if maxRetries == 0 {
		maxRetries = 3
	}

	for attempt := 1; attempt <= maxRetries; attempt++ {
		pingCtx, cancel := context.WithTimeout(ctx, c.config.DialTimeout)
		err := c.client.Ping(pingCtx).Err()
		cancel()

		if err == nil {
			return nil
		}

		lastErr = err

		// Call retry callback if provided
		if c.opts.OnRetry != nil {
			c.opts.OnRetry(attempt, err)
		}

		// Don't sleep after the last attempt
		if attempt < maxRetries {
			backoff := c.calculateBackoff(attempt)
			time.Sleep(backoff)
		}
	}

	return &ConnectionError{
		URL:        c.config.URL,
		ClientType: c.config.Type,
		Attempt:    maxRetries,
		Err:        lastErr,
	}
}

// calculateBackoff calculates the backoff duration for retry attempts
func (c *Client) calculateBackoff(attempt int) time.Duration {
	min := c.config.MinRetryBackoff
	max := c.config.MaxRetryBackoff

	if min == 0 {
		min = 8 * time.Millisecond
	}
	if max == 0 {
		max = 512 * time.Millisecond
	}

	backoff := min * time.Duration(1<<uint(attempt-1))
	if backoff > max {
		backoff = max
	}

	return backoff
}

// UniversalClient returns the underlying Redis universal client
func (c *Client) UniversalClient() redis.UniversalClient {
	return c.client
}

// Ping checks the connection to Redis server
func (c *Client) Ping(ctx context.Context) error {
	if c.isClosed {
		return ErrClientClosed
	}
	return c.client.Ping(ctx).Err()
}

// Close closes the Redis client connection
func (c *Client) Close() error {
	if c.isClosed {
		return ErrClientClosed
	}
	c.isClosed = true
	return c.client.Close()
}

// Stats returns the connection pool statistics
func (c *Client) Stats() *PoolStats {
	if c.isClosed {
		return nil
	}

	var stats *redis.PoolStats
	switch client := c.client.(type) {
	case *redis.Client:
		stats = client.PoolStats()
	case *redis.ClusterClient:
		// For cluster, get stats from the first node
		stats = client.PoolStats()
	}

	if stats == nil {
		return nil
	}

	return &PoolStats{
		Hits:       stats.Hits,
		Misses:     stats.Misses,
		Timeouts:   stats.Timeouts,
		TotalConns: stats.TotalConns,
		IdleConns:  stats.IdleConns,
		StaleConns: stats.StaleConns,
	}
}

// WithNamespace returns a namespaced key
func (c *Client) WithNamespace(key string) string {
	if c.namespace == "" {
		return key
	}
	return c.namespace + ":" + key
}

// WithoutNamespace removes the namespace from a key
func (c *Client) WithoutNamespace(key string) string {
	if c.namespace == "" {
		return key
	}
	prefix := c.namespace + ":"
	return strings.TrimPrefix(key, prefix)
}

// Config returns the client configuration
func (c *Client) Config() Config {
	return c.config
}

// IsClosed returns whether the client is closed
func (c *Client) IsClosed() bool {
	return c.isClosed
}

// GetMetrics returns collected metrics (if enabled)
func (c *Client) GetMetrics() *Metrics {
	if c.metrics == nil {
		return nil
	}
	return c.metrics.get()
}
