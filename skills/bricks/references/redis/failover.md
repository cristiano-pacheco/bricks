# Connect through the failover URL branch

This application writes `demo:skill:example`, reads `hello`, deletes the key, and closes the connection through Fx. You need a Go application module with its selected Bricks dependency and an unauthenticated Sentinel at `localhost:26379` monitoring `mymaster`, with reachable advertised Redis addresses. Use a disposable key namespace, since the example overwrites its key. These examples have not been compiled or executed in this documentation change.

## 1. Create `config/base.yaml`

```yaml
app:
  redis:
    type: failover
    url: redis://localhost:26379
    mastername: mymaster
    db: 0
    namespace: demo
```

Set addresses and credentials for your deployment. The module reads `app.redis` from `base.yaml` under `APP_CONFIG_DIR`, default `config`. `APP_ENV` selects an optional overlay and defaults to `local`. Field names are lowercase Go field names without added underscores.

## 2. Create `main.go`

The client is constructed and pinged while Fx builds the dependency graph, then pinged again on start. The application hook runs after the client start hook. Fx closes the client on stop, after application hooks finish.

```go
package main

import (
    "context"
    "errors"
    "fmt"
    "os"
    "time"

    "github.com/cristiano-pacheco/bricks/pkg/redis"
    "go.uber.org/fx"
)

func roundTrip(ctx context.Context, client *redis.Client) (err error) {
    raw := client.UniversalClient()
    key := client.WithNamespace("skill:example")
    if err := raw.Set(ctx, key, "hello", time.Minute).Err(); err != nil {
        return fmt.Errorf("set example: %w", err)
    }
    defer func() {
        cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()
        err = errors.Join(err, raw.Del(cleanupCtx, key).Err())
    }()
    value, err := raw.Get(ctx, key).Result()
    if err != nil {
        return fmt.Errorf("get example: %w", err)
    }
    if value != "hello" {
        return fmt.Errorf("unexpected value %q", value)
    }
    fmt.Printf("%s=%s\n", key, value)
    return nil
}

func registerOperation(lc fx.Lifecycle, client *redis.Client) {
    lc.Append(fx.Hook{OnStart: func(ctx context.Context) error {
        return roundTrip(ctx, client)
    }})
}

func run() error {
    app := fx.New(redis.ClientModule, fx.Invoke(registerOperation))
    if err := app.Err(); err != nil {
        return err
    }
    startCtx, cancelStart := context.WithTimeout(context.Background(), 30*time.Second)
    startErr := app.Start(startCtx)
    cancelStart()
    stopCtx, cancelStop := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancelStop()
    return errors.Join(startErr, app.Stop(stopCtx))
}

func main() {
    if err := run(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}
```

The namespace is explicit. Calling `Set` on the underlying client does not prefix keys automatically. The operation gives the temporary key a one-minute expiry and attempts deletion with a fresh cleanup context. Cleanup and stop errors reach the main boundary before a nonzero exit.

## 3. Run the application

```sh
APP_CONFIG_DIR="$PWD/config" APP_ENV=local go run .
```

Expect `demo:skill:example=hello`, Fx lifecycle messages, and exit status zero. This is expected behavior, not a recorded execution result. A successful connection does not prove automatic failover during a primary failure.

For copying rather than retyping, use [the distributed example](../../examples/redis-failover/README.md), its `main.go`, and `config/minimal.yaml` renamed to `config/base.yaml`. It also includes the full configuration and a separate TLS scenario.

## Direct construction

To use this branch without Fx, replace `run` above with the following function and remove the Fx import. It reuses `roundTrip`, returns errors, and owns close explicitly. It does not load YAML.

```go
func run() (err error) {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    client, err := redis.NewClient(ctx, redis.Config{Type: redis.ClientTypeFailover, URL: "redis://localhost:26379", MasterName: "mymaster", Namespace: "demo"})
    if err != nil {
        return err
    }
    defer func() { err = errors.Join(err, client.Close()) }()
    return roundTrip(ctx, client)
}
```

To keep YAML in the direct version, add the `github.com/cristiano-pacheco/bricks/pkg/config` import, load `config.New[redis.Config](config.WithPath("app.redis"))`, return any load error, and pass `loaded.Get()` to `NewClient` instead of the literal.

## Full configuration for this branch

[base.yaml](../../examples/redis-failover/config/base.yaml) contains every shared Config setting and every setting used by this topology. Fields belonging only to other topologies are intentionally omitted. `unstablemode` and `commandtimeout` remain visible because the type accepts them, but they do not change runtime behavior.

```yaml
app:
  redis:
    type: failover
    url: redis://localhost:26379
    mastername: mymaster
    db: 0
    password: ""
    maxretries: 3
    minretrybackoff: 8ms
    maxretrybackoff: 512ms
    dialtimeout: 5s
    readtimeout: 3s
    writetimeout: 3s
    contexttimeoutenabled: true
    poolfifo: false
    poolsize: 10
    pooltimeout: 4s
    minidleconns: 2
    maxidleconns: 0
    connmaxidletime: 30m
    connmaxlifetime: 1h
    enabletls: false
    tlsskipverify: false
    tlsservername: ""
    protocol: 3
    disableindentity: false
    identitysuffix: ""
    namespace: demo
    enablemetrics: false
    commandtimeout: 5s
    unstablemode: false
```

Despite its name, this is another Sentinel-backed `NewFailoverClient`, not a direct connection to a Redis primary. `url` is a required string parsed as a Redis URL, but only its address becomes the single Sentinel address. Use a Sentinel port. The URL username, password, database, TLS scheme, and query options are not copied into FailoverOptions. `mastername` is a required nonblank string with no default. `db` is an optional integer, defaults 0, and is not range-validated here. This branch ignores `sentineladdrs`, `sentinelusername`, `sentinelpassword`, `routebylatency`, `routerandomly`, and `replicaonly`. Choose the separate Sentinel guide if Sentinel authentication or multiple discovery addresses are needed. TLS comes from explicit TLS fields or a nonempty Redis `password`, not this URL scheme.

All other fields here are optional. Their types, defaults, units, and constraints are defined once in [shared configuration](configuration.md). Load that reference when tuning pools, retries, protocol, or TLS. For verified TLS, replace the file with [tls.yaml](../../examples/redis-failover/config/tls.yaml), set real TLS endpoint addresses, and replace `tlsservername` with the certificate's server name. Keep `tlsskipverify` false. Do not overlay a topology change onto an old file, since leftover URL values can take precedence.

## Client choices and failures

`redis.ClientModule` provides only `*redis.Client`. If consumers need `redis.UniversalClient` instead, install `redis.Module` and change their parameter types. It is an alias of go-redis's interface. Install one client module, not both. To retain the wrapper and expose the same underlying connection, keep `ClientModule` and add `fx.Provide(func(c *redis.Client) redis.UniversalClient { return c.UniversalClient() })`. This does not create a second client or transfer close ownership.

Constructor configuration errors wrap `ErrInvalidConfig` and retain `*ConfigError`. URL parsing and exhausted initial pings return `*ConnectionError`, whose `Unwrap` preserves the cause. Do not depend on `ErrConnectionFailed` or `ErrPingFailed` for these failures, since these constructors do not wrap them. Use `errors.As` for the typed errors and `errors.Is` for actual wrapped causes. Diagnostics can contain the configured URL, so avoid embedding secrets in it.

For functional callbacks, namespace helpers, pool statistics, local metric limitations, and close semantics, load [wrapper behavior](client.md). Pool collection and tracing require separate modules and supplied interfaces; installing a client module does not instrument every topology or expose a metrics endpoint.
