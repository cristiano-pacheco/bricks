# Redis sentinel example

Use [the sentinel guide](../../references/redis/sentinel.md) for the numbered setup, prerequisites, inline program, and field behavior.

1. In your application module, copy this directory's `main.go` into a disposable example directory.
2. Copy `config/minimal.yaml` to that directory's `config/base.yaml` and replace deployment addresses. Use `config/base.yaml` from this distributed directory instead when you need the full-field example. `config/tls.yaml` is a separate TLS scenario requiring real TLS endpoints and the correct certificate server name.
3. From the copied directory, run `APP_CONFIG_DIR="$PWD/config" APP_ENV=local go run .`.
4. Expect `demo:skill:example=hello` and exit zero. The program deletes its temporary key and closes the client. It may overwrite an existing key with that name, so use a disposable namespace.

Compilation, configuration-loader execution, connectivity, failure handling, failover, and shutdown execution were waived by the user for this change. The output above is expected, not observed.
