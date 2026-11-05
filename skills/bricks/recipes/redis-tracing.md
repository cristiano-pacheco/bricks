# Trace supported Redis clients

Export a parent application span and a Redis command span through a single standalone connection. You need an application module with Bricks, a Redis server on localhost:6379, and Docker for the optional local OTLP receiver.

## 1. Create the files

Create a directory inside your existing application Go module. Copy the distributed `main.go` and `config/base.yaml` from [the redis tracing example](../examples/storage-redis-tracing/). Keep the filenames shown here. Your application must already select a Bricks dependency; these files do not select or download another framework version. Also copy `collector.yaml` when using the development receiver.

## 2. Configure the components

Start your controlled local Redis deployment on localhost:6379 before running the example. Change the URL if your deployment uses another address. Create `config/base.yaml`:

```yaml
app:
  redis:
    type: single_node
    url: redis://localhost:6379
    tracing:
      enabled: true
  open-telemetry:
    app_name: storage-demo
    trace_enabled: true
    trace_url: localhost:4318
    exporter_type: http
    insecure: true
    sample_rate: 1.0
```

`type` is a string selecting `single_node` here. An omitted type selects that same default. `url` is the required Redis URL. This local example has no authentication or TLS and uses database zero. Server authentication, addresses, and topology must match your deployment.

`app.redis.tracing.enabled` is a boolean instrumentation switch, false when omitted. It is separate from the wrapper's local telemetry and the pool collector switch.

Under `app.open-telemetry`, `app_name` is a required nonempty string and sets `service.name`. `trace_enabled` is a boolean, false when omitted. Enabled export requires the `trace_url` string, a host and port without a scheme or path. `exporter_type` accepts `http` or `grpc` and defaults to `grpc`. `insecure` is a boolean, false by default. True disables transport security for the local receiver. `sample_rate` is a number from 0 to 1, default 0.01. The explicit 1.0 samples every root operation here. Export happens in batches, and shutdown flushes pending spans within its deadline.

For a local receiver, create `collector.yaml` beside `main.go` with the following content. This is OpenTelemetry Collector configuration, not Bricks configuration. It accepts OTLP/HTTP and writes decoded spans to its logs. If you already have an OTLP/HTTP receiver on port 4318, use it instead.

```yaml
receivers:
  otlp:
    protocols:
      http:
        endpoint: 0.0.0.0:4318
exporters:
  debug:
    verbosity: detailed
service:
  pipelines:
    traces:
      receivers: [otlp]
      exporters: [debug]
```

Start this explicit development dependency before running the application:

```sh
docker run --rm --name storage-otel -p 127.0.0.1:4318:4318 \
  -v "$PWD/collector.yaml:/etc/otelcol/config.yaml:ro" \
  otel/opentelemetry-collector:0.123.0 --config=/etc/otelcol/config.yaml
```

Keep its terminal open to observe spans. The first run may download the image. The receiver has no persistent data.

## 3. Add the Go wiring

`trace.Module` owns the provider. `RedisModule()` only installs Redis instrumentation and requires the base lifecycle. `ClientModule` owns the wrapper; `underlying` exposes its existing go-redis client through the universal interface. Application work runs after instrumentation startup and uses the parent span context. Do not install both Redis client modules. Create `main.go`:

