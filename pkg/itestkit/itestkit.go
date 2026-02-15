package itestkit

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres" // Register postgres driver for migrations.
	_ "github.com/golang-migrate/migrate/v4/source/file"       // Register file source driver for migrations.
	"github.com/stretchr/testify/require"
	testcontainers "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	containerStartupTimeout = 30 * time.Second
	retryDelay              = 500 * time.Millisecond
	connectionRetryAttempts = 10
	cleanupTimeout          = 15 * time.Second
	postgresReadyLogCount   = 2
)

// ITestKit manages integration test infrastructure.
// Use individual methods to start/stop only what you need.
type ITestKit struct {
	config         Config
	db             *gorm.DB
	redis          redis.UniversalClient
	dsn            string
	migrateDSN     string
	pgContainer    testcontainers.Container
	redisContainer testcontainers.Container
	pgOnce         *sync.Once
	redisOnce      *sync.Once
	migrateOnce    *sync.Once
}

// New creates an ITestKit with the given configuration.
func New(config Config) *ITestKit {
	if config.PostgresImage == "" {
		config = DefaultConfig()
	}
	return &ITestKit{
		config:      config,
		pgOnce:      &sync.Once{},
		redisOnce:   &sync.Once{},
		migrateOnce: &sync.Once{},
	}
}

// StartPostgres starts the PostgreSQL container.
// Returns error if container fails to start.
func (k *ITestKit) StartPostgres() error {
	var initErr error
	k.pgOnce.Do(func() {
		ctx := context.Background()
		c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
			ContainerRequest: testcontainers.ContainerRequest{
				Image:        k.config.PostgresImage,
				ExposedPorts: []string{"5432/tcp"},
				Env: map[string]string{
					"POSTGRES_DB":       k.config.Database,
					"POSTGRES_USER":     k.config.User,
					"POSTGRES_PASSWORD": k.config.Password,
				},
				WaitingFor: wait.ForLog("database system is ready to accept connections").
					WithOccurrence(postgresReadyLogCount).WithStartupTimeout(containerStartupTimeout),
			},
			Started: true,
		})
		if err != nil {
			initErr = fmt.Errorf("start postgres container: %w", err)
			return
		}
		k.pgContainer = c

		host, hostErr := c.Host(ctx)
		if hostErr != nil {
			initErr = fmt.Errorf("get container host: %w", hostErr)
			return
		}
		port, portErr := c.MappedPort(ctx, "5432")
		if portErr != nil {
			initErr = fmt.Errorf("get container port: %w", portErr)
			return
		}
		k.dsn = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
			host, port.Port(), k.config.User, k.config.Password, k.config.Database)
		k.migrateDSN = (&url.URL{
			Scheme: "postgres",
			User:   url.UserPassword(k.config.User, k.config.Password),
			Host:   net.JoinHostPort(host, port.Port()),
			Path:   "/" + k.config.Database,
			RawQuery: (&url.Values{
				"sslmode": []string{"disable"},
			}).Encode(),
		}).String()

		for range connectionRetryAttempts {
			k.db, initErr = gorm.Open(postgres.Open(k.dsn), &gorm.Config{})
			if initErr == nil {
				break
			}
			time.Sleep(retryDelay)
		}
		if initErr != nil {
			initErr = fmt.Errorf("connect to database: %w", initErr)
		}
	})
	return initErr
}

// StartRedis starts the Redis container.
// Returns error if container fails to start.
func (k *ITestKit) StartRedis() error {
	var initErr error
	k.redisOnce.Do(func() {
		ctx := context.Background()
		c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
			ContainerRequest: testcontainers.ContainerRequest{
				Image:        k.config.RedisImage,
				ExposedPorts: []string{"6379/tcp"},
				WaitingFor:   wait.ForLog("Ready to accept connections"),
			},
			Started: true,
		})
		if err != nil {
			initErr = fmt.Errorf("start redis container: %w", err)
			return
		}
		k.redisContainer = c

		host, hostErr := c.Host(ctx)
		if hostErr != nil {
			initErr = fmt.Errorf("get redis host: %w", hostErr)
			return
		}
		port, portErr := c.MappedPort(ctx, "6379")
		if portErr != nil {
			initErr = fmt.Errorf("get redis port: %w", portErr)
			return
		}
		addr := fmt.Sprintf("%s:%s", host, port.Port())
		k.redis = redis.NewClient(&redis.Options{Addr: addr})

		for range connectionRetryAttempts {
			_, initErr = k.redis.Ping(ctx).Result()
			if initErr == nil {
				break
			}
			time.Sleep(retryDelay)
		}
		if initErr != nil {
			initErr = fmt.Errorf("connect to redis: %w", initErr)
		}
	})
	return initErr
}

// RunMigrations applies migrations from the configured path.
// Must call StartPostgres() first.
func (k *ITestKit) RunMigrations() error {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	var initErr error
	k.migrateOnce.Do(func() {
		migrationsPath := k.config.MigrationsPath
		if strings.HasPrefix(migrationsPath, "file://") {
			relativePath := strings.TrimPrefix(migrationsPath, "file://")
			if !filepath.IsAbs(relativePath) {
				absolutePath := filepath.Join(getProjectRoot(), relativePath)
				migrationsPath = "file://" + absolutePath
			}
		}

		m, err := migrate.New(migrationsPath, k.migrateDSN)
		if err != nil {
			initErr = fmt.Errorf("create migrate instance: %w", err)
			return
		}
		defer func() {
			srcErr, dbErr := m.Close()
			if srcErr != nil {
				logger.Warn("[itestkit] close migration source", "error", srcErr)
			}
			if dbErr != nil {
				logger.Warn("[itestkit] close migration database", "error", dbErr)
			}
		}()
		if err = m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			initErr = fmt.Errorf("run migrations: %w", err)
		}
	})
	return initErr
}

