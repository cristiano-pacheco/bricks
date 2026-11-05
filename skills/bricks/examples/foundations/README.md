# Foundations Fx example

Load this example when you need a complete application combining configuration, logger interface binding, structured errors, and shutdown. For individual API choices, read only the relevant guide for [configuration](../../references/config/usage.md), [logging](../../references/logger/usage.md), or [structured errors](../../references/errs/usage.md).

[main.go](main.go) is a complete runnable program, not a fragment. It uses [config/base.yaml](config/base.yaml). It has not been compiled, graph-checked, or executed for ticket 03 because the user waived checks. The source inspection is not runtime evidence.

## Run from a consuming application

Use a Go application that already depends on Bricks and has compatible tooling and dependencies available. Copy this directory's `main.go` and `config/` into a separate example directory in that application. No Bricks checkout or other skill is required. Do not replace an existing application entry point.

From the copied directory, the intended command is:

```sh
APP_CONFIG_DIR=./config APP_ENV=local go run .
```

This command is usage guidance, not a recorded check. The example has no separate module file and uses the consuming application's dependency selection. It requires write access to the current directory for `application.log`. Do not run it concurrently with another instance writing the same file.

## Providers and lifecycle

`logger.Module` provides its own `config.Config[logger.Config]` and binds the constructed logger as `logger.Logger`. The application provides `config.Config[ServiceConfig]` at `app.service`. `fx.Invoke(registerService)` demands both providers, so configuration failures are observable during application construction. The constructor validates service settings explicitly. It does not start an HTTP listener; the port is configuration data for this example.

The service registers start and stop hooks after demanding the logger. Fx stops hooks in reverse order, so the service's final log entry precedes the logger's flush. `NewWithLifecycle` inside the logger module owns that flush. The application calls `Start` and `Stop` with separate five-second contexts and reports their errors. File-backed output avoids treating a terminal's unsupported-sync behavior as evidence of a logger failure.

The intended result is JSON entries for service start, a sample structured-error warning, and service stop in `application.log`. The sample error is demonstration data, not a startup failure. Its `OriginalError` is private metadata and is not part of `Error()` output. There is no HTTP response in this example.

## Configuration variants and failures

The base file supplies all logger settings needed by this example. `local.yaml` is optional and is intentionally absent. To exercise an overlay, create `config/local.yaml` beside the base with:

```yaml
app:
  service:
    name: env://SERVICE_NAME
  logger:
    level: debug
```

Then set `SERVICE_NAME=orders-local` before running. The loader resolves the variable before merging; this example's validation rejects an unset name. Removing `base.yaml`, choosing a missing subtree, or setting an unsupported encoding should fail construction. An unwritable log destination should also fail logger construction. These are suggested consumer checks, not claims that checks ran.

For whole-document or direct subtree loading without Fx, use the fragments in the configuration guide. For direct logging with application-owned flush, use the logging guide. Structured errors need no Fx provider.
