# Load YAML configuration

Use `github.com/cristiano-pacheco/bricks/pkg/config` to read YAML into a Go struct. Start with a base file, define matching types, then load the section your component needs.

## 1. Create `config/base.yaml`

Copy this file into your application:

```yaml
app:
  service:
    name: orders
    port: 8080
    origins:
      - http://localhost:3000
```

`base.yaml` is required. The package has no fixed application schema. Your Go type determines the available fields. This example defines `name`, `port`, and `origins`.

## 2. Define the type and load its section

Save this complete program as `main.go` in an application that already depends on Bricks:

```go
package main

import (
    "fmt"
    "os"

    "github.com/cristiano-pacheco/bricks/pkg/config"
)

type Service struct {
    Name    string   `config:"name"`
    Port    int      `config:"port"`
    Origins []string `config:"origins"`
}

func run() error {
    cfg, err := config.New[Service](config.WithPath("app.service"))
    if err != nil {
        return err
    }
    service := cfg.Get()
    if service.Name == "" || service.Port < 1 || service.Port > 65535 {
        return fmt.Errorf("service requires a name and a port between 1 and 65535")
    }
    fmt.Printf("%s listens on port %d; origins: %v\n", service.Name, service.Port, service.Origins)
    return nil
}

func main() {
    if err := run(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}
```

`WithPath("app.service")` selects that YAML section. `New` returns `config.Config[Service]`; call `Get()` to obtain `Service`. The loader does not call a `Validate` method, apply application defaults, or watch for file changes. Validate the result before using it.

## 3. Run with the base file

From the directory containing `main.go` and `config/`, run:

```sh
APP_CONFIG_DIR=./config APP_ENV=local go run .
```

The program should print `orders`, port `8080`, and the configured origin. No `local.yaml` file is needed.

These are the loader's environment settings:

| Setting | Default | Meaning |
| --- | --- | --- |
| `APP_CONFIG_DIR` | `config` | Directory containing `base.yaml`. Relative paths start at the process working directory. Absolute paths also work. |
| `APP_ENV` | `local` | Optional overlay filename without `.yaml`. For example, `production` selects `production.yaml`. |

The loader trims and cleans the directory path. A blank directory setting uses `config`. It trims and lowercases a nonempty environment setting. Set a real environment name; a whitespace-only `APP_ENV` becomes empty, not `local`.

## 4. Add an environment overlay when needed

Keep `base.yaml`. Create `config/production.yaml` beside it:

```yaml
app:
  service:
    name: env://SERVICE_NAME
    port: env://SERVICE_PORT
    origins:
      - env://ALLOWED_ORIGIN
```

Run with the values your deployment supplies:

```sh
SERVICE_NAME=orders-production SERVICE_PORT=8081 \
ALLOWED_ORIGIN=https://orders.example.com \
APP_CONFIG_DIR=./config APP_ENV=production go run .
```

The loader reads the base first, then merges the overlay. The production values replace the matching base values. An absent overlay is allowed. An unreadable or invalid overlay fails loading.

A value beginning with `env://` becomes the value of that environment variable. Resolution visits maps and lists in each file before merging. An unset variable becomes an empty string. It may then fail typed decoding or your validation. This is whole-value substitution, not `${NAME}` expansion or interpolation inside a connection string. Environment variables affect YAML keys only where you reference them. Keep secrets out of printed configuration.

## Load the whole document instead

Use this type when one consumer owns the complete document:

```go
type Document struct {
    App struct {
        Service Service `config:"service"`
    } `config:"app"`
}
```

With `Service` and the imports from step 2, replace the beginning of `run` with this fragment and keep its validation and output:

```go
cfg, err := config.New[Document]()
if err != nil {
    return err
}
service := cfg.Get().App.Service
```

An empty `WithPath` also selects the whole document. A missing nonempty path returns an error. Path options are trimmed; later options replace earlier ones. Nil options are ignored.

## Provide configuration with Fx

For an Fx application, use this complete alternative `main.go` with the same base YAML:

```go
package main

import (
    "fmt"

    "github.com/cristiano-pacheco/bricks/pkg/config"
    "go.uber.org/fx"
)

type Service struct {
    Name    string   `config:"name"`
    Port    int      `config:"port"`
    Origins []string `config:"origins"`
}

func main() {
    fx.New(
        config.Provide[Service]("app.service"),
        fx.Invoke(func(cfg config.Config[Service]) error {
            service := cfg.Get()
            if service.Name == "" || service.Port < 1 || service.Port > 65535 {
                return fmt.Errorf("service requires a name and a port between 1 and 65535")
            }
            fmt.Printf("loaded %s on port %d\n", service.Name, service.Port)
            return nil
        }),
    ).Run()
}
```

The invoke demands the provider, so Fx actually loads the configuration. `Provide` provides the wrapper, not `Service` itself. A custom provider returning `config.New[Document]()` is another option.

Two unnamed providers of the same `config.Config[T]` collide, even when their paths differ. Use distinct types or named Fx bindings for those cases. To combine configuration with logger lifecycle and structured errors, follow [the foundations example](../../examples/foundations/README.md).

## Defaults and failures

Missing fields retain their Go zero values. Loading a struct does not call another package's default constructor. `Get()` returns the stored value by value, not a deep copy of maps or slices.

Use `errors.Is` with exported sentinels when handling these failures:

| Sentinel | Meaning |
| --- | --- |
| `ErrConfigDirNotFound` | The resolved directory is missing or is not a directory. |
| `ErrConfigFileNotFound` | A requested YAML file is absent. Missing base files fail. The loader allows an absent overlay. |
| `ErrUnmarshalFailed` | Decoding the whole document or selected section into `T` failed. |

`ErrMissingConfigDir` exists, but `New` does not return it. A blank setting uses the default directory. Missing sections, working-directory failures, permissions, read errors, and YAML parse failures are other returned errors. Handle the returned error rather than matching its message text.
