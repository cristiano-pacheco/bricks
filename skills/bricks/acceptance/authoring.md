# Bricks skill authoring contract

This directory holds maintainer acceptance records. It is not a required consumer read. The consumer entry point and all required usage instructions must travel inside `skills/bricks/`.

## Before authoring

Load `writing-for-agents` before designing the hierarchy and pointers. Load `unslop` before creating or editing Markdown, then apply it again during prose review. These are authoring requirements, not consumer dependencies. A copied Bricks skill must work without either authoring skill or a framework checkout.

Read current source and applicable repository instructions before documenting behavior. The inventory records the source revision used. Refresh affected exports, branches, examples, and evidence when public APIs change. Document current code, including unreleased changes. Keep recommendations separate from supported alternatives and limitations.

## Reference ownership

Use these destination types, relative to `skills/bricks/`:

- `SKILL.md` is the compact capability router. Each pointer names an exact target and the task condition that warrants reading it. Route directly to the needed reference or recipe rather than through mandatory area indexes.
- `references/<package>/usage.md` owns a package's shared usage contract. Preserve the source package hierarchy, including separate areas for nested support packages such as `references/i18n/ports/` and `references/http/server/chi/request/`.
- `references/<package>/<branch>.md` owns an optional usage branch. Redis topologies, manual decorator chains, factory wrapping, and optional server integrations must remain independently loadable when their tasks differ.
- `recipes/<goal>.md` owns one cross-package integration goal. Include the wiring needed for that goal without reproducing complete package guides. Point to deeper package options only under explicit conditions.
- `examples/<goal>/` holds larger runnable examples. Point to one only when the caller needs the complete example. Keep short fragments beside the explanation they illustrate.
- `references/local-source.md` is conditional source-discovery guidance. Load it for investigation, undocumented behavior, API verification, or a discrepancy with the application's installed code. Ordinary usage does not require repository exploration.

Keep a concept's definition, caveats, defaults, errors, and lifecycle instructions together. A dependency import does not justify loading that dependency's complete guide. Combine multi-goal reference sets and read each shared reference once. Ask a focused clarification when missing information changes the selected branch.

## Coverage records

`coverage.json` maps each inventoried package and supported public branch to a destination, example classification, and verification evidence. It also records existing package READMEs and exported APIs. A destination string is a plan until its file exists and its checks pass. Do not create empty guides to make planned paths resolve.

Use these states consistently:

- `planned` means the replacement or its proof is not complete. An existing file alone does not change this state.
- `verified` means the destination exists and recorded evidence demonstrates the documented behavior. Include the command, result, and scope of each relevant check.
- `blocked` means an identified dependency or behavior prevents completion. Record the blocker and preserve unverified criteria.

Group trivial exports only when they share behavior. Give distinct construction, configuration, lifecycle, errors, or extension behavior its own branch or a clearly identified example. Record limitations rather than presenting every export as recommended usage.

Classify short snippets as `fragment`. They are not independently compilable examples. Classify complete examples as `runnable`, with a real example path and evidence for the checks they require. Mark Fx examples with `"fx": true`. Each check records `command`, `scope`, and `result`. Runnable checks also record `kind` as `compile`, `runtime`, or `graph`. Fx examples must include their providers, interface bindings, groups, configuration inputs, startup hooks, and shutdown ownership. Show a shorter direct-construction alternative where supported.

Compilation, Fx graph validation, and runtime behavior are separate claims. Record each separately. A graph check does not prove startup or shutdown. Service-backed behavior needs controlled real dependencies. A complete HTTP flow needs the actual listening transport and application composition, not an isolated handler substitute. Report unavailable tooling or dependencies as verification gaps, not passes.

## Routing evidence

`routing.json` holds task descriptions, required references, excluded references, conditional follow-ups, clarification behavior, and deduplication expectations. Paths there are planned identifiers until authored. A conditional follow-up is justified only when its stated condition holds. Exclusions apply to the base task, not a later task that explicitly requests the excluded capability.

Evaluate from a consumer context containing only the distributed skill. Record actual reference reads and resulting guidance or runnable examples, together with the model and repeated-run results. Check required knowledge, unrelated loads, clarification, and deduplication. Do not require one heading order or one exact wording.

Each evaluation records its `case` ID, `model`, transcript or result location in `evidence`, actual reference `reads`, `consumer_only` flag, and `result`. Record a `clarification` when the case requires one. Keep separate records for repeated runs.

The `evaluations` list starts empty. Structural validation and link checks are not LLM routing evidence. Until the router exists and behavioral runs are recorded, report routing evaluation as not run. Preserve the gap rather than inferring success from the index.

## Acceptance commands and token budgets

Run `make check-bricks-skill` for utility tests and structural acceptance. CI runs the same target. To run only the dependency-free utility tests, use `python3 -m unittest discover -s scripts -p 'test_check_bricks_skill.py'`.

Run `python3 scripts/check-bricks-skill.py` during authoring. It checks structural records, actual links, example classifications, and token measurements while allowing explicitly planned destinations. Run `python3 scripts/check-bricks-skill.py --final` before declaring replacement coverage complete. Final acceptance requires verified coverage and routing evaluation; planned or unverified work must fail that gate.

The initial soft targets are 1,000 tokens for the router body, 2,000 for one focused reference, and 4,000 for a routine task's combined documentation load. Estimate tokens as the UTF-8 byte length divided by four, rounded up. This is a repeatable estimate, not a model tokenizer count. Measure the router body without frontmatter. Deduplicate selected references and include the router when reporting combined task loads. Report conditional loads separately. Exclude larger runnable examples unless the task requests them.

An overrun prompts investigation, not automatic failure. Split optional branches before cutting required explanations or compressing sentences into shorthand. Missing planned documents have no measured size and must not appear as zero-cost completed references.

Run the repository's applicable checks as well. Record each as passed, failed, or not run with its reason. Keep the authored prose, coverage map, routing cases, example claims, and measurement report consistent after revisions.

## Defects and README migration

Keep suspected defects in the inventory as verification leads until current source and focused checks establish their behavior. A verified limitation is documentation evidence, not permission to change the framework. Propose a blocking code fix separately and obtain explicit approval before implementation. Revalidate affected examples after an approved fix.

Retain package READMEs until useful replacement coverage, example verification, and inbound-link updates pass. Unverified runtime scenarios block removal of the affected README. Preserve the root project overview. Redirect affected root, glossary, and other documentation pointers to valid replacement content only at the migration gate.
