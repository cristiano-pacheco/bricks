package redis

import (
	"context"
	"time"
)

// options holds internal configuration options
type options struct {
	ConnectTimeout time.Duration
	MaxRetries     int
	RetryDelay     time.Duration
	OnRetry        func(attempt int, err error)
	OnConnect      func(ctx context.Context, client *Client) error
	OnDisconnect   func(client *Client) error
}

// Option is a functional option for configuring the Redis client
type Option func(*options)

// Default values
const (
	defaultConnectTimeout = 30 * time.Second
	defaultRetryDelay     = 1 * time.Second
)

// defaultOptions returns the default options
func defaultOptions() options {
	return options{
		ConnectTimeout: defaultConnectTimeout,
		MaxRetries:     defaultMaxRetries,
		RetryDelay:     defaultRetryDelay,
	}
}

// WithConnectTimeout sets the connection timeout
func WithConnectTimeout(timeout time.Duration) Option {
	return func(o *options) {
		if timeout > 0 {
			o.ConnectTimeout = timeout
		}
	}
}

// WithMaxRetries sets the maximum number of connection retry attempts
func WithMaxRetries(maxRetries int) Option {
	return func(o *options) {
		if maxRetries > 0 {
			o.MaxRetries = maxRetries
		}
	}
}

// WithRetryDelay sets the base delay between retry attempts
func WithRetryDelay(delay time.Duration) Option {
	return func(o *options) {
		if delay > 0 {
			o.RetryDelay = delay
		}
	}
}

// WithRetryCallback sets a callback function to be called on each retry attempt
func WithRetryCallback(callback func(attempt int, err error)) Option {
	return func(o *options) {
		o.OnRetry = callback
	}
}

// WithOnConnect sets a callback function to be called after successful connection
func WithOnConnect(callback func(ctx context.Context, client *Client) error) Option {
	return func(o *options) {
		o.OnConnect = callback
	}
}

// WithOnDisconnect sets a callback function to be called before disconnection
func WithOnDisconnect(callback func(client *Client) error) Option {
	return func(o *options) {
		o.OnDisconnect = callback
	}
}
