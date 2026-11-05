# Tune Redis configuration

Use this reference after selecting a topology. It owns the shared `redis.Config` field contract. To create an application first, select only the [standalone](standalone.md), [cluster](cluster.md), [Sentinel](sentinel.md), or [failover URL](failover.md) guide matching your deployment.

## Apply a setting

1. Start with the selected topology's minimal `config/base.yaml`. Preserve its connection selector.
2. Copy the shared fields you need from that topology's full-field example into `app.redis`. The keys are lowercase Go field names, not snake case. Duration values use strings such as `500ms`, `5s`, or `30m`.
3. Run the topology's program and inspect its public set/read result. Configuration loading, successful graph construction, and a real connection are different checks. No execution results are claimed for these examples.

The loader only unmarshals values. `NewClient` applies `SetDefaults` to a copy and then validates it. A standalone `Validate()` call does not apply defaults. Use exact lowercase `type` values. Type validation tolerates some spelling variations, but the constructor switch does not normalize them.

## Shared field reference

All fields below are optional. Topology-specific required fields are explained in the selected setup guide. Defaults describe `NewClient`, not a zero Go struct or an unconstructed Fx `redis.Config` value.

| YAML field | Type | Default or zero behavior, units, and constraints |
| --- | --- | --- |
| `type` | string | Empty becomes `single_node`. Use exactly `single_node`, `cluster`, `sentinel`, or `failover`. |
| `password` | string | Empty. Overrides parsed URL password in standalone and cluster. Sets Redis data-node authentication in all branches. Any nonempty value forces TLS even with `enabletls: false`. It is not the Sentinel password. |
| `maxretries` | integer | Zero becomes 3. Negative fails Bricks validation. Controls the number of initial ping attempts, including the first attempt, and is also passed to go-redis as its command retry setting. Cannot disable retries with zero or -1 here. |
| `minretrybackoff` | duration string | Zero becomes 8ms. Initial ping backoff starts here and doubles. Also passed to go-redis for command retries. Use nonnegative durations. |
| `maxretrybackoff` | duration string | Zero becomes 512ms. Caps later initial-ping backoffs. Also passed to go-redis. Keep at least the minimum backoff. Bricks does not validate the ordering. |
| `dialtimeout` | duration string | Zero becomes 5s. Passed to go-redis and used as each initial ping context timeout. Use positive durations. |
| `readtimeout` | duration string | Zero becomes 3s. Socket-read timeout. go-redis supports the special duration values `-1ns` for no timeout and `-2ns` to skip setting deadlines. These are sentinels, not negative second counts. Prefer positive durations. |
| `writetimeout` | duration string | Zero becomes 3s. Socket-write timeout, with the same negative sentinels as read timeout. |
| `contexttimeoutenabled` | bool | False. Enables go-redis's use of context deadlines for socket operations. The full example selects true. Pass bounded contexts to commands as well. |
| `poolfifo` | bool | False selects last-in-first-out idle connection reuse. True selects first-in-first-out. |
| `poolsize` | integer | Zero becomes 10. Negative fails. Passed as go-redis base pool size, per node for cluster. This is not a Bricks-enforced hard connection cap. |
| `pooltimeout` | duration string | Zero becomes 4s. Wait for a pool connection. Use a positive duration. It does not bound all command work. |
| `minidleconns` | integer | Zero becomes 2. Negative fails. Zero cannot request no minimum idle connections through this constructor. |
| `maxidleconns` | integer | Zero means no idle-connection limit in go-redis. Negative fails Bricks validation. Use a value compatible with your pool plan. |
| `connmaxidletime` | duration string | Zero becomes 30m. Idle age for retiring connections. go-redis treats negative as disabled. |
| `connmaxlifetime` | duration string | Zero becomes 1h. Connection age for retirement. go-redis treats nonpositive as disabled, but Bricks replaces zero first. |
| `enabletls` | bool | False. True creates a TLS configuration with minimum TLS 1.2. A password can also force TLS. Standalone and cluster URL parsing can supply TLS independently. |
| `tlsskipverify` | bool | False. Only used when explicit `enabletls` is true. Keep false to verify certificate chains and names. True removes that protection. |
| `tlsservername` | string | Empty. Only used when explicit `enabletls` is true. Set to the certificate server name when the dial address differs. |
| `protocol` | integer | Bricks passes zero through. The selected go-redis defaults values below 2 to RESP3. Use 2 or 3. Bricks does not validate other integers. |
| `disableindentity` | bool | False. The misspelling is the actual Bricks field/key. Passed to go-redis `DisableIdentity` to disable automatic client identity information. |
| `identitysuffix` | string | Empty. Passed to go-redis's client identity suffix setting. |
| `namespace` | string | Empty. `WithNamespace` concatenates this string, a colon, and the key. Commands do not apply it automatically. |
| `enablemetrics` | bool | False. Collects only constructor ping-attempt metrics locally. It does not observe ordinary commands or publish Prometheus metrics. |
| `commandtimeout` | duration string | Zero becomes 5s, but the value is never applied to command execution. Use command contexts and actual socket timeout fields instead. |
| `unstablemode` | bool | False. Accepted and retained in Config, but never used to build any client. |

Bricks validates only type, required connection inputs, standalone DB range, and nonnegative pool counts and retry count. It does not generally validate timeout ranges, pool relationships, TLS certificate availability, addresses, or credentials. Constructor success requires a reachable Redis node because it pings immediately.

## Fields that differ by topology

Each selected guide owns its keys, defaults, and limitations. This summary helps identify ignored inputs without requiring every topology guide.

| Field | Used by |
| --- | --- |
| `url` | Required for standalone and failover. Optional cluster URL takes precedence over `clusteraddrs`. Ignored for Sentinel. |
| `db` | Standalone, Sentinel, failover. Default 0. Only standalone validates 0 through 15. Ignored for cluster. |
| `clusteraddrs` | Cluster only, unless a URL takes precedence. |
| `sentineladdrs` | Sentinel only. Failover derives one Sentinel address from its URL instead. |
| `mastername` | Required for Sentinel and failover. |
| `sentinelpassword`, `sentinelusername` | Sentinel only. Not forwarded by the failover URL branch. |
| `routebylatency`, `routerandomly` | Sentinel forwards these bools, but the selected go-redis `NewFailoverClient` panics when either is true. Keep false. |
| `replicaonly` | Sentinel only. True connects to replicas, which cannot serve this guide's write operation. |
| `maxredirects`, `readonly` | Cluster only. |

The separate tracing and Prometheus pool modules also read `app.redis.tracing.enabled` and `app.redis.metrics.enabled`, respectively. Both default false. These nested keys are not fields of `redis.Config`; `enablemetrics` does not replace them. They require Go-side tracing lifecycle or registry dependencies and are not part of the basic client examples.

## Configuration boundaries

`redis.Module` and `redis.ClientModule` both load `app.redis` and provide a `redis.Config` value plus one client. Do not install both. To load the subtree without Fx, call `config.New[redis.Config](config.WithPath("app.redis"))` and pass `Get()` to `NewClient`.

Environment overlays merge into the base file. A value beginning with `env://` resolves an environment variable before decoding. Missing variables become empty strings, not secret-loading errors. For overlay rules and whole-document loading, read the [configuration guide](../config/usage.md) only when you need those choices. Keep deployment secrets outside committed YAML.
