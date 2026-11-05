# Logging

Use `github.com/cristiano-pacheco/bricks/pkg/logger` for structured Zap-backed logging. Prefer constructors that return errors when configuration comes from deployment inputs.

## Constructors and configuration

`New(Config)` returns `*ZapLogger, error` and uses the supplied configuration without merging defaults. Start direct construction with `DefaultConfig()` or `DevelopmentConfig()`, then change fields. `MustNew(Config)` panics on a build error.

`NewWithOptions(...Option)` starts with `DefaultConfig()` and applies options in order. `MustNewWithOptions` uses the same path but panics on error. A nil functional option panics; omit absent options.

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

`Level` recognizes lowercase `debug`, `info`, `warn`, `error`, `dpanic`, `panic`, and `fatal`. Unknown strings, including uppercase variants, fall back to `info`. Encoding supports `json` and `console`; invalid encodings and invalid or inaccessible output paths can fail Zap construction. Internal logger errors use `ErrorOutputPaths`, not the destination for ordinary Error-level entries.

`SamplingConfig` has `Initial` and `Thereafter` integers. Zap sampling keeps the initial entries for a level/message per second and then every Nth entry. Nil disables sampling. `Development` enables Zap development behavior; it does not independently select console encoding or debug level.

The functional options map to these settings:

- `WithLevel`, `WithEncoding`, and `WithDevelopment` set their corresponding fields.
- `WithCaller(enabled)` sets `DisableCaller` to the inverse. `WithStacktrace(enabled)` does the same for `DisableStacktrace`.
- `WithOutputPaths(...string)` and `WithErrorOutputPaths(...string)` replace their path lists.
- `WithInitialFields(map[string]any)` merges fields. `WithField(key, value)` sets one field. Later values replace earlier values with the same key.
- `WithSampling(initial, thereafter)` replaces the sampling configuration.

Direct-construction fragment, inside a function returning `error`, with `logger` imported:

```go
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
log.Info("started")
return log.Sync()
```

If work can return early, arrange a deferred flush and preserve its error according to application policy. `Sync()` flushes shared sinks; it is not a general file-close API. Some stdout/stderr destinations report unsupported-sync errors. Handle those deliberately rather than silently discarding every flush failure.

## Messages, fields, and children

`Logger` is the interface implemented by `*ZapLogger`. It includes `Debug`, `Info`, `Warn`, `Error`, `DPanic`, `Panic`, and `Fatal`, each accepting a message and variadic `Field` values. `Field` aliases `zap.Field`.

`DPanic` logs at DPanic level and panics in development mode. `Panic` panics after logging. `Fatal` exits the process and bypasses deferred cleanup and normal Fx shutdown. Prefer returning an error when shutdown hooks must run.

Use the field helpers `String`, `Int`, `Int32`, `Int64`, `Uint64`, `Float64`, `Bool`, `Error`, `Any`, `Duration`, `Time`, and `Stringer`. `Namespace` scopes subsequent fields under a key. JSON encoding emits durations in seconds and uses `timestamp` and `message`; console encoding uses string durations and its console encoder.

This fragment assumes an existing `log`:

```go
child := log.Named("checkout").With(logger.String("request_id", "r-17"))
child.Info("accepted", logger.Int("items", 2))
child.WithError(err).Error("checkout failed")
```

`Named` appends a logger name. `With` and `WithError` return children with attached fields, leaving the parent context unchanged. `WithError` uses the `error` field key. Children share backend sinks; the application owns their common flush lifecycle. `GetZapLogger()` exposes the underlying `*zap.Logger` for Zap-specific operations.

## Fx and shutdown

`logger.Module` loads `config.Config[logger.Config]` at `app.logger`, constructs the logger with `NewWithLifecycle`, and exposes it as `logger.Logger`. It does not also expose `*logger.ZapLogger`. Consumers must request the provided interface.

The YAML configuration is explicit, not a merge with `DefaultConfig()`. Supply `level`, `encoding`, `outputpaths`, `erroroutputpaths`, and any other needed fields. The struct has no `config` tags, so use lowercase field names such as `disablecaller`, `initialfields`, and `samplingconfig`, not guessed snake-case names.

`NewWithLifecycle(lc, cfg)` is an alternative for custom Fx wiring. It returns the concrete logger and appends an `OnStop` hook that returns `Sync()` errors. Bind it to `logger.Logger` with `fx.Annotate(..., fx.As(new(logger.Logger)))` if consumers require the interface. Do not install both this provider and `logger.Module` for the same unnamed logger.

For complete provider wiring, YAML, startup, shutdown, and file-backed flush ownership, read [the foundations example](../../examples/foundations/README.md). Load [configuration usage](../config/usage.md) only when you need overlay, environment substitution, subtree, or loader-error details beyond that example.
