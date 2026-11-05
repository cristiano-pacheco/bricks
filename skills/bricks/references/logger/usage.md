# Add structured logging

Use `github.com/cristiano-pacheco/bricks/pkg/logger` for Zap-backed logging. The steps below load a logger from YAML, write an entry, and let Fx flush the logger at shutdown.

## 1. Create `config/base.yaml`

Start with this configuration. It writes JSON to a local file so the shutdown flush does not depend on terminal behavior:

```yaml
app:
  logger:
    level: info
    encoding: json
    outputpaths:
      - application.log
    erroroutputpaths:
      - stderr
```

The field names are lowercase, not snake case. `logger.Module` reads `app.logger` and does not merge it with `DefaultConfig()`. This setup leaves sampling disabled. The distributed [minimal file](../../examples/foundations/config/minimal.yaml) also includes the service settings used by the foundations example. See the full configuration below when you need shared fields or sampling.

## 2. Add the module and a consumer

Save this complete program as `main.go` in an application that already depends on Bricks:

```go
package main

import (
    "context"

    "github.com/cristiano-pacheco/bricks/pkg/logger"
    "go.uber.org/fx"
)

func register(lc fx.Lifecycle, log logger.Logger) {
    lc.Append(fx.Hook{
        OnStart: func(context.Context) error {
            log.Info("service started", logger.Int("port", 8080))
            return nil
        },
        OnStop: func(context.Context) error {
            log.Info("service stopped")
            return nil
        },
    })
}

func main() {
    fx.New(logger.Module, fx.Invoke(register)).Run()
}
```

Request `logger.Logger`, as shown. The module provides that interface, not `*logger.ZapLogger`. The invoke makes Fx construct the otherwise lazy provider. The module supplies its own `config.Config[logger.Config]`; you do not need another configuration provider.

## 3. Start the application and check its output

From the directory containing `main.go` and `config/`, run:

```sh
APP_CONFIG_DIR=./config APP_ENV=local go run .
```

The process remains running until you interrupt it. It should write a JSON `service started` entry to `application.log`, then `service stopped` when Fx shuts down normally. The directory must be writable.

The module registers a shutdown hook that calls `Sync()`. Because the consumer registers its hook after constructing the logger, its final entry is written before the logger flushes. `Sync` flushes shared sinks; it is not a general file-close API. Some stdout/stderr destinations return unsupported-sync errors. Decide how your application handles those errors rather than ignoring every flush failure.

## Use a console logger for local development

Keep the base file. For this alternative, create `config/local.yaml` using the YAML below. A copy is also distributed as [config/development.yaml](../../examples/foundations/config/development.yaml); select that filename with `APP_ENV=development` instead of `local`:

```yaml
app:
  logger:
    level: debug
    encoding: console
    development: true
    outputpaths:
      - stdout
    samplingconfig: null
```

Run with `APP_ENV=local` to merge these values over the base. Console output has its own encoder and string durations. JSON output uses `timestamp` and `message` keys and durations in seconds. For other overlays and environment references, read [configuration usage](../config/usage.md).

## Construct a logger without Fx

Use this alternative complete program when your application owns the flush:

```go
package main

import (
    "fmt"
    "os"

    "github.com/cristiano-pacheco/bricks/pkg/logger"
)

func run() error {
    log, err := logger.NewWithOptions(
        logger.WithLevel("info"),
        logger.WithEncoding("json"),
        logger.WithOutputPaths("application.log"),
        logger.WithErrorOutputPaths("stderr"),
        logger.WithField("service", "orders"),
    )
    if err != nil {
        return err
    }
    log.Info("service started")
    return log.Sync()
}

func main() {
    if err := run(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}
```

This program does not read YAML. `NewWithOptions` starts with `DefaultConfig()` and applies the options in order. When later application work can return early, arrange a deferred flush and preserve its error according to your shutdown policy.

`New(cfg)` instead uses exactly the supplied `Config`. Start with `DefaultConfig()` or `DevelopmentConfig()` when editing a config in Go. `MustNew` and `MustNewWithOptions` panic on construction errors; prefer error-returning constructors for deployment inputs.

## Full YAML configuration

Copy [full.yaml](../../examples/foundations/config/full.yaml) to `config/base.yaml` for every logger field, with comments describing types and defaults:

```yaml
app:
  logger:
    level: info
    encoding: json
    development: false
    disablecaller: false
    disablestacktrace: false
    outputpaths:
      - application.log
    erroroutputpaths:
      - stderr
    initialfields:
      application: foundations
      environment: local
    samplingconfig:
      initial: 100
      thereafter: 100
```

