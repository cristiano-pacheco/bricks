# Migration authoring evidence

Ticket 05 documents migration contracts against revision `f862043107e9c04944af0fc348e053d7c3fe9e9c`. This is a maintainer record, not a consumer prerequisite.

## Acceptance evidence

| Ticket criterion | Delivered content and source evidence |
| --- | --- |
| Filesystems, naming, public operations, rollback, errors, and execution ownership | The [shared contract](../references/migration/usage.md) covers `FileSystem`, its `FS` field, `New`, `NewRunner`, flat roots, `fs.Sub`, numeric naming, operation selection, and application execution ownership. Backend references cover each public runner method and all six sentinels. Inspected `pkg/migration/filesystem.go`, `runner.go`, and `sqlite.go`. |
| Backend inputs and differing semantics | The [PostgreSQL branch](../references/migration/postgresql.md) explains URL input, separate connection ownership, exact-filename overwrite, duplicate version/direction rejection, skipped malformed filenames, dirty state, transaction boundaries, and absent rollback operations. The [SQLite branch](../references/migration/sqlite.md) explains caller-owned handles, strict contribution validation, version bounds, schema-state errors, atomic pending batches, and all-matching development rollback. These are source-confirmed contracts, not executed verification. |
| Actual Fx integration | The shared guide states that Bricks supplies no migration module, group, lifecycle constructor, or automatic execution. Its fragment supplies an application-defined group and runner. The complete PostgreSQL command explicitly registers an invoke and startup hook. No convenience module is invented. |
| Complete package-level examples without database constructors | The [PostgreSQL example](../examples/migration-postgresql/README.md) includes the Fx program and embedded SQL. The [SQLite example](../examples/migration-sqlite/README.md) includes a standard-library connection, explicit driver registration, owned close, embedded SQL, and an explicit down flag. Neither uses a Bricks database constructor. Backend database-module and GORM lifecycle compositions remain tickets 06 and 07. |
| Independent backend disclosure and common ownership | Shared contribution and application-owned Fx rules live in `usage.md`. Each backend guide contains its own connection, collision, version, transaction, error, and rollback rules. Pointers name exact files and their loading conditions. Larger examples load only when requested. Ordinary usage needs no checkout or external skill. |
| Writing contract, coverage, navigation, routing, and measurements | Loaded `writing-for-agents`, its skill mechanics reference, and `unslop` before authoring. Applied their hierarchy and prose rules during the writing pass. Updated migration documentation evidence in `coverage.json`, inventory pointers, and four routing cases. Byte-based measurements appear below. Automated structural and link checks were waived. |
| Integration verification and compilation | Waived by the user, not implemented or passed. Both examples are complete source but uncompiled and unexecuted. No new tests were added. |

The pinned upstream golang-migrate source at `v4.20.1` supplied PostgreSQL parsing and driver details. Inspected `source/parse.go`, `source/iofs/iofs.go`, relevant portions of `migrate.go`, and `database/postgres/postgres.go`. The configured cache was located through `GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off go env GOMODCACHE`, without dependency retrieval. Read existing `pkg/migration/runner_test.go` as behavioral prior art only; its tests did not run.

`CODING_STANDARDS.md` was absent from the repository and checked ancestor directories. Available repository instructions and the shared writing contract applied. The new `main.go` files are executable composition examples. No framework code, module declarations, existing READMEs, or root documentation changed.

## Verification and waivers

The user requested no reviews, lint, tests, or other checks for this skill-only ticket. These gates are not run and waived for ticket resolution, not passed:

- Compilation of both complete examples and PostgreSQL Fx graph validation.
- Controlled PostgreSQL integration verification for execution, repeated up, contribution collisions, malformed input, version state, and failures.
- SQLite public-operation integration verification for execution, rollback, malformed or duplicate contributions, missing down files, version state, and atomic failures.
- Startup, shutdown, connection closure, and failure-path execution.
- `make check-bricks-skill`, structural and link checks, and final documentation acceptance.
- Repository tests, integration tests, lint, static checks, and the default suite.
- `ai-code-review`, which was not invoked.

Only source inspection, the authoring prose pass, and size-only file measurements took place. No behavioral or automated-check result is claimed. Coverage branches and complete examples remain `planned`, with empty execution evidence. Retain the migration README until the later replacement gate has compilation and runtime evidence. Router implementation and consumer-only LLM routing evaluation remain later-ticket work.

## Token measurements

The method is UTF-8 byte length divided by four, rounded up per file. A Python file-size calculation produced these estimates without running tests or an acceptance checker.

| Document | Bytes | Estimated tokens |
| --- | --- | --- |
| `references/migration/usage.md` | 4557 | 1140 |
| `references/migration/postgresql.md` | 4403 | 1101 |
| `references/migration/sqlite.md` | 6055 | 1514 |
| `examples/migration-postgresql/README.md` | 2051 | 513 |
| `examples/migration-postgresql/main.go` | 1225 | 307 |
| PostgreSQL up SQL | 85 | 22 |
| PostgreSQL down SQL | 39 | 10 |
| `examples/migration-sqlite/README.md` | 2245 | 562 |
| `examples/migration-sqlite/main.go` | 1065 | 267 |
| SQLite up SQL | 86 | 22 |
| SQLite down SQL | 39 | 10 |

Each focused reference meets the 2,000-token soft target. Shared-plus-PostgreSQL guidance costs 2,241 estimated tokens. Shared-plus-SQLite guidance costs 2,654. A deliberate backend comparison reads the shared contract once and both branches for 3,755 tokens. Complete examples add 852 tokens for PostgreSQL or 861 for SQLite only when requested. Router-inclusive totals remain unavailable until the router exists.

## Routing cases

`migration-postgresql` requires only shared and PostgreSQL migration guidance. `migration-sqlite` requires only shared and SQLite migration guidance. Neither loads Bricks database-constructor or database-lifecycle recipes merely because a migration needs a connection. Their complete examples are conditional follow-ups.

`migration-backend-ambiguity` asks which backend the application uses before loading a backend branch. `multigoal-migration-backends` combines both branches and reads the shared contract once, without full examples or unrelated source inspection. These cases describe expected selection, not observed LLM evaluation results.
