# Integration Test Kit (itestkit)

The `itestkit` package provides integration test infrastructure for Docker containers (PostgreSQL and Redis) with automatic cleanup.

## Features

- **Automatic Container Management**: Start/stop PostgreSQL and Redis containers for integration tests
- **Database Migrations**: Apply database migrations automatically
- **Automatic Cleanup**: Ensures all containers are cleaned up even if tests panic or fail
- **Testify Suite Support**: Works seamlessly with `testify/suite`

## Usage

### Basic Test File Setup

For integration tests using `testify/suite`, simply use `itestkit.TestMain`:

```go
//go:build integration

package user_test

import (
    "testing"
    "github.com/stretchr/testify/suite"
    "github.com/cristiano-pacheco/pingo/pkg/itestkit"
)

type UserRegisterUseCaseTestSuite struct {
    suite.Suite
    kit *itestkit.ITestKit
    // ... other fields
}

func (s *UserRegisterUseCaseTestSuite) SetupSuite() {
    s.kit = itestkit.New(itestkit.Config{
        PostgresImage:  "postgres:16-alpine",
        RedisImage:     "redis:7-alpine",
        MigrationsPath: "file://migrations",
        Database:       "pingo_test",
        User:           "pingo_test",
        Password:       "pingo_test",
    })

    err := s.kit.StartPostgres()
    s.Require().NoError(err)

    err = s.kit.RunMigrations()
    s.Require().NoError(err)
}

func (s *UserRegisterUseCaseTestSuite) TearDownSuite() {
    if s.kit != nil {
        if err := s.kit.StopPostgres(); err != nil {
            s.T().Logf("Error stopping PostgreSQL: %v", err)
        }
    }
}

func TestUserRegisterUseCaseSuite(t *testing.T) {
    suite.Run(t, new(UserRegisterUseCaseTestSuite))
}

// Use itestkit.TestMain for automatic cleanup
func TestMain(m *testing.M) {
    itestkit.TestMain(m)
}
```

### How Automatic Cleanup Works

The `itestkit.TestMain` function provides automatic cleanup that:

1. **Runs on normal exit**: Cleans up containers when tests complete successfully
2. **Runs on test failure**: Cleans up containers when tests fail
3. **Runs on panic**: Recovers from panics and cleans up containers
4. **Cleans orphaned containers**: Uses the `org.testcontainers` label to find and remove any orphaned containers

This ensures that no matter what happens during test execution, all Docker containers will be cleaned up properly.

### Manual Cleanup Options

If you need more control over cleanup:

```go
func TestMain(m *testing.M) {
    defer func() {
        if r := recover(); r != nil {
            fmt.Printf("Panic recovered: %v\n", r)
        }
        itestkit.CleanupAll() // Manually call cleanup
    }()

    code := m.Run()
    os.Exit(code)
}
```

### ITestKit Methods

#### Starting Containers

- `StartPostgres() error`: Starts a PostgreSQL container
- `StartRedis() error`: Starts a Redis container

#### Running Migrations

- `RunMigrations() error`: Applies database migrations from the configured path

#### Stopping Containers

- `StopPostgres()`: Stops the PostgreSQL container
- `StopRedis()`: Stops the Redis container
- `Cleanup()`: Stops all containers (PostgreSQL and Redis) and closes connections

#### Database Operations

- `DB() *gorm.DB`: Returns the GORM database connection
- `TruncateTables(t *testing.T)`: Truncates all tables except `schema_migrations` (useful between tests)

#### Redis Operations

- `Redis() redis.UniversalClient`: Returns the Redis client

### Configuration

Use `itestkit.New()` with a `Config` struct:

```go
type Config struct {
    PostgresImage  string  // Docker image for PostgreSQL (default: postgres:16-alpine)
    RedisImage     string  // Docker image for Redis (default: redis:7-alpine)
    MigrationsPath string  // Path to migrations (e.g., "file://migrations")
    Database       string  // Database name
    User           string  // Database user
    Password       string  // Database password
}
```

Or use the default configuration:

```go
kit := itestkit.New(itestkit.DefaultConfig())
```

## Important Notes

- **Build Tag**: All integration test files should use the `//go:build integration` tag
- **Cleanup Priority**: The normal cleanup path (`TearDownSuite`) is preferred. The `itestkit.TestMain` cleanup is a safety net.
- **Container Labels**: The cleanup function uses the `org.testcontainers` label to find containers created by the testcontainers library.

## Example Test Structure

```
test/integration/
└── modules/
    └── identity/
        └── usecase/
            └── user/
                └── user_register_usecase_test.go
```

Each integration test file follows this pattern:

1. `SetupSuite()`: Initialize ITestKit, start containers, run migrations
2. `TearDownSuite()`: Stop containers and clean up
3. `SetupTest()`: Truncate tables and reset state between individual tests
4. Test methods: Use `s.T()`, `s.Require()`, `s.Assert()` from testify/suite
5. `TestMain(m)`: Call `itestkit.TestMain(m)` for automatic cleanup
