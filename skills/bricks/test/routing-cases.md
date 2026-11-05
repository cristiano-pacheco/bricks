# Check Bricks routing

Use these cases when changing the skill's routes or tutorial boundaries. They check document selection and required behavior. They do not claim that generated Go or a consumer application ran.

## 1. Walk the selected path

For each prompt, start with `SKILL.md`. Record the actual files read, why each was needed, and which related guides stayed excluded. Follow conditional pointers only when the prompt supplies their trigger. Read a shared destination once across several goals.

| Prompt | Select | Exclude unless another contract needs work |
| --- | --- | --- |
| Load a typed YAML subtree | `references/config/usage.md` | Database, server, go-backend |
| Connect PostgreSQL using Fx | `references/database/usage.md` | Full configuration, SQLite, metrics, tracing |
| Tune PostgreSQL pool size or TLS | `references/database/configuration.md` | Unchanged startup recipe and other backends |
| Configure Redis Sentinel without a specific API | `references/redis/sentinel.md` | Standalone, cluster, failover-URL guides |
| Start a small HTTP application | `references/http/server/chi/usage.md` | Database, decorators, observability |
| Decode, validate, execute, and respond | `recipes/http-flow.md` | Decorator guides and component guides already covered by the recipe |
| Expose an existing operation over HTTP | `recipes/existing-operation-http.md` | New use-case policy, database setup, full server bootstrap |
| Add metrics to an application | Ask whether the task means operation, pool, or listener metrics | Unrelated metrics branches before clarification |
| Explicitly decorate an operation | Manual or factory guide according to the requested setup | Changes to neighboring direct operations |
| Verify changed decorator flags | `references/ucdecorator/verification.md` and the selected setup if unverified | Unrelated exporter or storage tests |
| Run PostgreSQL migrations as a command | `recipes/migration-command.md` and the SQL contribution branch it needs | HTTP server and application GORM pool |
| Test with one SQL directory | `references/itestkit/migrations.md` | Embedded-module bridge and Redis |
| Test with several embedded SQL owners | `references/itestkit/embedded-migrations.md` and required kit setup | A second migration route or test-only schema copies |
| Add database pool metrics and GORM tracing | Both selected recipes, shared setup once | Redis, use-case decorators, unrelated listeners |
| Application selects v0.1.21 | `references/compatibility.md`, then bounded source verification for the affected contract | Current-checkout APIs without version evidence |
| Example differs from the selected dependency | `references/local-source.md` and the affected package only | Machine-wide searches, downloads, dependency edits |

## 2. Check skill boundaries

Bricks works without go-backend. Framework-only questions do not load the backend workflow. When both skills apply, project and backend architecture rules control application layout, responsibilities, naming, and test targets. Bricks supplies framework APIs and setup.

Default HTTP examples invoke concrete use cases directly. Application constructors return concrete values and Fx binds interfaces. A decorator tutorial applies only to explicitly selected decoration or an existing decorated operation under change.

## 3. Check tutorial completeness

For each changed path, confirm that the reader can identify prerequisites, filenames, imports, required providers, configuration, expected outcomes, and cleanup. A targeted existing-application recipe may name verified prerequisites instead of recreating them. Mark fragments as fragments.

Keep Go inside Markdown. Keep separate assets limited to configuration and supporting data. A self-contained recipe must not require reading every imported component's guide.

## 4. Separate document checks from execution

Check local links and cross-skill destinations. Estimate context with a stated method and include every file actually read. For execution proof, copy the inline setup into a temporary consumer and record compilation, Fx validation, startup, runtime requests, and shutdown separately. Do not turn expected output or static links into a passing runtime claim.
