# Check a SQLite connection

This one-shot Fx application opens SQLite, prints `foreign_keys=1`, and closes its pool. It requires an application module with the selected Bricks dependency, CGO, and a C compiler. No server is needed.

1. Copy `main.go` and `config/base.yaml` from this directory into an application directory. For a smaller configuration, copy `config/minimal.yaml` as `config/base.yaml` instead.
2. From that directory, run:

   ```sh
   APP_CONFIG_DIR="$PWD/config" APP_ENV=local CGO_ENABLED=1 go run .
   ```

3. Expect `foreign_keys=1`, Fx lifecycle messages, and a zero exit status. The relative path creates `bricks-skill.db` in the working directory. It creates no application tables. The program closes the connection before exiting.
4. To choose another scenario, replace `config/base.yaml` with `config/dsn.yaml` or `config/memory.yaml`. These are complete alternative base files, not overlays. Do not merge the DSN scenario with the path configuration. The memory database disappears when its last connection closes.

For field semantics, direct ownership, and GORM options, read the [SQLite guide](../../references/database/sqlite/usage.md). For SQL schema setup before application queries, use the [SQLite startup recipe](../../recipes/sqlite-migrations.md).
