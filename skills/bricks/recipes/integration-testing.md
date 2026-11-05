# Test one application with PostgreSQL and Redis

Run an Fx service that queries PostgreSQL and caches its result in Redis. The public operation must return `hello Ada`. Each test owns a fresh pair of containers. This recipe needs no schema migrations.

You need an application Go module with Bricks selected, compatible local Go tooling, and Docker accessible to testcontainers. The default images are `postgres:16-alpine` and `redis:7-alpine`. Docker may pull missing images. This is code-only setup. There are no itestkit YAML keys or automatic Fx container providers.

## Connect the application to its fixtures

1. Create `test/integration/combined/application_test.go` and copy the complete file below.

```go
//go:build integration

package integrationtesting_test

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"testing"
	"time"

	"github.com/cristiano-pacheco/bricks/pkg/itestkit"
	bricksredis "github.com/cristiano-pacheco/bricks/pkg/redis"
	upstreamredis "github.com/redis/go-redis/v9"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

type greetingService struct {
	db *gorm.DB
	cache *bricksredis.Client
}

func (s *greetingService) Execute(ctx context.Context, name string) (string, error) {
	if name == "" {
		return "", errors.New("name is required")
	}
	var greeting string
	if err := s.db.WithContext(ctx).Raw("SELECT 'hello ' || ?::text", name).Scan(&greeting).Error; err != nil {
		return "", fmt.Errorf("query greeting: %w", err)
	}
	key := s.cache.WithNamespace("greeting")
	raw := s.cache.UniversalClient()
	if err := raw.Set(ctx, key, greeting, time.Minute).Err(); err != nil {
		return "", fmt.Errorf("cache greeting: %w", err)
	}
	value, err := raw.Get(ctx, key).Result()
	if err != nil {
		return "", fmt.Errorf("read cached greeting: %w", err)
	}
	return value, nil
}

func exercise() (err error) {
	kit := itestkit.New(itestkit.DefaultConfig())
	defer kit.Cleanup()
	if err := kit.StartPostgres(); err != nil {
		return err
	}
	if err := kit.StartRedis(); err != nil {
		return err
	}
	raw, ok := kit.Redis().(*upstreamredis.Client)
	if !ok {
		return fmt.Errorf("expected standalone kit client, got %T", kit.Redis())
	}
	cfg := bricksredis.Config{
		Type: bricksredis.ClientTypeSingleNode,
		URL: (&url.URL{Scheme: "redis", Host: raw.Options().Addr}).String(),
		Namespace: "combined-test",
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cache, err := bricksredis.NewClient(ctx, cfg)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, cache.Close()) }()
	var service *greetingService
	app := fx.New(
		fx.Provide(func() *gorm.DB { return kit.DB() }),
		fx.Supply(cache),
		fx.Provide(func(db *gorm.DB, cache *bricksredis.Client) *greetingService {
			return &greetingService{db: db, cache: cache}
		}),
		fx.Populate(&service),
	)
	defer func() {
		stopCtx, cancelStop := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancelStop()
		err = errors.Join(err, app.Stop(stopCtx))
	}()
	if err := app.Err(); err != nil {
		return err
	}
	if err := app.Start(ctx); err != nil {
		return err
	}
	greeting, err := service.Execute(ctx, "Ada")
	if err != nil {
		return err
	}
	if greeting != "hello Ada" {
		return fmt.Errorf("unexpected cached greeting %q", greeting)
	}
	if _, err := service.Execute(ctx, ""); err == nil {
		return errors.New("empty name did not fail")
	}
	return nil
}

func TestCombinedApplication(t *testing.T) {
	if err := exercise(); err != nil {
		t.Error(err)
	}
}
```

2. Keep fixture setup in `exercise`. `StartPostgres` creates a mapped PostgreSQL server and its GORM connection. `StartRedis` creates a standalone server and an upstream client. The application borrows `kit.DB()` through an explicit Fx provider. Its Redis configuration reads the mapped address from the kit client and opens an independently owned Bricks wrapper. The service validates the name before using either dependency, queries PostgreSQL, writes the resulting greeting, and reads the cached value.

