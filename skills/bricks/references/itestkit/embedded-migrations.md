# Apply embedded module migrations in a PostgreSQL fixture

Use this branch when the application collects SQL from several modules. For one SQL directory, use [test migrations](migrations.md) instead. This bridge uses the kit's PostgreSQL container and its pgx-backed GORM handle. It does not support an arbitrary production database configuration.

## 1. Start the owned fixture

Follow [PostgreSQL testkit](usage.md). Register `kit.Cleanup` before `StartPostgres`. Do not call `kit.RunMigrations` as well as the embedded runner below. Supply every production module filesystem, including foreign-key owners, with globally unique versions.

## 2. Add the fixture bridge

Keep this code in the application's integration fixture package. It borrows one connection to read pgx's actual configuration, closes that connection, then lets the Bricks runner open its own migration connection.

```go
package fixture

import (
    "context"
    "errors"
    "fmt"
    "net"
    "net/url"
    "strconv"

    "github.com/cristiano-pacheco/bricks/pkg/migration"
    "github.com/jackc/pgx/v5/stdlib"
    "gorm.io/gorm"
)

func ApplyEmbeddedMigrations(ctx context.Context, db *gorm.DB, files []migration.FileSystem) error {
    pool, err := db.DB()
    if err != nil {
        return err
    }
    connection, err := pool.Conn(ctx)
    if err != nil {
        return err
    }
    var migrationURL url.URL
    extractErr := connection.Raw(func(raw any) error {
        driverConnection, ok := raw.(*stdlib.Conn)
        if !ok {
            return fmt.Errorf("integration connection must use pgx stdlib")
        }
        cfg := driverConnection.Conn().Config()
        migrationURL = url.URL{
            Scheme: "postgres",
            User: url.UserPassword(cfg.User, cfg.Password),
            Host: net.JoinHostPort(cfg.Host, strconv.Itoa(int(cfg.Port))),
            Path: "/" + cfg.Database,
            RawQuery: url.Values{"sslmode": {"disable"}}.Encode(),
        }
        return nil
    })
    if err := errors.Join(extractErr, connection.Close()); err != nil {
        return err
    }
    return migration.NewRunner(files).Up(migrationURL.String())
}
```

The kit exposes its GORM handle but no migration URL getter. This bridge reads the real driver configuration rather than parsing a password-bearing DSN. If the driver differs, adapt to that driver's API. Do not cast it to pgx anyway.

## 3. Apply SQL before application construction

After successful PostgreSQL startup, call `ApplyEmbeddedMigrations(ctx, kit.DB(), files)` with the production file contributions. Return any failure before constructing the operation or starting the application. The file collection is the same as [migration contributions](../migration/usage.md), not a test-only schema copy.

The URL above uses `sslmode=disable` only for the kit's owned plaintext container. Production TLS, Unix sockets, search paths, and connection settings need their actual contract. Never log the URL. `Runner.Up` has no context argument, so the supplied context bounds connection acquisition, not an in-flight migration.

## 4. Prove and clean up

Invoke the application's public operation after applying the schema. Check the promised durable result and relevant failure state. This verifies the fixture's migration readiness, not the production Bricks database constructor or YAML path.

Stop the application before `kit.Cleanup` closes its pool and containers. For serial shared fixtures, reset only owned state. Do not run destructive resets concurrently.

A successful bridge run proves the selected pgx fixture path, not every driver or release. For runner collision, rollback, and dialect behavior, load only the selected [migration backend](../migration/usage.md).
