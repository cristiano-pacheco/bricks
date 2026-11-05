# PostgreSQL connection example

This complete Fx program installs `database.Module`, forces construction through an invoke consuming `*gorm.DB`, runs `SELECT 1` on startup, and stops the application. The module owns pool closure. It needs no logger module, interface adapter, filesystem group, or migration provider.

The program and its YAML are complete source, not verified execution evidence. Compilation, Fx graph validation, PostgreSQL startup/failure/shutdown checks, lint, and repository suites were waived by the user for ticket 06. Retain the existing package README until those replacement gates pass.

## Run in a consumer application

Copy [main.go](main.go) and [config/base.yaml](config/base.yaml) together into a directory of a Go application that already depends on the desired Bricks version. The example uses that application's dependency declarations and Go toolchain. It needs no Bricks checkout at runtime. If package imports need adding, manage them through the application's normal dependency workflow rather than silently fetching another Bricks version.

Create a disposable PostgreSQL instance with an existing `bricks_skill` database and `bricks_skill` user, reachable on localhost port 5432. For a local Docker instance, these are optional operator commands, not commands executed during authoring:

```sh
docker run --name bricks-skill-postgres --rm -d \
  -e POSTGRES_DB=bricks_skill -e POSTGRES_USER=bricks_skill \
  -e POSTGRES_PASSWORD=local-example-password -p 5432:5432 postgres:17
```

Wait until PostgreSQL accepts connections. From the copied example directory:

```sh
export POSTGRES_PASSWORD=local-example-password
export APP_CONFIG_DIR="$PWD/config"
go run .
```

A successful program exits zero after starting and stopping. It does not remain running as a service. Stop only the container you created with `docker stop bricks-skill-postgres` when finished. Do not apply broad container cleanup to shared Docker infrastructure. The sample disables TLS for local use; select deployment-appropriate TLS settings outside a disposable local environment.

## Configuration and lifecycle

The supplied YAML includes all required connection fields, a password reference, explicit pool limits, and positive driver and statement timeouts. Change host and port when PostgreSQL is not reachable at the sample address. `APP_CONFIG_DIR` points to the copied directory, and `base.yaml` is mandatory. `env://POSTGRES_PASSWORD` reads an existing environment variable; an absent value becomes empty, not a loader error.

`registerQuery` consumes the database before appending its hook. That forces the eager constructor and its close-hook registration first. `run` gives start and stop separate contexts, calls stop after a start error, and joins their errors. Constructor attempts happen before these contexts and do not accept them. Start timeout does not establish a constructor deadline. No migrations or application tables are involved.

For connection options and a direct-construction fragment, read the [database guide](../../references/database/usage.md). For SQL migrations before application queries, load the [separate startup example](../postgresql-startup/README.md) only when that ordering is required.