| Field | Type and behavior |
| --- | --- |
| `level` | String. Use `debug`, `info`, `warn`, `error`, `dpanic`, `panic`, or `fatal`. Unknown or uppercase names fall back to `info`. |
| `encoding` | String. Use `json` or `console`. An omitted or invalid encoding fails construction. |
| `development` | Boolean, false when omitted. Enables Zap development behavior; it does not set the level or encoding. |
| `disablecaller` | Boolean, false when omitted. True omits automatic caller information. |
| `disablestacktrace` | Boolean, false when omitted. True disables automatic stacktrace capture. |
| `outputpaths` | List of strings. Ordinary entry destinations. Omitted paths mean no ordinary output. |
| `erroroutputpaths` | List of strings. Internal logger diagnostics, not ordinary Error-level entries. Omitted paths mean no diagnostic output. |
| `initialfields` | Map of string keys to values, empty when omitted. Adds fields to every entry. |
| `samplingconfig.initial` | Integer. Keep this many entries per level/message per second before sampling. |
| `samplingconfig.thereafter` | Integer. After the initial allowance, keep every Nth entry. |

Omit `samplingconfig` or set it to `null` to disable sampling. A supplied map uses zero for omitted integers, not the example values. The configuration struct has no `config` tags. The loader supplies zero values rather than the defaults listed below.

## Constructors and options

| Setting | `DefaultConfig()` | `DevelopmentConfig()` |
| --- | --- | --- |
| Level | `info` | `debug` |
| Encoding | `json` | `console` |
| Development | `false` | `true` |
| DisableCaller / DisableStacktrace | Both `false` | Both `false` |
| OutputPaths | `stdout` | `stdout` |
| ErrorOutputPaths | `stderr` | `stderr` |
| InitialFields | Empty map | Empty map |
| SamplingConfig | Nil | Nil |

The functional options are Go functions, not additional YAML fields:

- `WithLevel`, `WithEncoding`, and `WithDevelopment` set their matching fields.
- `WithCaller(enabled)` sets `DisableCaller` to the inverse. `WithStacktrace(enabled)` does the same for `DisableStacktrace`.
- `WithOutputPaths` and `WithErrorOutputPaths` replace their destination lists.
- `WithInitialFields` merges a map. `WithField` sets one field. Later values replace earlier values with the same key.
- `WithSampling(initial, thereafter)` replaces the sampling configuration.

Omit absent functional options. Passing a nil option to `NewWithOptions` panics. Invalid or inaccessible destinations can fail construction.

For custom Fx wiring, `NewWithLifecycle(lc, cfg)` takes `fx.Lifecycle` and `config.Config[logger.Config]`, returns the concrete logger, and registers a `Sync` shutdown hook. Bind its result with `fx.Annotate(logger.NewWithLifecycle, fx.As(new(logger.Logger)))` when consumers request the interface. Use this instead of `logger.Module`, not alongside it for the same unnamed logger.

## Messages, fields, and children

`Logger` has `Debug`, `Info`, `Warn`, `Error`, `DPanic`, `Panic`, and `Fatal` methods. Each accepts a message and variadic `Field` values. `Field` aliases `zap.Field`.

Inside a function that already has `log logger.Logger` and `err error`, use this fragment:

```go
child := log.Named("checkout").With(logger.String("request_id", "r-17"))
child.Info("accepted", logger.Int("items", 2))
if err != nil {
    child.WithError(err).Error("checkout failed")
}
```

`Named` appends a logger name. `With` and `WithError` return children without changing the parent's fields. `WithError` uses the `error` key. Children share their backend sinks, so the application owns one common flush lifecycle.

Field helpers include `String`, `Int`, `Int32`, `Int64`, `Uint64`, `Float64`, `Bool`, `Error`, `Any`, `Duration`, `Time`, and `Stringer`. `Namespace` places subsequent fields under a key. `GetZapLogger()` returns the underlying `*zap.Logger` for operations that require Zap directly. Backend objects and application-specific values belong in Go, not invented YAML keys.

`DPanic` panics in development mode after logging. `Panic` always panics after logging. `Fatal` exits the process, bypassing deferred cleanup and normal Fx shutdown. Return an error when shutdown hooks must run.

For a complete configuration, logging, and structured-error application with explicit startup and shutdown, follow [the foundations example](../../examples/foundations/README.md).
