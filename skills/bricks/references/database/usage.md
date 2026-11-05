# Connect an application to PostgreSQL

Use `github.com/cristiano-pacheco/bricks/pkg/database` to provide a PostgreSQL `*gorm.DB`. The database and user must already exist. This module opens a connection pool. It does not create the database or run schema migrations.

## 1. Create the configuration

In an application module that already depends on Bricks, create `config/base.yaml`:

```yaml
app:
  database:
    host: localhost
    port: 5432
    name: bricks_skill
    user: bricks_skill
    password: env://POSTGRES_PASSWORD
```

This local setup uses plaintext connections. Host, name, user, and a port between 1 and 65535 are required. Authentication policy may require a password. An unset environment reference becomes empty rather than a loader error. Keep secrets out of logs.

The loader requires `base.yaml`. It reads `config/` unless `APP_CONFIG_DIR` selects another directory. Load [configuration](../config/usage.md) only when changing overlays, environment resolution, or subtree loading. For TLS, pooling fields, defaults, or direct construction, read [database configuration](configuration.md).

## 2. Register a database consumer

Create `main.go`:

```go
package main

import (
    "context"
    "fmt"

    "github.com/cristiano-pacheco/bricks/pkg/database"
    "go.uber.org/fx"
    "gorm.io/gorm"
)

func registerQuery(lifecycle fx.Lifecycle, db *gorm.DB) {
    lifecycle.Append(fx.Hook{
        OnStart: func(ctx context.Context) error {
            var value int
            if err := db.WithContext(ctx).Raw("SELECT 1").Scan(&value).Error; err != nil {
                return fmt.Errorf("query PostgreSQL: %w", err)
            }
            return nil
        },
    })
}

func main() {
    fx.New(database.Module, fx.Invoke(registerQuery)).Run()
}
```

`database.Module` reads `app.database` and provides one unnamed `*gorm.DB`. The invoke requests it, so Fx constructs it. Unused providers can remain lazy. Do not install another unnamed database or configuration provider alongside this module.

The constructor connects before `OnStart`. Run application queries in startup hooks or later, not in constructors that assume migrations have finished. For migrations before queries or request handling, use [PostgreSQL startup migrations](../../recipes/postgresql-migrations.md).

## 3. Run the application

From the application directory:

```sh
export POSTGRES_PASSWORD=local-example-password
export APP_CONFIG_DIR="$PWD/config"
go run .
```

Successful startup means the connection opened and `SELECT 1` succeeded. The process stays running. This query does not prove the application's schema or business behavior.

## 4. Stop the application

Press Ctrl-C. Fx closes the underlying pool during normal shutdown. Keep one pool per owned resource rather than closing it after each request. The constructor opens eagerly, so graph failure before lifecycle startup can require separate cleanup. Do not assume a stop hook covers every failed build.

Create the Go file from step 2 and copy the [minimal YAML](../../examples/postgresql/config/minimal.yaml) if preferred. These files need no Bricks checkout. This walkthrough describes expected behavior; it does not claim a consumer run.

## Select another task

- For TLS, pool fields, retries, DSNs, or direct construction, read [database configuration](configuration.md).
- For pool monitoring, read [pool metrics](../../recipes/database-pool-metrics.md).
- For GORM spans, read [GORM tracing](../../recipes/gorm-tracing.md).
- For simultaneous connections, read [multiple databases](../../recipes/multiple-databases.md). Stock PostgreSQL and SQLite modules both provide an unnamed `*gorm.DB` and collide.

Do not load these guides for a connection-only task.