```go
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	brickstrace "github.com/cristiano-pacheco/bricks/pkg/otel/trace"
	bricksredis "github.com/cristiano-pacheco/bricks/pkg/redis"
	"go.uber.org/fx"
)

func underlying(client *bricksredis.Client) bricksredis.UniversalClient {
	return client.UniversalClient()
}

func registerWork(lc fx.Lifecycle, client bricksredis.UniversalClient) {
	lc.Append(fx.Hook{OnStart: func(ctx context.Context) error {
		ctx, span := brickstrace.Span(ctx, "check-redis")
		defer span.End()
		value, err := client.Ping(ctx).Result()
		if err != nil {
			return fmt.Errorf("ping Redis: %w", err)
		}
		fmt.Println("redis=" + value)
		return nil
	}})
}

func run() error {
	app := fx.New(
		brickstrace.Module,
		brickstrace.RedisModule(),
		bricksredis.ClientModule,
		fx.Provide(underlying),
		fx.Invoke(registerWork),
	)
	if err := app.Err(); err != nil {
		return err
	}
	startCtx, cancelStart := context.WithTimeout(context.Background(), 30*time.Second)
	startErr := app.Start(startCtx)
	cancelStart()
	stopCtx, cancelStop := context.WithTimeout(context.Background(), 15*time.Second)
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

The `run` function returns errors after Fx shutdown. `main` reports any failure and exits nonzero only after cleanup. The stop context is fresh, so a cancelled startup or operation context does not prevent cleanup.

## 4. Run and observe the result

Run from the directory containing `main.go`:

```sh
APP_ENV=local APP_CONFIG_DIR=./config go run .
```

The application prints `redis=PONG` and exits after stopping instrumentation, closing its client, and flushing tracing. The receiver logs should show a `check-redis` parent span and a child Redis command span for service `storage-demo`. Startup success does not prove export; inspect the receiver.

Stop the example receiver after export with `docker stop storage-otel`. Stop only the container you created for this walkthrough. Backend availability is not a startup guarantee; exporter errors, sampling, and queue pressure can prevent delivery.

## Eligible topologies and lifecycle

The tracing module accepts only a nonnil underlying `*go-redis.Client`. This includes the clients returned by standalone, supported Sentinel, and failover construction. A `*redis.ClusterClient` or `*redis.Ring` rejects graph construction. Framework-owned cluster constructors also reject enabled tracing before creating the client. The universal interface alone does not imply instrumentation eligibility. The configuration's topology name is not a substitute for the actual underlying type.

To use Sentinel or failover, replace the standalone YAML with that topology's configuration and keep the Fx graph. Respect its authentication and routing limitations. Do not enable replica-only mode and then expect writes to work. Pool metrics support a broader client interface than tracing and do not prove tracing eligibility.

Redis has no hook-removal API. Startup installs or reuses the module's hook and binds it to the initialized provider. Shutdown disables the hook; it does not remove it or close the client. The connection owner closes the wrapper. A concurrently active second instrumenter for the same client fails startup. Do not separately call `redisotel.InstrumentTracing` on a module-managed client. Statement text, caller data, and dial spans are suppressed by this module; some command families are filtered. These controls are not a guarantee that arbitrary third-party error text contains no sensitive information.

## Trace Redis and GORM together

In an application that already supplies both clients, use this Fx option fragment in place of the separate provider module choices:

```go
fx.Options(
    brickstrace.GORMModule(),
    brickstrace.RedisModule(),
    database.Module,
    bricksredis.ClientModule,
    fx.Provide(underlying),
)
```

Import `github.com/cristiano-pacheco/bricks/pkg/database` as `database`. Enable `app.database.tracing.enabled` and `app.redis.tracing.enabled` independently and keep one `app.open-telemetry` section. `GORMModule()` includes the base provider; adding `Module` again would duplicate it. Register application lifecycle work after these modules. This is a composition fragment, not a standalone second program.

For direct construction, initialize tracing once, instrument an eligible client with `redisotel.InstrumentTracing` after initialization, run commands with context, close the client, and shut down tracing. That third-party direct instrumentation has different filtering and retained-hook ownership from the Bricks lifecycle module. Return every setup, command, close, and shutdown error.

## Troubleshooting and optional configuration

For full client fields and topology limitations, load only [standalone](../references/redis/standalone.md), [Sentinel](../references/redis/sentinel.md), or [failover](../references/redis/failover.md). For exporter options and sampling, load the [tracing guide](../references/otel/trace/usage.md). For the full GORM half of a combined setup, load [GORM tracing](gorm-tracing.md).

Run from the copied example directory or set `APP_CONFIG_DIR` to an absolute path. The loader merges `base.yaml` with the selected environment file. Set `APP_ENV=local` as shown to avoid accidentally selecting a deployment overlay. These examples have not been compiled or executed as part of this documentation change; the results above are expected observations, not recorded test results.
