# Use PostgreSQL and SQLite in the same Fx application

Open two independent GORM databases, resolve each by name, run one query against each, then close both. You need an application Go module with the selected Bricks dependency, an existing local PostgreSQL database, CGO, and a C compiler for SQLite.

## 1. Create the files

Create a directory inside your existing application Go module. Create `main.go` from the inline code below and copy [config/base.yaml](../examples/storage-multiple-databases/config/base.yaml) into that directory. Keep the filenames shown here. Your application must already select a Bricks dependency; these files do not select or download another framework version.

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
  sqlite:
    path: ./storage-demo.db
```

`host`, `name`, and `user` are required nonempty strings. `port` is a required integer from 1 to 65535. `password` is an optional string, empty by default. `env://POSTGRES_PASSWORD` reads that environment variable while loading configuration. `sslmode` is a boolean, false by default. These examples disable TLS only for a local development database. Names and credentials must match an existing database. PostgreSQL field names follow the exported struct names without separators, unlike SQLite's explicit underscore tags.

`app.sqlite.path` is a string naming the SQLite file relative to the process working directory. Supply `path` or `dsn`, not both. The file parent directory must already exist. The constructor enforces foreign keys and defaults to WAL, a five-second busy timeout, immediate transactions, and one open and idle connection. Fx names belong in Go tags, not YAML.

## 3. Add the Go wiring

`config.Provide` loads different typed configurations, so those providers do not collide. Both database constructors return `*gorm.DB`, so this application names their results `postgres` and `sqlite`. The `databases` input requires the matching names. Do not also install `database.Module` and `sqlite.Module`, which both provide an unnamed `*gorm.DB`. PostgreSQL's lifecycle constructor accepts the configuration value; SQLite's accepts `config.Config[sqlite.Config]` directly. Create `main.go`:

```go
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"go.uber.org/fx"

	"github.com/cristiano-pacheco/bricks/pkg/config"
	"github.com/cristiano-pacheco/bricks/pkg/database"
	"github.com/cristiano-pacheco/bricks/pkg/database/sqlite"
	"gorm.io/gorm"
)

func newPostgres(lc fx.Lifecycle, cfg config.Config[database.Config]) (*gorm.DB, error) {
	return database.NewWithLifecycle(cfg.Get(), lc)
}

type databases struct {
	fx.In
	Postgres *gorm.DB `name:"postgres"`
	SQLite   *gorm.DB `name:"sqlite"`
}

func registerWork(lc fx.Lifecycle, db databases) {
	lc.Append(fx.Hook{OnStart: func(ctx context.Context) error {
		var postgresValue int
		if err := db.Postgres.WithContext(ctx).Raw("SELECT 1").Scan(&postgresValue).Error; err != nil {
			return fmt.Errorf("query PostgreSQL: %w", err)
		}
		var sqliteValue int
		if err := db.SQLite.WithContext(ctx).Raw("SELECT 2").Scan(&sqliteValue).Error; err != nil {
			return fmt.Errorf("query SQLite: %w", err)
		}
		if postgresValue != 1 || sqliteValue != 2 {
			return fmt.Errorf("unexpected results: postgres=%d sqlite=%d", postgresValue, sqliteValue)
		}
		fmt.Printf("postgres=%d sqlite=%d\n", postgresValue, sqliteValue)
		return nil
	}})
}

func run() error {
	app := fx.New(
		config.Provide[database.Config]("app.database"),
		config.Provide[sqlite.Config]("app.sqlite"),
		fx.Provide(fx.Annotate(newPostgres, fx.ResultTags(`name:"postgres"`))),
		fx.Provide(fx.Annotate(sqlite.NewWithLifecycle, fx.ResultTags(`name:"sqlite"`))),
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
APP_ENV=local APP_CONFIG_DIR=./config POSTGRES_PASSWORD="$POSTGRES_PASSWORD" CGO_ENABLED=1 go run .
```

The application prints `postgres=1 sqlite=2` and exits after closing both pools. Fx also writes lifecycle logs. No rows or tables are created. SQLite can leave `storage-demo.db` and its WAL files in the working directory.

After the process exits, remove only the example SQLite files if you no longer need them. Do not remove live database files.

## Bind existing named clients

Application services should inject the `databases` input shown above, or use an `fx.In` field with one of the same names. A legacy component that needs an unnamed database can receive a deliberate adapter. Add this fragment beside the other providers and install `fx.Provide(defaultDatabase)` only when that component should use PostgreSQL:

```go
func defaultDatabase(db databases) *gorm.DB { return db.Postgres }
```

The adapter returns the existing handle. It must not close it or create another database. For direct construction, call `database.New` and `sqlite.New`, retain both returned handles, and close both underlying `sql.DB` values. Close the first database if opening the second fails. Do not defer cleanup until after a fatal exit.

## Troubleshooting and optional configuration

For additional connection settings or TLS scenarios, load the [PostgreSQL guide](../references/database/usage.md). For SQLite DSNs, locking, and complete fields, load the [SQLite guide](../references/database/sqlite/usage.md). For migrations on both connections, use the separate [PostgreSQL](postgresql-migrations.md) and [SQLite](sqlite-migrations.md) recipes and pass the selected connection explicitly.

Run from the application directory where you created `main.go` and `config/`, or set `APP_CONFIG_DIR` to an absolute path. The loader merges `base.yaml` with the selected environment file. Set `APP_ENV=local` as shown to avoid accidentally selecting a deployment overlay. These examples have not been compiled or executed as part of this documentation change; the results above are expected observations, not recorded test results.
