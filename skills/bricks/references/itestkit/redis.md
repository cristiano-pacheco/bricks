# Test a Redis application with a Bricks client

Start a disposable standalone Redis server, derive its mapped address, and give an Fx application its own Bricks wrapper. The service writes and reads `hello Ada` under `greeting-test:greeting`. PostgreSQL and migrations are not needed.

You need a consumer Go module with Bricks selected, compatible local Go tooling, and a running Docker daemon accessible to testcontainers. The default Redis image is `redis:7-alpine`, which Docker may pull. Configuration is code-only. The kit does not have YAML keys or an Fx module.

## Set up the test

1. Create `test/integration/redis/application_test.go`. Copy this complete file or the matching [distributed example](../../examples/integration-testing-redis/application_test.go). The kit creates its own upstream Redis client. The example type-checks that client, reads its mapped address, and opens a separate client through `bricksredis.NewClient`.

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
)

type greetingService struct{ client *bricksredis.Client }

func (s *greetingService) Execute(ctx context.Context, name string) (string, error) {
	if name == "" {
		return "", errors.New("name is required")
	}
	key := s.client.WithNamespace("greeting")
	raw := s.client.UniversalClient()
	if err := raw.Set(ctx, key, "hello "+name, time.Minute).Err(); err != nil {
		return "", fmt.Errorf("store greeting: %w", err)
	}
	value, err := raw.Get(ctx, key).Result()
	if err != nil {
		return "", fmt.Errorf("read greeting: %w", err)
	}
	return value, nil
}

func exercise() (err error) {
	kit := itestkit.New(itestkit.DefaultConfig())
	defer kit.Cleanup()
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
		Namespace: "greeting-test",
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	client, err := bricksredis.NewClient(ctx, cfg)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, client.Close()) }()
	var service *greetingService
	app := fx.New(
		fx.Supply(client),
		fx.Provide(func(client *bricksredis.Client) *greetingService { return &greetingService{client: client} }),
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
		return fmt.Errorf("unexpected greeting %q", greeting)
	}
	if _, err := service.Execute(ctx, ""); err == nil {
		return errors.New("empty name did not fail")
	}
	return nil
}

func TestRedisApplication(t *testing.T) {
	if err := exercise(); err != nil {
		t.Error(err)
	}
}
```

2. Keep the two clients' ownership separate. The kit owns its upstream client and container. `exercise` owns the Bricks wrapper it constructs. Fx borrows that wrapper, so it does not install a second close hook. Deferred cleanup stops Fx first, closes the wrapper second, and cleans up the kit last. Wrapper-close and Fx-stop errors join the returned error before the test reports it. The kit logs its own cleanup errors and provides no error return.

3. Run from your application's module root:

```sh
go test -tags integration -count=1 -v ./test/integration/redis
```

Expected result is `PASS`. The service validates the name, stores `hello Ada` with a one-minute TTL, and reads it back. An empty name must fail before any Redis operation. These results describe the intended assertions, not an executed run. In a Bricks checkout, run the distributed copy with `go test -tags integration -count=1 -v ./skills/bricks/examples/integration-testing-redis`.

## Configuration and constructor coverage

`itestkit.DefaultConfig()` supplies the six Go fields below. No YAML loader participates in this test.

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

This is an alternative configuration fragment for the `kit :=` line in `exercise`. All fields are strings. Only `RedisImage` affects the container started here. The PostgreSQL image must still be nonempty, because an empty `PostgresImage` makes `New` replace the entire configuration with defaults. `Database`, `User`, `Password`, and `MigrationsPath` do not configure Redis. The kit starts standalone Redis without authentication, TLS, namespaces, or persistence configuration. It exposes no cluster or Sentinel fixture.

The application `bricksredis.Config` uses a mapped `redis://` URL, `single_node`, and the namespace `greeting-test`. `NewClient` supplies its normal pool, retry, and timeout defaults and pings the container. Calling `WithNamespace` is explicit; the wrapper does not automatically prefix raw commands. For the complete Redis field and option contract, load [standalone Redis usage](../redis/standalone.md) only when you need to customize it. YAML in that package guide is an alternative application configuration path, not an itestkit contract.

This example exercises the Bricks wrapper constructor and a public application service. The kit's own client uses upstream `redis.NewClient`, not Bricks. Supplying `kit.Redis()` directly to a service accepting `redis.UniversalClient` is shorter, but it bypasses `bricksredis.NewClient`, its defaults, namespace helper, and connection behavior. Neither arrangement proves cluster, Sentinel, or failover behavior.

Without Fx, construct `service := &greetingService{client: client}` after `NewClient`, call `Execute`, and retain the same cleanup order.

## Isolation and startup failures

A new kit for each test creates a separate container. The example's fixed key is isolated by that container and expires after one minute. If a suite shares one kit, use a unique namespace per test and delete only owned keys. A shared kit requires serial tests or a deliberate concurrency policy. Avoid `FLUSHALL` on shared servers.

Register kit cleanup before startup. `StartRedis` can fail after storing a container or client, so a returned error still needs cleanup. Startup uses a once guard whose failure is not preserved for later calls; create a new kit after failure. Kit startup takes no caller context, so the 30-second operation context in this example does not bound Docker startup. `StopRedis` and `Cleanup` log failures and return no error. They do not prove resource deletion.

Do not add `itestkit.TestMain` or `CleanupAll` to this setup. Both use daemon-wide label selection through the Docker CLI and can remove another suite's testcontainers. Per-kit cleanup is limited to the resources this kit stored. For the complete kit API, suite isolation, and timeout behavior, read [the kit reference](usage.md) when those details matter. For both services in one application, use [the combined recipe](../../recipes/integration-testing.md).
