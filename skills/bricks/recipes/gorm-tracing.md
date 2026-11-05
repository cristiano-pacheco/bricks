# Trace PostgreSQL operations through GORM

Export a parent application span and a GORM query span to a local OTLP receiver. You need an application module with Bricks, an existing local PostgreSQL database, and Docker for the optional receiver below.

## 1. Create the files

Create a directory inside your existing application Go module. Create `main.go` from the inline code below and copy [config/base.yaml](../examples/storage-gorm-tracing/config/base.yaml) into that directory. Keep the filenames shown here. Your application must already select a Bricks dependency; these files do not select or download another framework version. Also copy `collector.yaml` when using the development receiver.

## 2. Configure the components

Set `POSTGRES_PASSWORD` in your shell to the password of the existing `bricks_skill` PostgreSQL user. That user must have access to an existing database named `bricks_skill` on localhost:5432. Change the connection fields if your controlled development database uses different values. Create `config/base.yaml`:

```yaml
app:
  database:
    host: localhost
    port: 5432
    name: bricks_skill
    user: bricks_skill
    password: env://POSTGRES_PASSWORD
    sslmode: false
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

`host`, `name`, and `user` are required nonempty strings. `port` is a required integer from 1 to 65535. `password` is an optional string, empty by default. `env://POSTGRES_PASSWORD` reads that environment variable while loading configuration. `sslmode` is a boolean, false by default. These examples disable TLS only for a local development database. Names and credentials must match an existing database. PostgreSQL field names follow the exported struct names without separators, unlike SQLite's explicit underscore tags.

`app.database.tracing.enabled` is a boolean instrumentation switch, false when omitted. It does not enable export on its own.

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

`GORMModule()` includes base tracing initialization, so do not also install `trace.Module`. The database remains application-owned through `database.Module`. Register the application hook after the tracing module so provider initialization and callback installation precede the query. The context returned by `Span` reaches GORM through `WithContext`. Create `main.go`:

```go
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/cristiano-pacheco/bricks/pkg/database"
	brickstrace "github.com/cristiano-pacheco/bricks/pkg/otel/trace"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

func query(ctx context.Context, db *gorm.DB) error {
	ctx, span := brickstrace.Span(ctx, "check-postgres")
	defer span.End()
	var value int
	if err := db.WithContext(ctx).Raw("SELECT 1").Scan(&value).Error; err != nil {
		return fmt.Errorf("query PostgreSQL: %w", err)
	}
	if value != 1 {
		return fmt.Errorf("unexpected query result: %d", value)
	}
	fmt.Println("postgres=1")
	return nil
}

func registerWork(lc fx.Lifecycle, db *gorm.DB) {
	lc.Append(fx.Hook{OnStart: func(ctx context.Context) error {
		return query(ctx, db)
	}})
}

func run() error {
	app := fx.New(
		brickstrace.GORMModule(),
		database.Module,
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
APP_ENV=local APP_CONFIG_DIR=./config POSTGRES_PASSWORD="$POSTGRES_PASSWORD" go run .
```

The application prints `postgres=1` and exits after removing the instrumentation callbacks, closing the pool, and flushing tracing. The receiver logs should contain a `check-postgres` parent span and a child query span for service `storage-demo`. The query span name is plugin-defined. Successful startup alone does not prove delivery.

Stop the example receiver after export with `docker stop storage-otel`. Stop only the container you created for this walkthrough. Backend availability is not a startup guarantee; exporter errors, sampling, and queue pressure can prevent delivery.

## Callback ownership and direct construction

Enabled GORM instrumentation requires an initialized unnamed `*gorm.DB`. It rejects an existing tracing plugin or conflicting tracing callback names. It installs callbacks at startup and removes only its own plugin and callbacks at shutdown. Do not modify callbacks concurrently with queries. Finish application work before stopping the Fx application.

The plugin disables its metrics, suppresses query variables, and replaces SQL text with an empty string. This does not sanitize third-party error messages. Set `app.database.tracing.enabled: false` to skip the plugin while retaining base tracing initialization. Set `trace_enabled: false` to select disabled export. These switches have different responsibilities.

For direct lifecycle ownership, initialize the provider with `trace.Initialize`, install `gorm.io/plugin/opentelemetry/tracing.NewPlugin` with an explicit tracer provider, run queries with context, then close the database and call `trace.Shutdown` with a fresh bounded context. Propagate installation and shutdown errors. Do not combine direct provider initialization with `GORMModule()`. The convenience module's callback removal is not a general GORM plugin unload API.

## Troubleshooting and optional configuration

For full PostgreSQL fields, load the [database guide](../references/database/usage.md). For exporter choices, sampling, and complete tracing fields, load the [tracing guide](../references/otel/trace/usage.md). For a simultaneous Redis client, load [Redis tracing](redis-tracing.md) and use its combined-module arrangement.

Run from the application directory where you created `main.go` and `config/`, or set `APP_CONFIG_DIR` to an absolute path. The loader merges `base.yaml` with the selected environment file. Set `APP_ENV=local` as shown to avoid accidentally selecting a deployment overlay. These examples have not been compiled or executed as part of this documentation change; the results above are expected observations, not recorded test results.
