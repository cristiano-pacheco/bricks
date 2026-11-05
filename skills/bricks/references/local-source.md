# Inspect the application's local Bricks source

Load this reference for source investigation, undocumented behavior, API verification, or a discrepancy between an example and the application's installed Bricks. Ordinary usage does not require source inspection.

## Establish the effective build

Work in the consuming application's package directory, with the environment and build flags used by its actual build. Ask for missing build settings if they could change dependency selection. The module path is `github.com/cristiano-pacheco/bricks`.

Run Go commands with these process-local guards before even querying the toolchain:

```sh
export GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOVCS='*:off'
go version
go env -json GOMOD GOWORK GOFLAGS GOMODCACHE GOVERSION
```

Use a subshell if setting these variables in an interactive shell. Keep the application's other environment settings. `GOTOOLCHAIN=local` prevents automatic toolchain retrieval. `GOPROXY=off` and `GOVCS=*:off` prevent module retrieval through proxies and version-control fallback. `GOSUMDB=off` prevents checksum-service requests. A required newer toolchain is a stopping condition, not permission to fetch one.

Read only the reported module file, effective workspace file, and relevant build configuration. `GOFLAGS` may select `-modfile`, `-mod`, or `-overlay`. Preserve those settings and any equivalent command-line flags. An alternate module file also has its own sum file. Workspace `use` and `replace` entries can override module-level selection. `GOMODCACHE` is the configured cache location, not necessarily `$HOME/go/pkg/mod`.

Determine effective vendor mode before resolving packages:

- An explicit `-mod=vendor`, `-mod=readonly`, or `-mod=mod` in build flags takes precedence over automatic vendoring.
- Without an explicit mode, a module with a `go` directive of at least 1.14 and a vendor directory at its module root uses vendor mode outside a workspace.
- Workspace automatic vendoring requires Go 1.22 or newer, a workspace `go` directive of at least 1.22, and a vendor directory beside the effective `go.work`. A module's vendor directory is not the workspace vendor directory.

Use `-mod=vendor` when that is the effective mode. Otherwise use `-mod=readonly` for inspection, even when the build permits `-mod=mod`. This retains module selection while refusing dependency-file updates. If readonly resolution needs changes, report that limitation and stop. Do not repair it with `tidy`, `get`, or `download`.

## Resolve the selected directory

The following shell snippets are fragments. Keep the guards above active and include the application's effective alternate-module or overlay flags in each command.

Outside vendor mode, query only the selected module:

```sh
go list -mod=readonly -m -json github.com/cristiano-pacheco/bricks
```

Inspect `Path`, `Version`, `Main`, `Dir`, `GoMod`, `Replace`, and any error. `Dir` is Go's resolved local source directory, including local replacements and workspace modules. A replacement may have a different module path. Prefer this directory to a manually constructed cache path. `GoMod` or a version record alone does not prove source is unpacked. Require an existing `Dir` containing the relevant package files.

Resolve the specific package needed by the question, without loading its dependencies:

```sh
# Non-vendor example. Substitute the relevant Bricks package.
go list -mod=readonly -find -json github.com/cristiano-pacheco/bricks/pkg/http/request
# Vendor example. Use this instead when effective vendor mode applies.
go list -mod=vendor -find -json github.com/cristiano-pacheco/bricks/pkg/http/request
```

Check the returned package `Dir`, `Root`, `Module`, and errors. In vendor mode the package `Dir` is authoritative, even when module metadata names a replacement elsewhere. Read the effective vendor directory's `modules.txt` for the selected version and replacement record. Module queries do not reliably locate vendored source. Do not switch to readonly mode to bypass a vendor inconsistency.

Use the returned package directory for package-local investigation. If the build uses `-overlay`, read its JSON mapping and inspect mapped replacement contents for affected files. The physical directory alone does not describe an overlaid build. For cross-package work, resolve each required package the same way. If `Dir` is absent, unreadable, or resolution fails, report the selected version or replacement, effective mode, configured cache, and the exact error. A different cached version is not a substitute. Do not search the machine or fetch missing source. Ask the user to make the intended source or compatible tooling available.

## Inspect only the relevant contract

Set `package_dir` to the exact returned directory. Start with a bounded query for the requested symbol or behavior:

```sh
rg -n -g '*.go' 'func ReadJSON|func ReadJSONWithMaxSize' "$package_dir"
rg -n -g '*_test.go' 'ReadJSON|Content-Type' "$package_dir"
```

Read the matching declarations, implementation, and focused tests. If caller behavior matters, search the selected module's relevant package subtree for that symbol, not the whole cache. Expand to another package only when the call or interface relationship requires it. Keep searches inside the selected source directories. Filesystem-wide searches are prohibited.

Stop once the public signature, defaults, relevant errors, and requested runtime behavior are supported by implementation and applicable callers or tests. Report file and line evidence and distinguish tested behavior from inference. Missing evidence is a limitation, not a reason to load the whole framework.

Keep this investigation read-only. Do not execute dependency code, run generators or tests, edit cached source, or change module, sum, workspace, or vendor files as part of discovery. Running application tests is a separate user-authorized task.
