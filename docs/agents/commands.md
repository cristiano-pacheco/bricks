# Command map

The [Makefile](../../Makefile) defines the commands and loads `.env` when present.

| Task | Command | Notes |
| --- | --- | --- |
| Test packages | `make test` | Enables CGO for SQLite. |
| Test coverage | `make unit-test-coverage` | Writes `coverage.txt`. |
| Race and HTML coverage | `make cover` | Writes `reports/cover.html`. |
| Integration tests | `make test-integration` | Requires Docker. |
| Integration race or coverage | `make test-integration-race` / `make test-integration-cover` | Coverage writes `reports/integration-cover.html`. |
| Static checks | `make static` | Runs `lint`, `vuln-check`, and `nilaway`. |
| Install tools | `make install-libs` | Static checks also install their tools. |
| Generate mocks | `make update-mocks` | Requires `mockery` on PATH. |
| Default suite | `make` | Installs tools, lints, tests, and generates HTML coverage. |

For Docker socket overrides, run `./scripts/integration-test.sh help`. CI checks are in `.github/workflows/ci.yml`.
