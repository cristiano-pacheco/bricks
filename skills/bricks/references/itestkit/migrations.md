# Test application readiness after PostgreSQL migrations

Apply real SQL before constructing an Fx registration service. The service inserts a user and returns the stored name. A duplicate name must fail. This setup uses only PostgreSQL.

You need a consumer Go module with Bricks selected, compatible local Go tooling, and a running Docker daemon accessible to testcontainers. Docker may pull `postgres:16-alpine`. The kit's migration path and credentials are Go configuration, not YAML.

## Create the migrated application test

1. Create `test/integration/migrated/migrations/000001_create_users.up.sql`:

```sql
CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE
);
```

Create `test/integration/migrated/migrations/000001_create_users.down.sql`:

```sql
DROP TABLE users;
```

The [distributed migration directory](../../examples/integration-testing-migrations/migrations/) contains both files. Keep their numeric version and `.up.sql` and `.down.sql` suffixes.

2. Create `test/integration/migrated/application_test.go` with the following complete inline test. Go tests run with their package directory as the working directory. `filepath.Abs("migrations")` therefore selects the SQL beside the test. Encoding an absolute file URL avoids the kit's module-root-relative path behavior.

```go
//go:build integration

package integrationtesting_test

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"testing"
	"time"

	"github.com/cristiano-pacheco/bricks/pkg/itestkit"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

type userService struct{ db *gorm.DB }

func (s *userService) Register(ctx context.Context, name string) (string, error) {
	if name == "" {
		return "", errors.New("name is required")
	}
	var saved string
	if err := s.db.WithContext(ctx).Raw("INSERT INTO users (name) VALUES (?) RETURNING name", name).Scan(&saved).Error; err != nil {
		return "", fmt.Errorf("register user: %w", err)
	}
	return saved, nil
}

func exercise() (err error) {
	path, err := filepath.Abs("migrations")
	if err != nil {
		return fmt.Errorf("resolve migration directory: %w", err)
	}
	cfg := itestkit.DefaultConfig()
	cfg.MigrationsPath = (&url.URL{Scheme: "file", Path: filepath.ToSlash(path)}).String()
	kit := itestkit.New(cfg)
	defer kit.Cleanup()
	if err := kit.StartPostgres(); err != nil {
		return err
	}
	if err := kit.RunMigrations(); err != nil {
		return err
	}
	var service *userService
	app := fx.New(
		fx.Provide(func() *gorm.DB { return kit.DB() }),
		fx.Provide(func(db *gorm.DB) *userService { return &userService{db: db} }),
		fx.Populate(&service),
	)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err = errors.Join(err, app.Stop(ctx))
	}()
	if err := app.Err(); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := app.Start(ctx); err != nil {
		return err
	}
	saved, err := service.Register(ctx, "Ada")
	if err != nil {
		return err
	}
	if saved != "Ada" {
		return fmt.Errorf("unexpected saved name %q", saved)
	}
	if _, err := service.Register(ctx, "Ada"); err == nil {
		return errors.New("duplicate registration did not fail")
	}
	return nil
}

func TestMigratedApplication(t *testing.T) {
	if err := exercise(); err != nil {
		t.Error(err)
	}
}
```

3. Keep `RunMigrations` between successful `StartPostgres` and application construction. The kit uses its own PostgreSQL migration URL with `sslmode=disable`. No separate database credentials, YAML, or migration Fx providers are needed. `DefaultConfig` supplies `postgres:16-alpine`, `redis:7-alpine`, and `itest` for the PostgreSQL database, user, and password. This test overrides only `MigrationsPath`. Redis does not start.

4. Run from your application's module root:

```sh
go test -tags integration -count=1 -v ./test/integration/migrated
```

Expected result is `PASS`. The service must return `Ada` after inserting it into the migrated table. A second `Ada` must return the database uniqueness error. This establishes the intended migration-readiness boundary rather than checking table metadata alone. It has not been executed as part of this documentation change.

## Migration behavior and ownership

The kit uses upstream golang-migrate with a registered PostgreSQL database driver and file source. It calls `Up` and treats `migrate.ErrNoChange` as success. It closes the migration instance after the attempt and logs close errors rather than returning them. It does not call Bricks `migration.Module`, merge embedded filesystems, or exercise Bricks migration service constructors. For a fixture with several production module filesystems, use [embedded test migrations](embedded-migrations.md) instead of `RunMigrations`. For production startup behavior, use [PostgreSQL migrations](../../recipes/postgresql-migrations.md).

`RunMigrations` takes no context. Its once guard means only the first call attempts migration. After failure, a later call can return nil without applying anything. Stop the kit and create a new one to retry. Call it only after successful PostgreSQL startup. `MigrationsPath` is a string source URL. Its default `file://migrations` resolves relative to the nearest ancestor with `go.mod`, not the test directory. The absolute URL used here remains valid when these files are copied into a consumer module. Keep test credentials simple, because the kit's GORM connection string interpolates them without escaping.

The test borrows the kit's GORM connection through Fx. Deferred cleanup stops the app before the kit closes the pool and terminates the container. The kit's stop methods have no error result, so cleanup failures appear only in logs. A fresh kit per test gives an empty database with its own migration state. For a serial shared suite, `kit.TruncateTables(t)` removes public table data except `schema_migrations` and resets identities with `CASCADE`. It is not appropriate for concurrent tests or unrelated schemas. It uses Testify assertions internally rather than returning errors.

`CleanupAll` and `itestkit.TestMain` can remove other suites' labeled testcontainers. Keep the narrowly owned `defer kit.Cleanup()` shown above. For kit configuration fields and the other exported methods, load [the kit reference](usage.md) only when needed. To combine storage without migrations, use [the combined recipe](../../recipes/integration-testing.md).
