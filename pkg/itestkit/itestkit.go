package itestkit

import (
	"context"
	"fmt"
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
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/stretchr/testify/require"
	testcontainers "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
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
					WithOccurrence(2).WithStartupTimeout(30 * time.Second),
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
		k.migrateDSN = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
			k.config.User, k.config.Password, host, port.Port(), k.config.Database)

		for i := 0; i < 10; i++ {
			k.db, initErr = gorm.Open(postgres.Open(k.dsn), &gorm.Config{})
			if initErr == nil {
				break
			}
			time.Sleep(500 * time.Millisecond)
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

		for i := 0; i < 10; i++ {
			_, initErr = k.redis.Ping(ctx).Result()
			if initErr == nil {
				break
			}
			time.Sleep(500 * time.Millisecond)
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
		if err = m.Up(); err != nil && err != migrate.ErrNoChange {
			initErr = fmt.Errorf("run migrations: %w", err)
		}
		m.Close()
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
	if k.db != nil {
		if sqlDB, _ := k.db.DB(); sqlDB != nil {
			sqlDB.Close()
		}
	}
	if k.pgContainer != nil {
		testcontainers.TerminateContainer(k.pgContainer)
	}
}

// StopRedis stops the Redis container.
func (k *ITestKit) StopRedis() {
	if k.redis != nil {
		k.redis.Close()
	}
	if k.redisContainer != nil {
		testcontainers.TerminateContainer(k.redisContainer)
	}
}

// Cleanup stops all containers and cleans up resources.
// This is a convenience method that calls StopPostgres and StopRedis.
func (k *ITestKit) Cleanup() {
	if k.db != nil {
		if sqlDB, _ := k.db.DB(); sqlDB != nil {
			sqlDB.Close()
		}
	}
	if k.pgContainer != nil {
		testcontainers.TerminateContainer(k.pgContainer)
	}
	if k.redis != nil {
		k.redis.Close()
	}
	if k.redisContainer != nil {
		testcontainers.TerminateContainer(k.redisContainer)
	}
}

// CleanupAll removes all testcontainers Docker containers.
// This should be called in TestMain as a safety net to ensure no orphaned containers remain.
// It finds and removes any containers created by the testcontainers library.
func CleanupAll() {
	fmt.Println("[itestkit] Running cleanup for orphaned Docker containers...")

	// Find all testcontainers containers (both running and stopped)
	cmd := exec.Command("docker", "ps", "-a", "-q", "--filter", "label=org.testcontainers")
	output, err := cmd.Output()
	if err != nil || len(output) == 0 {
		fmt.Println("[itestkit] No testcontainers found, cleanup complete")
		return
	}

	containerIDs := strings.TrimSpace(string(output))
	if containerIDs == "" {
		fmt.Println("[itestkit] No testcontainers found, cleanup complete")
		return
	}

	// Stop containers
	fmt.Printf("[itestkit] Stopping %d testcontainer(s)...\n", len(strings.Fields(containerIDs)))
	stopCmd := exec.Command("docker", "stop", containerIDs)
	_ = stopCmd.Run()

	// Remove containers
	fmt.Printf("[itestkit] Removing %d testcontainer(s)...\n", len(strings.Fields(containerIDs)))
	rmCmd := exec.Command("docker", "rm", "-f", containerIDs)
	_ = rmCmd.Run()

	fmt.Println("[itestkit] Cleanup completed successfully")
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
	// Set up signal handlers for cleanup
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Run tests in a goroutine to handle both normal exit and signals
	testDone := make(chan int, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("[itestkit] Panic recovered during test execution: %v\n", r)
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
		fmt.Printf("[itestkit] Received signal %v, cleaning up...\n", sig)
		code = 1
	}

	// Run cleanup BEFORE calling os.Exit
	fmt.Println("[itestkit] Running cleanup for testcontainers...")
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
