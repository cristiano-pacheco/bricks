package database

import "time"

// ConnectionStats represents database connection statistics
type ConnectionStats struct {
	MaxOpenConnections int           // Maximum number of open connections to the database
	OpenConnections    int           // The number of established connections both in use and idle
	InUse              int           // The number of connections currently in use
	Idle               int           // The number of idle connections
	WaitCount          int64         // The total number of connections waited for
	WaitDuration       time.Duration // The total time blocked waiting for a new connection
	MaxIdleClosed      int64         // The total number of connections closed due to SetMaxIdleConns
	MaxLifetimeClosed  int64         // The total number of connections closed due to SetConnMaxLifetime
}
