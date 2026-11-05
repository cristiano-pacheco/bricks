# Local-source verification

Ticket 02 adds `references/local-source.md`. The router is deferred to ticket 16. Its pointer must trigger for source investigation, undocumented behavior, API verification, and discrepancies with installed code. Ordinary usage must not load this reference. The existing `local-source-investigation` routing case and conditional follow-up in `routing.json` retain that contract.

## Passed checks

- `make check-bricks-skill` passed with Go 1.27.3 on darwin/arm64, Python 3, and `rg`. It ran 15 acceptance-utility tests and 14 local-source tests, then structural and link validation.
- `git diff --check` passed.
- Source-discovery scenarios covered a configured module cache with a decoy version, a local replacement, workspace replacement and `use` overrides, an alternate module file, automatic module vendoring, explicit vendor mode, explicit readonly override, workspace vendoring, inconsistent vendor metadata, cached metadata without source, missing metadata, and incompatible local tooling.
- The consumer fixtures use the current Bricks module path and small synthetic package files. They prove dependency selection and inspection behavior, not library runtime behavior. Cache setup uses a temporary file proxy before discovery. Discovery disables module, checksum, VCS, and toolchain retrieval. A local HTTP and HTTPS proxy trap received no requests. File-content snapshots found no changes to consumer declarations, workspace, vendor, or cached source files.
- Successful scenarios compared Go's returned package directory with the expected directory and ran the reference's bounded `rg` queries inside that directory.

The reference's shell snippets are fragments. Tests execute their Go resolution commands and search patterns. They do not implement a consumer-facing discovery program.

## Writing and measurements

Applied `writing-for-agents` to keep this guide independently loadable, give discovery steps observable stopping conditions, and keep vendor and missing-source caveats beside their instructions. Applied `unslop` before authoring and during prose review. Consumer instructions have no external skill or repository-document dependency.

The acceptance command estimates tokens as UTF-8 bytes divided by four, rounded up. The reference measures 1,479 tokens against a soft target of 2,000. No overrun occurred. Combined task loads remain unmeasured until the router exists.

## Not run

LLM routing and consumer-only behavioral evaluation await tickets 16 and 17. Structural checks are not routing proof. Go package, Docker-backed integration, and Go static suites were not run because this ticket changes documentation and Python acceptance tooling only. Package READMEs and framework code remain unchanged. Code reviews were omitted at the user's request.
