# SQLite database

The SQLite package opens a `*gorm.DB` backed by `mattn/go-sqlite3` and can
register the underlying connection with an Uber Fx lifecycle.

A complete configuration example is in
[`config/config.yaml`](config/config.yaml). It can be loaded into
`sqlite.Config` by the application's configuration layer.

```yaml
app:
  sqlite:
    dsn: ./data/workplan.db
    foreign_keys: true
    journal_mode: WAL
    busy_timeout: 5s
    tx_lock: immediate
    max_open_connections: 1
    max_idle_connections: 1
    prepare_stmt: false
    skip_default_transaction: false
    disable_foreign_key_constraint_when_migrating: false
    enable_logs: false
```

```go
package main

import (
	"log"
	"path/filepath"

	"github.com/cristiano-pacheco/bricks/pkg/database/sqlite"
)

func main() {
	db, err := sqlite.New(sqlite.Config{
		DSN: filepath.Join("data", "workplan.db"),
	})
	if err != nil {
		log.Fatal(err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal(err)
	}
	defer sqlDB.Close()
}
```

`New` applies these defaults:

- foreign-key enforcement
- WAL journal mode
- a five-second busy timeout
- immediate transaction locks
- one open and one idle connection per process

Use `GORMConfig` or the convenience GORM fields on `Config` to customize
GORM behavior. `NewWithLifecycle` uses the same connection setup and closes the
underlying `sql.DB` from its Fx `OnStop` hook.

```go
fx.New(
	sqlite.Module,
	fx.Invoke(func(db *gorm.DB) {
		// Use db. SQLite is closed when Fx stops.
	}),
)
```

`sqlite.Module` loads `app.sqlite` from the application's `config.yaml`, so
no separate config provider is needed.

The driver uses CGO, so programs and tests using this package need CGO enabled.
