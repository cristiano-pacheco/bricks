# Validation and pagination authoring evidence

Ticket 04 documents the validator and paginator packages against revision `702e646e40fe756db90bb25c508839a681e90df5`. This file records maintainer evidence, not consumer prerequisites.

## Acceptance evidence

| Ticket criterion | Delivered content and source evidence |
| --- | --- |
| Struct and scalar operations, aliases, rules, translations, errors, Fx and direct construction | The [validation guide](../references/validator/usage.md) covers every Bricks export. The [complete Fx example](../examples/validation/README.md) registers an alias, string rule, and translation in its provider before consumers receive the interface. Direct construction and error handling are inline fragments. Inspected `pkg/validator/validator.go`, `fx.go`, and existing examples and tests. |
| Engine extension points without an upstream tutorial | The validation guide explains initialization timing, aliases, field rules, cross-field rules, custom type extraction, field names, and the separate English translator. Inspected the pinned upstream `validator_instance.go` registration methods in the configured local cache. The complete example includes both translation callbacks and error handling. |
| Strict parsing, normalization, capping, offsets, limits, metadata and optional strings | The [pagination guide](../references/paginator/usage.md) covers all exports and their edge behavior. Inspected `pkg/paginator/paginator.go`, `errs.go`, and existing tests. The [complete pagination example](../examples/pagination/README.md) prints invalid and boundary cases without claiming executed proof. |
| Parsing versus validation and page-size policy | Both guides distinguish decoding from application validation. The pagination guide separates strict rejection, forgiving integer fallback, capping, and application rejection of oversize pages. It records unchecked defaults, offset overflow risk, and floating-point total-page calculation. |
| Separate ownership and conditional examples | Each package has its own authoritative guide. Branch-specific fragments stay beside their explanations. Each guide points to its complete example only when that setup is requested. No framework checkout, external writing skills, or repository files are consumer prerequisites. HTTP response composition remains later-ticket work. |
| Writing, coverage, links, routing and token records | Loaded `writing-for-agents`, its skill mechanics reference, and `unslop` before authoring. Applied their hierarchy, conditional pointers, co-location, and prose rules after writing. `coverage.json` records authored destinations, source evidence, and unverified complete examples. `routing.json` adds four focused cases. The inventory links to the delivered guides and this evidence. Size-only measurements appear below. Automated link and structural checks were waived. |
| Required behavior verification and example compilation | Waived by the user, not implemented or passed. The source is complete but uncompiled and unexecuted. No new tests were added. |

`CODING_STANDARDS.md` was absent from the root and checked ancestor directories. Available repository instructions and the shared authoring contract applied instead. New `main.go` files are executable composition examples, an explicit naming exception, not framework components. No framework implementation, dependency declaration, existing README, or root documentation changed.

## Verification and waivers

The user requested no tests, lint, other checks, or code review for this skill-only ticket. All following gates are not run and waived for ticket resolution, not passed:

- Compilation of both complete examples.
- Pagination unit or executable verification, including invalid and boundary inputs.
- Validator public-operation integration verification for custom rules, translations, aliases, invalid arguments, and Fx assembly.
- Fx graph validation, startup and shutdown execution.
- `make check-bricks-skill`, structural and link checks, and final acceptance.
- Repository tests, integration tests, lint, static checks, and the default suite.
- `ai-code-review`, which was not invoked.

Only source inspection and the writing-guideline prose pass were performed. A metadata-only `go env GOMODCACHE` query used `GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off` to locate the configured cache for pinned upstream declaration inspection. No dependency fetch or executable example ran. Existing tests were read as prior art, not run as evidence.

Coverage branch states remain `planned`, with empty verification arrays. Package documentation evidence records the delivered files and waivers separately. Existing READMEs remain. The later README-removal gate still requires the agreed compilation and runtime evidence. LLM routing evaluation remains not run; the router and consumer-only evaluation belong to later tickets.

## Token measurements

The method is UTF-8 byte length divided by four, rounded up per file. A Python file-size calculation performed these measurements without running an acceptance checker or tests.

| Document | Bytes | Estimated tokens |
| --- | --- | --- |
| `references/validator/usage.md` | 5403 | 1351 |
| `references/paginator/usage.md` | 5580 | 1395 |
| `examples/validation/README.md` | 2137 | 535 |
| `examples/validation/main.go` | 2905 | 727 |
| `examples/pagination/README.md` | 1783 | 446 |
| `examples/pagination/main.go` | 2077 | 520 |

Both focused references meet the 2,000-token soft target. A combined validation-and-pagination task reads each guide once for 2,746 estimated tokens before the future router. Complete programs load only on explicit example requests. Their additional README-plus-source loads are 1,262 tokens for validation and 966 for pagination. Router-inclusive measurements remain unavailable until the router exists.

## Routing cases

Added `validation-operations`, `validation-extension`, `pagination-policy`, and `multigoal-validation-pagination`. They require only the relevant guides, exclude unrelated HTTP, storage, i18n, and local-source reads, and deduplicate multi-goal loads. Full examples are conditional follow-ups. The existing JSON-parsing case still loads validation only when the user also asks for application validation. These cases specify expected behavior; they are not observed LLM evaluation results.
