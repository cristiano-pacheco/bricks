package redis

import (
	"sync"
	"sync/atomic"
	"time"
)

// PoolStats represents Redis connection pool statistics
type PoolStats struct {
	Hits       uint32 // Number of times free connection was found in the pool
	Misses     uint32 // Number of times free connection was NOT found in the pool
	Timeouts   uint32 // Number of times a wait timeout occurred
	TotalConns uint32 // Number of total connections in the pool
	IdleConns  uint32 // Number of idle connections in the pool
	StaleConns uint32 // Number of stale connections removed from the pool
}

// Metrics represents collected metrics for Redis operations
type Metrics struct {
	CommandsExecuted  uint64        // Total number of commands executed
	CommandsFailed    uint64        // Total number of failed commands
	TotalLatency      time.Duration // Total latency of all commands
	AverageLatency    time.Duration // Average latency per command
	LastCommandTime   time.Time     // Time of last command execution
	ConnectionRetries uint64        // Number of connection retry attempts
	ConnectionErrors  uint64        // Number of connection errors
}

// metricsCollector collects metrics for Redis operations
type metricsCollector struct {
	mu                sync.RWMutex
	commandsExecuted  uint64
	commandsFailed    uint64
	totalLatency      time.Duration
	lastCommandTime   time.Time
	connectionRetries uint64
	connectionErrors  uint64
}

// newMetricsCollector creates a new metrics collector
func newMetricsCollector() *metricsCollector {
	return &metricsCollector{}
}

// recordCommand records a command execution
func (m *metricsCollector) recordCommand(duration time.Duration, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	atomic.AddUint64(&m.commandsExecuted, 1)
	m.totalLatency += duration
	m.lastCommandTime = time.Now()

	if err != nil {
		atomic.AddUint64(&m.commandsFailed, 1)
	}
}

// recordRetry records a connection retry attempt
func (m *metricsCollector) recordRetry() {
	atomic.AddUint64(&m.connectionRetries, 1)
}

// recordConnectionError records a connection error
func (m *metricsCollector) recordConnectionError() {
	atomic.AddUint64(&m.connectionErrors, 1)
}

// get returns the current metrics
func (m *metricsCollector) get() *Metrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	commandsExecuted := atomic.LoadUint64(&m.commandsExecuted)
	averageLatency := time.Duration(0)
	if commandsExecuted > 0 {
		const maxInt64 = int64(^uint64(0) >> 1)
		divisor := int64(commandsExecuted)
		if commandsExecuted > uint64(maxInt64) {
			divisor = maxInt64
		}
		averageLatency = m.totalLatency / time.Duration(divisor)
	}

	return &Metrics{
		CommandsExecuted:  commandsExecuted,
		CommandsFailed:    atomic.LoadUint64(&m.commandsFailed),
		TotalLatency:      m.totalLatency,
		AverageLatency:    averageLatency,
		LastCommandTime:   m.lastCommandTime,
		ConnectionRetries: atomic.LoadUint64(&m.connectionRetries),
		ConnectionErrors:  atomic.LoadUint64(&m.connectionErrors),
	}
}

// reset resets all metrics
func (m *metricsCollector) reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	atomic.StoreUint64(&m.commandsExecuted, 0)
	atomic.StoreUint64(&m.commandsFailed, 0)
	m.totalLatency = 0
	m.lastCommandTime = time.Time{}
	atomic.StoreUint64(&m.connectionRetries, 0)
	atomic.StoreUint64(&m.connectionErrors, 0)
}
