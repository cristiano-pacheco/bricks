# Test a PostgreSQL application with itestkit

Run an Fx application against a disposable PostgreSQL container. Its public service returns `hello Ada` from a database query and rejects an empty name. This path needs no schema migrations or Redis.

You need a consumer Go module with Bricks already selected, compatible local Go tooling, and a running Docker daemon accessible to testcontainers. The default image is `postgres:16-alpine`. Docker may pull it if it is not present. This package has no YAML contract or Fx module. Container settings are Go values.

## Set up the test

1. Create `test/integration/postgresql/application_test.go` in your application. Copy the complete file below, or copy the matching [distributed example](../../examples/integration-testing-postgresql/application_test.go). Keep the `integration` build tag. The service validates its input before querying PostgreSQL.

```go
//go:build integration

package integrationtesting_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/cristiano-pacheco/bricks/pkg/itestkit"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

type greetingService struct{ db *gorm.DB }

func (s *greetingService) Execute(ctx context.Context, name string) (string, error) {
	if name == "" {
		return "", errors.New("name is required")
	}
	var greeting string
	if err := s.db.WithContext(ctx).Raw("SELECT 'hello ' || ?::text", name).Scan(&greeting).Error; err != nil {
		return "", fmt.Errorf("read greeting: %w", err)
	}
	return greeting, nil
}

func exercise() (err error) {
	kit := itestkit.New(itestkit.DefaultConfig())
	defer kit.Cleanup()
	if err := kit.StartPostgres(); err != nil {
		return err
	}
	var service *greetingService
	app := fx.New(
		fx.Provide(func() *gorm.DB { return kit.DB() }),
		fx.Provide(func(db *gorm.DB) *greetingService { return &greetingService{db: db} }),
		fx.Populate(&service),
	)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err = errors.Join(err, app.Stop(ctx))
	}()
	if err := app.Err(); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := app.Start(ctx); err != nil {
		return err
	}
	greeting, err := service.Execute(ctx, "Ada")
	if err != nil {
		return err
	}
	if greeting != "hello Ada" {
		return fmt.Errorf("unexpected greeting %q", greeting)
	}
	if _, err := service.Execute(ctx, ""); err == nil {
		return errors.New("empty name did not fail")
	}
	return nil
}

func TestPostgresApplication(t *testing.T) {
	if err := exercise(); err != nil {
		t.Error(err)
	}
}
```

2. Keep the kit's cleanup outside the Fx application. `defer kit.Cleanup()` is registered before startup, so a returned setup error still attempts cleanup. The application borrows `kit.DB()` and must not close it. The deferred calls stop Fx before the kit closes its SQL pool and terminates its container. `app.Stop` errors join the operation error before `t.Error` reports it. Kit cleanup methods log errors and return no error, so this test cannot assert termination success.

3. From your application's module root, run:

```sh
go test -tags integration -count=1 -v ./test/integration/postgresql
```

Expected result is `PASS`. `TestPostgresApplication` expects `hello Ada` and an error for an empty name. These are expected results, not recorded execution results. To run the distributed example in a Bricks checkout instead, use `go test -tags integration -count=1 -v ./skills/bricks/examples/integration-testing-postgresql`.

## What this test covers

The kit creates a GORM PostgreSQL connection through the upstream `gorm.io/driver/postgres` driver. Fx supplies that connection to the application's service. The test does not call `database.New`, `database.NewWithLifecycle`, or `database.Module`. It does not verify Bricks PostgreSQL configuration, pool settings, or constructor retries. For tests that need those constructors, obtain an explicit application `database.Config` from your fixture infrastructure and install the constructor under test. The kit exposes `DB()` but has no exported PostgreSQL host, port, or DSN accessor.

Without Fx, construct `service := &greetingService{db: kit.DB()}` after `StartPostgres` and call `Execute`. The kit still owns the database and container.

## Configure the kit in Go

Start with `cfg := itestkit.DefaultConfig()` and override individual fields. `itestkit.New` replaces the entire supplied configuration with defaults when `PostgresImage` is empty. It does not merge missing fields. Even a Redis-only setup should start from `DefaultConfig`.

```go
cfg := itestkit.Config{
    PostgresImage: "postgres:16-alpine",
    RedisImage: "redis:7-alpine",
    MigrationsPath: "file://migrations",
    Database: "itest",
    User: "itest",
    Password: "itest",
}
kit := itestkit.New(cfg)
```