func getProjectRoot() string {
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)
	for {
		goModPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

// TruncateTables truncates all tables except schema_migrations.
// Must call StartPostgres() first.
func (k *ITestKit) TruncateTables(t *testing.T) {
	t.Helper()
	rows, err := k.db.Raw(`
		SELECT table_name FROM information_schema.tables
		WHERE table_schema = 'public' AND table_type = 'BASE TABLE' AND table_name != 'schema_migrations'
	`).Rows()
	require.NoError(t, err)
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		require.NoError(t, rows.Scan(&name))
		tables = append(tables, name)
	}
	if len(tables) > 0 {
		require.NoError(
			t,
			k.db.Exec(fmt.Sprintf("TRUNCATE TABLE %s RESTART IDENTITY CASCADE", strings.Join(tables, ", "))).Error,
		)
	}
}

// StopPostgres stops the PostgreSQL container.
func (k *ITestKit) StopPostgres() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	if k.db != nil {
		if sqlDB, _ := k.db.DB(); sqlDB != nil {
			if err := sqlDB.Close(); err != nil {
				logger.Warn("[itestkit] close postgres sql db", "error", err)
			}
		}
	}
	if k.pgContainer != nil {
		if err := testcontainers.TerminateContainer(k.pgContainer); err != nil {
			logger.Warn("[itestkit] terminate postgres container", "error", err)
		}
	}
}

// StopRedis stops the Redis container.
func (k *ITestKit) StopRedis() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	if k.redis != nil {
		if err := k.redis.Close(); err != nil {
			logger.Warn("[itestkit] close redis client", "error", err)
		}
	}
	if k.redisContainer != nil {
		if err := testcontainers.TerminateContainer(k.redisContainer); err != nil {
			logger.Warn("[itestkit] terminate redis container", "error", err)
		}
	}
}

// Cleanup stops all containers and cleans up resources.
// This is a convenience method that calls StopPostgres and StopRedis.
func (k *ITestKit) Cleanup() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	if k.db != nil {
		if sqlDB, _ := k.db.DB(); sqlDB != nil {
			if err := sqlDB.Close(); err != nil {
				logger.Warn("[itestkit] close postgres sql db", "error", err)
			}
		}
	}
	if k.pgContainer != nil {
		if err := testcontainers.TerminateContainer(k.pgContainer); err != nil {
			logger.Warn("[itestkit] terminate postgres container", "error", err)
		}
	}
	if k.redis != nil {
		if err := k.redis.Close(); err != nil {
			logger.Warn("[itestkit] close redis client", "error", err)
		}
	}
	if k.redisContainer != nil {
		if err := testcontainers.TerminateContainer(k.redisContainer); err != nil {
			logger.Warn("[itestkit] terminate redis container", "error", err)
		}
	}
}

// CleanupAll removes all testcontainers Docker containers.
// This should be called in TestMain as a safety net to ensure no orphaned containers remain.
// It finds and removes any containers created by the testcontainers library.
func CleanupAll() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	logger.Info("[itestkit] Running cleanup for orphaned Docker containers...")
	ctx, cancel := context.WithTimeout(context.Background(), cleanupTimeout)
	defer cancel()

	// Find all testcontainers containers (both running and stopped)
	cmd := exec.CommandContext(ctx, "docker", "ps", "-a", "-q", "--filter", "label=org.testcontainers")
	output, err := cmd.Output()
	if err != nil || len(output) == 0 {
		logger.Info("[itestkit] No testcontainers found, cleanup complete")
		return
	}

	containerIDs := strings.TrimSpace(string(output))
	if containerIDs == "" {
		logger.Info("[itestkit] No testcontainers found, cleanup complete")
		return
	}

	ids := strings.Fields(containerIDs)

	// Stop containers
	logger.Info("[itestkit] Stopping testcontainers", "count", len(ids))
	stopArgs := append([]string{"stop"}, ids...)
	stopCmd := exec.CommandContext(ctx, "docker", stopArgs...)
	_ = stopCmd.Run()

	// Remove containers
	logger.Info("[itestkit] Removing testcontainers", "count", len(ids))
	rmArgs := append([]string{"rm", "-f"}, ids...)
	rmCmd := exec.CommandContext(ctx, "docker", rmArgs...)
	_ = rmCmd.Run()

	logger.Info("[itestkit] Cleanup completed successfully")
}

// TestMain is a convenience wrapper for integration test TestMain functions.
// It runs the provided test runner with automatic cleanup registered.
// Cleanup happens even if tests panic or fail.
// Usage in test files:
//
//	func TestMain(m *testing.M) {
//	    itestkit.TestMain(m)
//	}
func TestMain(m *testing.M) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	// Set up signal handlers for cleanup
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Run tests in a goroutine to handle both normal exit and signals
	testDone := make(chan int, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("[itestkit] Panic recovered during test execution", "panic", r)
				testDone <- 1
			}
		}()
		testDone <- m.Run()
	}()

	// Wait for either test completion or signal
	var code int
	select {
	case code = <-testDone:
		// Tests completed normally
	case sig := <-sigChan:
		// Received interrupt signal
		logger.Info("[itestkit] Received signal, cleaning up...", "signal", sig.String())
		code = 1
	}

	// Run cleanup BEFORE calling os.Exit
	logger.Info("[itestkit] Running cleanup for testcontainers...")
	CleanupAll()

	os.Exit(code)
}

// DB returns the GORM database connection.
func (k *ITestKit) DB() *gorm.DB {
	return k.db
}

// Redis returns the Redis client.
func (k *ITestKit) Redis() redis.UniversalClient {
	return k.redis
}
