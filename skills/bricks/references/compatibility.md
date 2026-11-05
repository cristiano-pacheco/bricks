# Adapt earlier Bricks releases

Read when the application selects Bricks v0.1.21 or a tutorial uses an API absent from the selected dependency. Earlier go-backend examples targeted v0.1.21. This skill's main tutorials describe the maintained implementation and are not a promise that every release supports the same fields.

## 1. Resolve the actual dependency

Follow [local-source discovery](local-source.md). Use the application's effective version, workspace, replacement, or vendor source. Do not substitute a newer checkout or update the dependency merely to fit a tutorial.

## 2. Check version-sensitive contracts

The v0.1.21 source has these differences:

- Chi config uses required `port` and `metricsport` fields. The current tutorials' listener addresses and disabled-metrics setup are not equivalent fields in that release.
- Chi's Fx startup starts serving asynchronously and discards unexpected serving failures. A successful `app.Start` does not prove a bound listener.
- Redis modules provide `config.Config[redis.Config]`, while their constructors consume plain `redis.Config`. The process needs the config projection described by the Redis guides.
- Tracing decodes its public value-object `ExporterType` rather than the current module's YAML conversion. Verify decoding before supplying a string field. Build explicit exporter values in Go when required.
- Tracing normalizes a zero sample rate to 0.01. Do not apply the current YAML explicit-zero sampling behavior to that release. Use its actual disabled-export contract when export must be off.
- SQLite package and migration APIs in a newer checkout do not establish support in this release. Require the selected source to contain them before use.

These are source-inspection findings, not a full release compatibility matrix. Check the requested package and stop when its contract is supported.

## 3. Preserve application behavior

Adapt only the selected mechanism. Keep existing route semantics, test boundaries, and resource owners. Record configuration loading, compilation, startup, listener readiness, and runtime results separately. Do not claim old-version runtime support from a current-checkout test.
