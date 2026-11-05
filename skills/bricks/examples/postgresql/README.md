# Check a PostgreSQL connection with Fx

This example opens a Bricks PostgreSQL connection, runs `SELECT 1`, and shuts down. It is a one-shot check, not a server. Fx owns pool closure.

## 1. Copy the application files

Copy [main.go](main.go) and [config/base.yaml](config/base.yaml) into a directory of a Go application that already selects the desired Bricks version. Keep this layout:

```text
main.go
config/
  base.yaml
```

The program installs `database.Module` and an invoke that requests `*gorm.DB`. That request forces connection construction. The invoke registers the query as an `OnStart` hook rather than querying in the constructor. No logger module, migration provider, interface adapter, or filesystem group is needed.

## 2. Configure the connection

The supplied `config/base.yaml` lists every scalar `database.Config` field:

```yaml
app:
  database:
    host: localhost # Required.
    port: 5432 # Required. Range 1 to 65535.
    name: bricks_skill # Required. The database must already exist.
    user: bricks_skill # Required.
    password: env://POSTGRES_PASSWORD
    sslmode: false # Local development only.
    sslcert: "" # Optional certificate paths, used only with TLS enabled.
    sslkey: ""
    sslrootcert: ""
    maxopenconnections: 10
    maxidleconnections: 2
    connecttimeout: 5 # Seconds per connection attempt.
    statementtimeout: 5000 # Milliseconds.
    locktimeout: 0 # Milliseconds. Zero leaves server policy in effect.
    idleintransaction: 0 # Milliseconds. Zero leaves server policy in effect.
    timezone: UTC
    applicationname: bricks_skill_connection
    searchpath: public
    preparestmt: false
    skipdefaulttransaction: false
    disableforeignkeyconstraint: false
    enablelogs: false # Supply custom GORM Logger implementations in Go.
    prefersimpleprotol: false # Exported spelling. Currently has no effect.
```

Change the host, port, database, and user for your instance. A missing `POSTGRES_PASSWORD` becomes an empty string, not a configuration-load error. Use deployment-appropriate TLS outside a disposable local environment. `Logger` and `NamingStrategy` are Go interfaces and cannot be configured in YAML. For omitted-field defaults and each field's behavior, read the [configuration reference](../../references/database/usage.md#configuration-reference).

## 3. Provide a disposable PostgreSQL database

Use an existing disposable instance with the configured database and user, or create this local container:

```sh
docker run --name bricks-skill-postgres --rm -d \
  -e POSTGRES_DB=bricks_skill -e POSTGRES_USER=bricks_skill \
  -e POSTGRES_PASSWORD=local-example-password -p 5432:5432 postgres:17
```

Wait until PostgreSQL accepts connections. The program does not create the database or user. These are operator commands, not recorded execution evidence.

## 4. Run the check

From the copied application directory:

```sh
export POSTGRES_PASSWORD=local-example-password
export APP_CONFIG_DIR="$PWD/config"
go run .
```

A successful run exits zero after `SELECT 1` and shutdown. `run` uses separate start and stop contexts, calls `Stop` after a start error, and joins start and stop errors.

Connections open during construction, before the start context exists. Constructor attempts do not accept that context, so the start timeout is not a connection deadline. The positive `connecttimeout` configures the driver timeout for an individual connection attempt.

## 5. Clean up the container you created

If you used the Docker command above:

```sh
docker stop bricks-skill-postgres
```

Stop only your container. Do not run broad cleanup on shared Docker infrastructure.

## Next changes

For direct construction, a custom GORM logger, or other connection options, read the [database guide](../../references/database/usage.md). For SQL migrations before application queries, use the [startup example](../postgresql-startup/README.md). This connection check does not create application tables or run migrations.
