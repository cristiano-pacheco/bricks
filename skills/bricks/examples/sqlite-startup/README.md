# Start SQLite after SQL migrations

This one-shot application uses the Bricks SQLite module's underlying connection for migrations, queries the migrated table, and lets Fx close the pool.

1. Copy `main.go`, `config/base.yaml`, and both files under `migrations/` into an application directory, preserving their relative paths. Use a fresh disposable database. Your application module needs the selected Bricks dependency, CGO, and a C compiler.
2. Run from that directory:

   ```sh
   APP_CONFIG_DIR="$PWD/config" APP_ENV=local CGO_ENABLED=1 go run .
   ```

   Expect `items=1`, Fx lifecycle messages, and exit status zero. `sqlite-startup.db` contains `bricks_skill_items`, one sample row, and migration version 1. Repeat the command to query the same row without applying the SQL again.
3. To remove the sample table on this disposable database, run:

   ```sh
   APP_CONFIG_DIR="$PWD/config" APP_ENV=local CGO_ENABLED=1 go run . -down
   ```

   Expect `development rollback complete`. This operation removes all available applied migrations, not just the latest one. The metadata table remains with no current version. A normal run reapplies the sample migration.

The connection closes before the program reports an error at `main`. Normal shutdown never runs rollback. The [startup recipe](../../recipes/sqlite-migrations.md) explains hook ordering, connection ownership, and failure behavior. The [SQLite guide](../../references/database/sqlite/usage.md) explains alternative configurations.