3. Retain the cleanup order. The kit cleanup is registered before either start call, so failures after partial setup still attempt cleanup. Deferred calls stop Fx, close the application Redis wrapper, and finally close the kit's own clients and terminate its containers. Fx-stop and wrapper-close errors join the operation error before the test reports it. Kit cleanup errors are logged, not returned. The test cannot assert that container deletion succeeded through that API.

4. Run from your application module root:

```sh
go test -tags integration -count=1 -v ./test/integration/combined
```

Expected result is `PASS`. `TestCombinedApplication` requires `hello Ada` after the database and cache operations, and rejects an empty name. It has not been executed for this documentation change.

## Configuration boundary

The minimal fixture configuration is `itestkit.DefaultConfig()`. The complete Go configuration below is a fragment that replaces the first `kit :=` line in `exercise`:

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

All fields are strings. Image fields select the corresponding Docker image. `Database`, `User`, and `Password` initialize PostgreSQL and its client. Use simple disposable credentials without spaces or connection-string syntax. `MigrationsPath` is a source URL used only by an explicit migration call; this recipe does not call it. An empty `PostgresImage` makes `New` replace the entire supplied configuration with defaults rather than merging fields. The kit has no YAML loader, callback options, custom topology settings, or configurable startup context.

The application Redis configuration uses `single_node`, the container-derived URL, and `combined-test` as its namespace. The Bricks constructor applies its own defaults and pings Redis. The `WithNamespace` call supplies the prefix explicitly. For more Redis options or complete application YAML, load [standalone Redis usage](../references/redis/standalone.md) only when needed. The fixture remains code-only.

## What the composition does and does not prove

The PostgreSQL client comes from upstream GORM inside the kit. It bypasses Bricks `database.New`, `database.NewWithLifecycle`, and `database.Module`. The Redis kit client also bypasses Bricks, but this application opens a second client through `bricksredis.NewClient`. The example exercises that constructor and application service wiring. It does not verify application YAML loading, PostgreSQL constructor options, production module composition, tracing, metrics, cluster routing, or Sentinel failover.

For direct construction without Fx, instantiate `service := &greetingService{db: kit.DB(), cache: cache}` after opening both dependencies, call `Execute`, and preserve client ownership. There is no kit-provided Fx module to install.

## Isolation and failure handling

A new kit per test isolates database state and Redis keys in separate containers. Startup methods take no caller context and use once guards. If a startup fails, create a new kit after cleaning up the failed one. A second call on the failed instance can return nil without retrying. Cleanup methods log failures and do not reset those guards.

For a serial suite, one kit can be shared across tests, but the suite owns data reset. `TruncateTables(t)` removes all public base-table data except migration metadata, resets identities, and cascades. Do not use it against schemas another test needs. It does not touch Redis. Give each test a separate key namespace and delete only those keys, or keep the per-test containers shown here.

Use `kit.Cleanup`, `StopPostgres`, or `StopRedis` for resources the kit owns. Avoid `CleanupAll` and `itestkit.TestMain` on shared Docker daemons. Both select all containers labeled `org.testcontainers`, including resources another process may still use. They do not limit selection to the current kit.

## Choose a narrower or schema-backed setup

- For PostgreSQL alone, use [the PostgreSQL guide](../references/itestkit/usage.md).
- For Redis alone, use [the Redis guide](../references/itestkit/redis.md).
- For SQL files and migration readiness before an application operation, use [the migration setup](../references/itestkit/migrations.md). Keep the SQL beside the copied test and use its absolute file URL setup.

When Docker, an image, or local Go tooling is unavailable, report the setup failure. A container that did not start is not an integration pass. None of these guides requires a framework checkout or the repository's integration wrapper in a consumer project.