This is a configuration fragment. Replace the `kit :=` line in `exercise` with it. All six fields are strings. There are no YAML keys, functional options, or configurable timeout fields in this package.

| Field | Default and constraints |
| --- | --- |
| `PostgresImage` | `postgres:16-alpine`. Docker image used only by `StartPostgres`. An empty string makes `New` discard all supplied fields and use defaults. |
| `RedisImage` | `redis:7-alpine`. Used only by `StartRedis`. It has no independent empty-value fallback when `PostgresImage` is set. |
| `MigrationsPath` | `file://migrations`. Used only by explicit `RunMigrations`. Relative `file://` paths resolve from the nearest ancestor containing `go.mod`. Use an absolute file URL when the fixture lives elsewhere. |
| `Database` | `itest`. PostgreSQL database name passed to `POSTGRES_DB` and the connection string. Set it for PostgreSQL setups. |
| `User` | `itest`. PostgreSQL user passed to `POSTGRES_USER` and the connection string. |
| `Password` | `itest`. Passed to `POSTGRES_PASSWORD` and the connection string. These are disposable test credentials. The default image requires initialization credentials. |

The kit does not validate these fields before creating containers. PostgreSQL's connection string uses unescaped interpolation, so use simple test credentials without spaces or connection-string syntax. Redis starts without configured authentication or TLS. Changing an image does not add custom server arguments, credentials, or topology support.

## Lifecycle, errors, and isolation

- `New(Config) *ITestKit` allocates a kit but starts nothing. `DB()` and `Redis()` return nil before their respective successful startup.
- `StartPostgres() error` starts one mapped container and opens one GORM connection. It waits for two PostgreSQL readiness log messages with a 30-second startup timeout, then tries the database connection up to ten times with 500-millisecond sleeps. It takes no context argument.
- `StartRedis() error` starts one standalone Redis container, waits for its readiness log, and retries `PING` up to ten times with 500-millisecond sleeps. It takes no context argument.
- `RunMigrations() error` is explicit, not automatic. Start PostgreSQL first. The [migration setup](migrations.md) shows real SQL and an absolute source URL.
- Start methods and migrations each use `sync.Once`. A failed first call consumes its once guard, and a later call can return nil without retrying. On failure, clean up and create a new kit rather than retrying the same instance. Stopping does not reset these guards.
- `StopPostgres()`, `StopRedis()`, and `Cleanup()` return no value. `Cleanup` calls both stop methods. Each stop closes its kit client and attempts to terminate only its stored container. Errors go to a new stderr `slog` logger. The methods do not clear stored fields or provide an idempotency result.

One fresh kit per test gives separate containers and avoids shared database rows or Redis keys. In a suite, start one kit during suite setup, register cleanup immediately, and run tests serially unless you provide separate data boundaries. `TruncateTables(t)` truncates every public base table except `schema_migrations`, resets identities, and uses `CASCADE`. It uses Testify assertions internally and has no error return. It is unsafe against a shared production database or concurrent tests. Its unquoted table-name assembly also limits unusual identifiers. Use it only with exclusively owned test schemas. It does not clear Redis. For shared Redis fixtures, assign per-test key prefixes and delete only those keys, or create a fresh kit per test.

## Global cleanup is a different operation

`CleanupAll()` executes Docker CLI commands selecting every container with the `org.testcontainers` label. Selection is not limited to this kit, this process, this application, or stopped containers. It can stop and remove containers that another suite is using. Command failures are not returned, and an empty or failed listing is logged as no containers found.

`itestkit.TestMain(m)` runs `m.Run` in a goroutine, handles SIGINT and SIGTERM, calls `CleanupAll`, and exits. Its recovery only covers panics in that runner goroutine. It cannot guarantee cleanup after every panic, `os.Exit`, forced termination, or process kill. Prefer the per-kit cleanup above, especially on a shared Docker daemon. Neither global API belongs in the recommended examples.

## Select another test goal

- For standalone Redis and Bricks wrapper construction, read [Redis application setup](redis.md).
- For one service using both stores, read [the combined recipe](../../recipes/integration-testing.md).
- For a schema-backed registration service, read [the optional migration setup](migrations.md).

If Docker startup fails, check the daemon endpoint and testcontainers environment for your runtime. The Bricks repository has an integration wrapper for its own suite, but copying this skill does not require that script. Missing Docker, images, or compatible Go tooling is a setup failure, not a skipped successful integration test.
