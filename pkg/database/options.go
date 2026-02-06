package database

import "time"

// options holds internal configuration options
type options struct {
	ConnectTimeout  time.Duration
	MaxRetries      int
	RetryDelay      time.Duration
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
	OnRetry         func(attempt int, err error)
}

// Option is a functional option for configuring the database client
type Option func(*options)

// defaultOptions returns the default options
func defaultOptions() options {
	return options{
		ConnectTimeout:  defaultConnectTimeout,
		MaxRetries:      defaultMaxRetries,
		RetryDelay:      defaultRetryDelay,
		ConnMaxLifetime: defaultConnMaxLifetime,
		ConnMaxIdleTime: defaultConnMaxIdleTime,
	}
}

// WithConnectTimeout sets the connection timeout
func WithConnectTimeout(timeout time.Duration) Option {
	return func(o *options) {
		o.ConnectTimeout = timeout
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
		o.RetryDelay = delay
	}
}

// WithConnMaxLifetime sets the maximum lifetime of a connection
func WithConnMaxLifetime(duration time.Duration) Option {
	return func(o *options) {
		o.ConnMaxLifetime = duration
	}
}

// WithConnMaxIdleTime sets the maximum idle time of a connection
func WithConnMaxIdleTime(duration time.Duration) Option {
	return func(o *options) {
		o.ConnMaxIdleTime = duration
	}
}

// WithRetryCallback sets a callback function to be called on each retry attempt
func WithRetryCallback(callback func(attempt int, err error)) Option {
	return func(o *options) {
		o.OnRetry = callback
	}
}
