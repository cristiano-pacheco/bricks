# Migration contributions and execution

`github.com/cristiano-pacheco/bricks/pkg/migration` combines application-owned migration files. Choose the operation by backend:

- For PostgreSQL DSNs, `Runner.Up` applies pending migrations. Read [PostgreSQL execution](postgresql.md) for connection ownership, collision handling, and failure behavior.
- For caller-owned SQLite `*sql.DB` handles, `Runner.UpSQLite` applies pending migrations. Read [SQLite execution](sqlite.md) for version checks, transactions, and explicit development rollback through `DownSQLite`.

Only load the chosen backend reference. Neither path requires a Bricks database constructor or GORM. There is no public PostgreSQL rollback method, generic `Down`, version query, force-version operation, or runner close method.

## Filesystem contract

`New(f fs.FS) FileSystem` wraps the filesystem in the exported `FS` field. `NewRunner(filesystems []FileSystem) *Runner` stores the contributions without validating or executing them. Supply valid, non-nil filesystems and keep their contents and slice unchanged while the runner is in use. Each execution reads them again.

Each contribution must expose migration files at its root. Both backends read `fs.ReadDir(FS, ".")`, skip directories, and never recurse. Use `fs.Sub` to select the embedded directory. This fragment requires an embedded `migrations` directory in the same Go package:

```go
// Fragment. Imports: embed, io/fs, and the Bricks migration package.
//go:embed migrations/*.sql
var embedded embed.FS

func migrationFiles() (migration.FileSystem, error) {
    root, err := fs.Sub(embedded, "migrations")
    if err != nil {
        return migration.FileSystem{}, err
    }
    return migration.New(root), nil
}
```

Use names such as `000001_create_accounts.up.sql` and `000001_create_accounts.down.sql`. The parser accepts a decimal version, an underscore, an identifier, a lowercase `up` or `down` direction, and an extension. It does not enforce SQL extensions or timestamp widths. Use positive versions and SQL files for predictable application conventions. Up and down at one version are separate directions. Versions sort numerically, not by contribution order or identifier. Reserve unique versions across modules, including timestamp-based names. Down SQL must undo the corresponding up SQL in reverse dependency order.

SQL is passed to the selected backend, without template interpolation or dialect conversion. File-read errors return through the execution method. Filename conflicts and malformed files differ by backend, so retain the selected backend's rules when composing contributions. Keep unrelated files outside the migration root.

## Application-owned Fx wiring

Bricks supplies no migration `Module`, configuration provider, lifecycle constructor, or automatic filesystem group. Applications register the runner and explicitly call an operation. A provider alone does not run migrations. Group names are application choices, not Bricks contracts.

This fragment collects contributions through an application-defined group:

```go
// Fragment. Imports: go.uber.org/fx and the Bricks migration package.
type migrationInputs struct {
    fx.In
    Files []migration.FileSystem `group:"application_migrations"`
}

func provideRunner(in migrationInputs) *migration.Runner {
    return migration.NewRunner(in.Files)
}

// Include these options in the application's fx.New call.
var migrationOptions = fx.Options(
    fx.Provide(
        fx.Annotate(migrationFiles,
            fx.ResultTags(`group:"application_migrations"`)),
        provideRunner,
    ),
)
```

Fx value groups have unspecified ordering. Avoid filename collisions rather than relying on group order. The application must also register an invoke or startup hook that executes the selected backend operation and returns failures. Run migrations before serving requests or starting workers that depend on the schema. `UpSQLite` never calls `DownSQLite`; rollback must remain an explicit development action. Serialize schema changes within the application's deployment workflow.

For a complete PostgreSQL-only Fx migration command with no application database provider, load the [PostgreSQL example](../../examples/migration-postgresql/README.md). For a complete SQLite command that owns its standard-library connection, load the [SQLite example](../../examples/migration-sqlite/README.md). Application database startup, GORM configuration, and named multi-database providers belong to database composition guides, not this package contract.
