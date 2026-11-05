# Run PostgreSQL migrations from Cobra

Use a separate migration command when deployment applies schema changes before starting the server. This recipe needs an existing Cobra application and a Bricks dependency. It does not start HTTP, construct repositories, or open an application GORM pool.

## 1. Collect the application's SQL

Follow [migration files](../references/migration/usage.md) to embed each module's SQL and return `migration.FileSystem`. Keep versions unique across modules. Annotate each provider with `group:"application_migrations"` as shown in that guide. Reuse the production files rather than creating a second migration directory.

The group name is application-owned. Replace it consistently below if the application already uses another name.

## 2. Add the command

Create `cmd/migrate_command.go`. Replace `package cmd` if the existing commands use another package.

```go
package cmd

import (
    "net"
    "net/url"
    "strconv"

    "github.com/cristiano-pacheco/bricks/pkg/config"
    "github.com/cristiano-pacheco/bricks/pkg/database"
    "github.com/cristiano-pacheco/bricks/pkg/migration"
    "github.com/spf13/cobra"
    "go.uber.org/fx"
)

type migrationInputs struct {
    fx.In
    Files []migration.FileSystem `group:"application_migrations"`
}

func provideMigrationRunner(input migrationInputs) *migration.Runner {
    return migration.NewRunner(input.Files)
}

func NewMigrateCommand(contributions ...fx.Option) *cobra.Command {
    return &cobra.Command{
        Use: "migrate",
        Args: cobra.NoArgs,
        RunE: func(command *cobra.Command, _ []string) error {
            if err := command.Context().Err(); err != nil {
                return err
            }
            options := []fx.Option{
                config.Provide[database.Config]("app.database"),
                fx.Provide(provideMigrationRunner),
                fx.Invoke(func(runner *migration.Runner, loaded config.Config[database.Config]) error {
                    cfg := loaded.Get()
                    if err := cfg.Validate(); err != nil {
                        return err
                    }
                    sslmode := "disable"
                    if cfg.SSLMode {
                        sslmode = "require"
                    }
                    timezone := cfg.TimeZone
                    if timezone == "" {
                        timezone = "UTC"
                    }
                    query := url.Values{"sslmode": {sslmode}, "TimeZone": {timezone}}
                    if cfg.SSLMode {
                        if cfg.SSLCert != "" {
                            query.Set("sslcert", cfg.SSLCert)
                        }
                        if cfg.SSLKey != "" {
                            query.Set("sslkey", cfg.SSLKey)
                        }
                        if cfg.SSLRootCert != "" {
                            query.Set("sslrootcert", cfg.SSLRootCert)
                        }
                    }
                    if cfg.SearchPath != "" {
                        query.Set("search_path", cfg.SearchPath)
                    }
                    if cfg.ApplicationName != "" {
                        query.Set("application_name", cfg.ApplicationName)
                    }
                    address := url.URL{
                        Scheme: "postgres",
                        User: url.UserPassword(cfg.User, cfg.Password),
                        Host: net.JoinHostPort(cfg.Host, strconv.Itoa(int(cfg.Port))),
                        Path: "/" + cfg.Name,
                        RawQuery: query.Encode(),
                    }
                    return runner.Up(address.String())
                }),
            }
            options = append(options, contributions...)
            return fx.New(options...).Err()
        },
    }
}
```

The invoke executes during graph construction. There are no application lifecycle resources in this graph. The runner owns its transient PostgreSQL migration connection. Pass only SQL contribution options, not a server module whose invokes could start unrelated work.

The URL escapes credentials and preserves the listed TLS and schema settings. It is not a conversion of every GORM connection option. Do not print the URL. `Runner.Up` takes no context, so the cancellation check does not cancel a migration already in progress.

## 3. Register the command and configure it

At the existing Cobra root, call `root.AddCommand(NewMigrateCommand(sqlContributions))`. Here `sqlContributions` is the `fx.Option` containing the annotated SQL providers from step 1. There is no built-in Bricks migration module to install.

Create `config/base.yaml`:

```yaml
app:
  database:
    host: localhost
    port: 5432
    name: orders
    user: orders
    password: env://POSTGRES_PASSWORD
    sslmode: false
```

The database must already exist. This plaintext configuration is for local development. Read [database configuration](../references/database/configuration.md) for production TLS and field behavior. Required secrets need application validation because an unset environment reference becomes an empty string.

## 4. Run and verify

Run the application's actual executable. For a root executable at the module root:

```sh
APP_CONFIG_DIR=./config APP_ENV=local go run . migrate
```

Expect pending SQL to apply and the command to exit successfully without an HTTP listener. Run it again to check the no-change path. In a controlled database, check that an operation requiring the new schema succeeds. A valid Fx graph alone does not prove migration execution.

Return errors through Cobra. The executable reports them and chooses its exit status. Do not add `os.Exit` or fatal logging inside the command. Record migration execution separately from compilation or graph checks.
