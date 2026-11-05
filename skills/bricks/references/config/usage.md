# Configuration

Use `github.com/cristiano-pacheco/bricks/pkg/config` to load YAML into a typed `Config[T]`. `Get()` returns the stored value by value, not a deep copy. Loading is eager, not a watcher or live reload mechanism.

## Files, environment, and defaults

Every load requires a directory containing `base.yaml`. `APP_CONFIG_DIR` selects the directory, defaults to `config`, and accepts an absolute path or a path relative to the process working directory. The loader trims whitespace and cleans the path. A blank value uses the default.

`APP_ENV` selects an optional `<environment>.yaml` overlay. An unset or empty value uses `local`. Other values are trimmed and lowercased. A whitespace-only value becomes empty rather than `local`, so supply a real environment name. The loader merges the overlay after the base. An absent overlay is allowed; unreadable or invalid overlays fail loading. An overlay alone cannot replace the required base file.

Use `config` struct tags for field names. The loader does not apply application defaults or call a `Validate` method. Supply required values in YAML, then validate them in your application. Fields absent from the loaded document retain their Go zero values. Do not assume other packages' default constructors run during unmarshaling.

## Whole document or subtree

These are fragments inside a function, with the `config` package imported. Handle `err` before using `Get()`.

```go
type Service struct {
    Name string `config:"name"`
    Port int    `config:"port"`
}
type Document struct {
    App struct {
        Service Service `config:"service"`
    } `config:"app"`
}
whole, err := config.New[Document]()
// After checking err, use whole.Get().App.Service.
```

For only the service section, replace the load with this fragment:

```go
section, err := config.New[Service](config.WithPath("app.service"))
// After checking err, use section.Get().
```

Both fragments use this `base.yaml`:

```yaml
app:
  service:
    name: orders
    port: 8080
```

`WithPath` trims the path. An empty path selects the whole document. A missing nonempty path returns an error instead of an empty value. Nil options are ignored; later path options replace earlier ones.

## Environment-variable references

A string beginning with `env://` becomes the value of the named environment variable. Resolution traverses maps and lists in each YAML file before merging. An unset variable becomes an empty string, not a load error. A replacement may still fail typed unmarshaling, for example when an integer field receives an invalid string.

```yaml
app:
  service:
    name: env://SERVICE_NAME
    port: env://SERVICE_PORT
  origins:
    - env://ALLOWED_ORIGIN
```

Set `SERVICE_NAME`, `SERVICE_PORT`, and `ALLOWED_ORIGIN` before loading. This mechanism substitutes the entire string. It does not expand `${NAME}`, interpolate embedded `env://` text, or automatically overlay arbitrary environment variables onto YAML keys. Avoid printing loaded secrets.

## Fx providers

`config.Provide[Service]("app.service")` provides `config.Config[Service]`, not `Service`. Consume the wrapper and call `Get()`. Passing an empty path provides the whole document. A custom provider returning `config.New[Document]()` is also supported.

Fx constructors are lazy. Include a consumer or invoke that demands the configuration so loading and failures actually occur. Two unnamed providers of the same `config.Config[T]` collide even if their paths differ. Use different types or named Fx bindings when you need both.

For a complete configuration-and-logger Fx application with supplied YAML, explicit start and stop, and a structured error, read [the foundations example](../../examples/foundations/README.md). For configuration loading alone, the fragments above are sufficient.

## Failures

Use `errors.Is` for exported sentinels carried through wrapped errors:

- `ErrConfigDirNotFound` means the resolved directory is missing or is not a directory.
- `ErrConfigFileNotFound` means a requested YAML file is absent. A missing base fails; the loader suppresses this sentinel for an absent environment overlay.
- `ErrUnmarshalFailed` means decoding into `T` failed, for either the whole document or a subtree.

`ErrMissingConfigDir` exists but `New` does not return it. An empty directory setting selects `config` instead. Missing subtree, working-directory, permission, read, and YAML parse errors are other failures, not promises of a particular sentinel. Check returned errors instead of matching message strings.
