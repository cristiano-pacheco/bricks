# Use the Redis wrapper deliberately

Use this reference when you need wrapper helpers, functional callbacks, or precise construction behavior. For a first application, load only the relevant topology guide and follow its numbered setup. There is no separate YAML contract for functional options or Go callbacks.

## Add a functional callback with Fx

1. Keep the selected topology guide's configuration and complete `main.go`.
2. Add this provider as another argument to `fx.New`, beside `redis.ClientModule`. It supplies one untagged `[]redis.Option`. `redis.Params.Options` is optional but is not an Fx group.

```go
fx.Provide(func() []redis.Option {
    return []redis.Option{
        redis.WithRetryCallback(func(attempt int, err error) {
            fmt.Fprintf(os.Stderr, "initial Redis ping attempt %d failed: %v\n", attempt, err)
        }),
        redis.WithOnConnect(func(ctx context.Context, client *redis.Client) error {
            return client.Ping(ctx)
        }),
    }
}),
```

This is an optional fragment inside the existing `fx.New` call. Its imports are already present in the topology program. The retry message can include endpoint information, so apply your application's logging and redaction policy before using it in production.

3. Run the selected guide's command. A reachable server should still produce the same namespaced value. Retry output appears only after a failed initial ping. The callback is not installed on ordinary command retries. No callback execution results are claimed here.

For direct construction, pass the same options after the config argument to `redis.NewClient(ctx, cfg, options...)`. Custom lifecycle construction is available through `NewWithFx(lifecycle, redis.Params{Config: cfg, Options: options})` or `NewUniversalClientWithFx` instead of a module. Supply configuration yourself in this form. The modules already install their corresponding constructors.

## Effective and ineffective options

| Go option | Actual behavior |
| --- | --- |
| `WithRetryCallback` | Called on each failed initial ping, including the final failure. Its signature is `func(attempt int, err error)` and it cannot return an error. |
| `WithOnConnect` | Called once after successful initial ping, during construction rather than `OnStart`. A returned error closes the underlying client and fails construction. |
| `WithConnectTimeout` | Stores a positive duration in internal options, but no connection operation reads it. Does not change actual timeouts. |
| `WithMaxRetries` | Stores a positive integer in internal options, but initial ping uses `Config.MaxRetries`, not this option. |
| `WithRetryDelay` | Stores a positive duration, but initial retry backoff uses Config minimum and maximum backoffs. |
| `WithOnDisconnect` | Stores a callback but `Close` never calls it. Do not depend on it for cleanup. |

Configure real initial retry behavior through YAML or `Config`: `maxretries`, `dialtimeout`, `minretrybackoff`, and `maxretrybackoff`. Constructor attempts use exponential backoff, with the first sleep at the minimum. Backoff uses `time.Sleep`, not context-aware waiting, so a canceled parent context does not interrupt sleep immediately. Each underlying ping can also use go-redis's own retries. Fx constructors use a background 30-second context, independently of a later application start deadline. Construction already contacts Redis; graph construction is not an offline connection check.

A constructor failure closes the underlying client, but discards that close error. Successful direct construction transfers ownership to the caller. Successful Fx construction appends start-ping and stop-close hooks. Avoid consuming a client and then failing an unrelated invoke before startup; an unstarted Fx application does not provide the same cleanup guarantee as a successfully started application. This guide does not change those library behaviors.

## Namespace and connection helpers

`WithNamespace("order:42")` returns `"demo:order:42"` when Namespace is `demo`. `WithoutNamespace` removes one matching `"demo:"` prefix, leaving unrelated keys unchanged. These methods only transform strings. They do not isolate credentials, change Redis ACLs, prefix pipeline commands, or enforce tenant boundaries. They also do not add Redis Cluster hash tags for multi-key commands. Use explicit `{tag}` key patterns when commands require keys in one slot.

`UniversalClient()` returns the underlying go-redis interface. `Config()` returns the constructor's defaulted configuration by value, but its slices are not deeply copied. Treat configuration as immutable. The Fx-provided `redis.Config` value is the loader result, not necessarily the defaulted copy inside the client.

`Ping(ctx)` returns `ErrClientClosed` after wrapper close. `Close()` marks the wrapper closed before closing go-redis, and a second call returns `ErrClientClosed`, not nil. Do not close again in an application stop hook when a client module already owns close. `IsClosed()` reports the wrapper flag. The wrapper's closed flag has no synchronization; do not race wrapper close against wrapper state checks. The underlying go-redis client supports concurrent commands, but that does not make every wrapper method concurrently safe.

`Stats()` returns a snapshot of pool hits, misses, timeouts, total connections, idle connections, and stale connections, or nil when closed or when the concrete underlying client is unsupported. It supports the `*go-redis.Client` used by standalone and Sentinel-backed clients, and `*go-redis.ClusterClient`. Cluster statistics come from go-redis's aggregate `PoolStats` call. Never dereference a nil result.

## Local metrics are not command telemetry

With `enablemetrics: true`, `GetMetrics()` returns initial constructor ping counts and latency data. `CommandsExecuted`, `CommandsFailed`, `TotalLatency`, `AverageLatency`, and `LastCommandTime` refer to those recorded ping attempts. `ConnectionErrors` counts failed initial attempts; `ConnectionRetries` counts failed attempts before the final attempt. Public `Ping`, `Set`, `Get`, pipelines, and commands issued through `UniversalClient()` do not update this collector. Disabled metrics return nil. `ResetMetrics()` clears the local collector and is a no-op when disabled.

Prometheus pool metrics require `redis.PoolMetricsModule()`, `app.redis.metrics.enabled: true`, a `redis.UniversalClient`, and an application-supplied `prometheus.Registerer`. The module registers on start, unregisters on stop, and does not close the client. It does not serve a listener or make collectors visible on an unrelated registry automatically.

Redis tracing requires `trace.RedisModule()`, `app.redis.tracing.enabled: true`, a separately owned tracing provider through `trace.Module` or `trace.GORMModule()`, and the same universal client binding. Supported concrete clients are `*go-redis.Client`, including Sentinel/failover. Cluster and Ring are rejected. The tracing add-on does not own the client connection. Load the metrics, tracing, or storage composition recipe only when the task asks for that integration.
