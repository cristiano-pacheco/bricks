# Foundations authoring evidence

Ticket 03 documents configuration, logging, and structured errors against source revision `518e8ba`. These records are maintainer evidence, not consumer prerequisites.

## Delivered coverage

| Ticket criterion | Content and evidence |
| --- | --- |
| Whole document, subtree, Fx, base file, overlays, environment references, defaults, failures | [Configuration guide](../references/config/usage.md). Inspected `pkg/config/config.go`, `provider.go`, and `errs.go`. |
| Logger constructors, configuration, options, children, backend, shutdown | [Logger guide](../references/logger/usage.md). Inspected `pkg/logger/logger.go`, `config.go`, `options.go`, and `fx.go`. |
| Error codes, messages, details, status, predefined error, original cause, wrapping | [Structured-error guide](../references/errs/usage.md). Inspected `pkg/errs/errs.go`, including JSON tags and method declarations. |
| Complete Fx wiring and direct alternatives | [Foundations example](../examples/foundations/README.md), with `main.go` and explicit `config/base.yaml`. The package guides classify direct alternatives as fragments. The example demands providers and owns start and stop; logger shutdown owns flush. |
| Package ownership and conditional example loads | Each package has its own guide. Each example pointer states when to load it. The logger guide explains its configuration wiring without requiring the complete configuration guide. |
| Current-source alignment | Constructor assignments establish the documented defaults and lack of default merging. `logger.Module` provides the interface only. `errs.Error` has no `Unwrap`. Static inspection is the only verification performed. |
| Writing and acceptance records | Applied `writing-for-agents` to hierarchy and conditional pointers. Loaded `unslop` before Markdown edits and applied its prose rules after authoring. Added foundation routing cases and per-package documentation evidence. |

`CODING_STANDARDS.md` was absent from the repository root and the checked ancestor directories. The available repository instructions and shared authoring contract governed this documentation work. No framework code or existing README was changed.

## Waived checks

The user explicitly requested no tests, lint, other checks, or code review for this skill-only task. The following gates are waived for this ticket, not passed:

- Compilation of the complete example.
- Configuration and structured-error unit checks.
- Fx graph resolution, configured logger integration, construction failures, startup, and shutdown execution.
- `make check-bricks-skill`, structural and link checking, and the final acceptance command.
- Repository test, integration, lint, static, and default suites.
- `ai-code-review`.

The authored example is complete source but unexecuted. `coverage.json` retains planned branch states so this waiver cannot become false verified coverage. Its per-package documentation evidence names the delivered references and examples. Existing READMEs remain in place. Future README removal still requires the verification defined by the shared acceptance contract. LLM routing evaluation remains not run; the router and its evaluation belong to later tickets.

## Token measurements

The method is UTF-8 bytes divided by four, rounded up. A small Python calculation measured file sizes only; it did not run acceptance checks or tests.

| Document or selected set | Bytes | Estimated tokens |
| --- | --- | --- |
| `references/config/usage.md` | 4548 | 1137 |
| `references/logger/usage.md` | 5671 | 1418 |
| `references/errs/usage.md` | 3398 | 850 |
| `examples/foundations/README.md` | 3561 | 891 |
| Configuration and logger, deduplicated | 10219 | 2555 |

All focused references are below the 2,000-token soft target. The combined configuration-and-logger load excludes examples and totals the per-file estimates. The router does not exist yet, so complete router-inclusive task loads remain unmeasured. Runnable source and YAML load only when the task requests the complete example and are outside routine reference estimates.

## Routing cases

`acceptance/routing.json` adds `config-loading`, `logger-lifecycle`, `structured-errors`, and `multigoal-foundations`. They distinguish ordinary usage from optional configuration details, complete examples, and local-source investigation. They specify unrelated-load exclusions and deduplication. These are planned behavioral cases, not observed LLM routing results.
