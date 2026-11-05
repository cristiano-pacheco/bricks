# Migrated application integration test

Run a public application operation against disposable Docker fixtures. This example is distributed with the Bricks skill and uses the application's selected Bricks dependency.

1. Copy `application_test.go` into its own test package in your application. Copy the `migrations` directory beside the test file. Keep its `integration` build tag. The [usage guide](../../references/itestkit/migrations.md) explains the complete wiring and expected results.
2. Start your Docker daemon and use compatible local Go tooling. The kit uses testcontainers and may pull its configured images.
3. From your application's module root, run `go test -tags integration -count=1 -v ./path/to/copied/package`. From a Bricks checkout, run `go test -tags integration -count=1 -v ./skills/bricks/examples/integration-testing-migrations`.
4. Expect `PASS`. The test returns setup, operation, and application cleanup errors before reporting them. Kit cleanup failures are logged because its stop methods return no error.

No YAML is required. The test configures the kit in Go and registers per-kit cleanup before startup. Do not add global `CleanupAll` or the kit's `TestMain` on a shared Docker daemon.

Execution, compilation, and Docker cleanup behavior were not checked for this documentation change, as requested by the user. The expected outcome is not a recorded pass.
