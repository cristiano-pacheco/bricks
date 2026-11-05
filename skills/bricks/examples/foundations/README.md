# Start an application with configuration and logging

This example loads service settings, starts an Fx application, logs a structured error, and flushes the logger at shutdown. It does not start an HTTP listener. The service port is configuration data.

Use a Go application that already depends on Bricks. Put the example in a separate directory so you do not replace your application's entry point.

## 1. Create `config/base.yaml`

The supplied [base file](config/base.yaml) contains:

```yaml
app:
  service:
    name: orders
    port: 8080
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
```

This base file supplies the settings used by the example. Sampling is omitted, so it is disabled.

For a copyable file containing every logger field, use [config/full.yaml](config/full.yaml). Copy its contents to `config/base.yaml` instead of the base above. Its comments identify field types, `DefaultConfig()` values, and sampling units. It includes the service section required by this program. The values are explicit example settings, not defaults merged during loading.

For a console alternative, keep your base and select the supplied [development overlay](config/development.yaml) with `APP_ENV=development`. That overlay enables debug console output and disables sampling. Terminal destinations may report unsupported-sync errors at shutdown. The logger guide explains [every configuration field](../../references/logger/usage.md).

## 2. Create `main.go`

Copy the complete program below, also supplied as [main.go](main.go):

```go
package main

import (
    "context"
    "errors"
    "fmt"
    "net/http"
    "os"
    "time"

    "github.com/cristiano-pacheco/bricks/pkg/config"
    "github.com/cristiano-pacheco/bricks/pkg/errs"
    "github.com/cristiano-pacheco/bricks/pkg/logger"
    "go.uber.org/fx"
)

type ServiceConfig struct {
    Name string `config:"name"`
    Port int    `config:"port"`
}

func registerService(lc fx.Lifecycle, cfg config.Config[ServiceConfig], log logger.Logger) error {
    service := cfg.Get()
    if service.Name == "" || service.Port < 1 || service.Port > 65535 {
        return errors.New("service requires a name and a port between 1 and 65535")
    }

    child := log.Named(service.Name).With(logger.Int("port", service.Port))
    lc.Append(fx.Hook{
        OnStart: func(context.Context) error {
            child.Info("service started")
            problem := errs.New("EXAMPLE_REJECTION", "Example request rejected", http.StatusUnprocessableEntity,
                []errs.Detail{{Field: "email", Message: "must be a valid address"}})
            problem.OriginalError = errors.New("example diagnostic cause")
            child.Warn("example application error", logger.String("code", problem.Code), logger.Error(problem))
            return nil
        },
        OnStop: func(context.Context) error {
            child.Info("service stopped")
            return nil
        },
    })
    return nil
}

func run() error {
    app := fx.New(
        logger.Module,
        config.Provide[ServiceConfig]("app.service"),
        fx.Invoke(registerService),
    )
    if err := app.Err(); err != nil {
        return err
    }

    startCtx, cancelStart := context.WithTimeout(context.Background(), 5*time.Second)
    err := app.Start(startCtx)
    cancelStart()

    stopCtx, cancelStop := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancelStop()
    return errors.Join(err, app.Stop(stopCtx))
}

func main() {
    if err := run(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}
```

`config.Provide[ServiceConfig]` loads `app.service`. `logger.Module` loads `app.logger` and provides `logger.Logger`. `registerService` requests both wrappers through Fx and validates the service settings. Requesting them makes their lazy providers run.

The logger module owns the flush hook. The service registers its hooks after the logger is constructed. Fx stops hooks in reverse order, so the service logs its final entry before the logger flushes.

## 3. Run and inspect the log

From the directory containing `main.go` and `config/`, run:

```sh
APP_CONFIG_DIR=./config APP_ENV=local go run .
```

The program starts and then stops immediately. It should write three JSON entries to `application.log`: service start, an example application-error warning, and service stop. The sample error demonstrates logging; it is not a startup failure or an HTTP response. Its original cause does not appear in `Error()` output.

The current directory must be writable. Run one instance at a time when sharing this output file. File-backed output avoids terminal destinations that may reject `Sync()`.

## 4. Try an environment-specific name

Create `config/local.yaml` alongside the base file:

```yaml
app:
  service:
    name: env://SERVICE_NAME
  logger:
    level: debug
```

Then run:

```sh
SERVICE_NAME=orders-local APP_CONFIG_DIR=./config APP_ENV=local go run .
```

The loader resolves the variable and merges this file over the base. The application now logs with the name `orders-local`. An unset `SERVICE_NAME` becomes empty, which this program's validation rejects. An absent `local.yaml` is allowed; an absent `base.yaml` is not.

For whole-document loading, other overlays, and subtree rules, follow [configuration usage](../../references/config/usage.md). For direct construction without Fx, follow [logging](../../references/logger/usage.md). For public error fields and cause handling, follow [structured errors](../../references/errs/usage.md).

To check failure handling, try a missing base file, a missing subtree, an invalid encoding, or an unwritable log destination. Handle these errors without starting the service.
