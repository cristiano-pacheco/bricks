# Contribute rooted locale filesystems

Give the loader files it can read as `en.json` or `pt-BR.json` at the filesystem root. This support package wraps `fs.FS`; it does not choose a language or load JSON by itself. Filesystems and Fx groups are Go-only inputs, not YAML fields.

## Add a module's locale contribution

1. Create `locales/en.json` beside your Go package:

```json
{"checkout": {"paid": "Paid"}}
```

   Add `locales/pt-BR.json` when you want that module's Portuguese translation:

```json
{"checkout": {"paid": "Pago"}}
```

2. In that package, add this provider fragment. It needs `embed`, `io/fs`, `github.com/cristiano-pacheco/bricks/pkg/i18n/locale`, and `go.uber.org/fx` imports:

```go
//go:embed locales/*.json
var files embed.FS

func provideLocales() (locale.FileSystem, error) {
    root, err := fs.Sub(files, "locales")
    return locale.New(root), err
}

var Module = fx.Module("checkout-locales",
    fx.Provide(fx.Annotate(provideLocales,
        fx.ResultTags(`group:"locale_filesystems"`))),
)
```

   This fragment belongs at package scope. Add the module to the same Fx application as `i18n.Module`. An ungrouped provider is not a locale contribution. `fs.Sub` removes the directory prefix; wrapping `files` directly would make `en.json` invisible to the loader.

3. For a complete runnable application rather than a module fragment, copy the [distributed two-filesystem example](../../../examples/i18n-translation/README.md) into `cmd/messages`. It includes application configuration, logger ownership, both filesystem providers, and a translation consumer. Run:

```sh
APP_CONFIG_DIR=cmd/messages/config APP_ENV=local go run ./cmd/messages
```

   Its `extra` directory is the checkout contribution above. Expect `Pago` as the last output line. The first contribution supplies the other messages. No filesystem shutdown API exists; the application flushes its logger.

## Public wrapper and alternatives

`FileSystem` embeds `fs.FS`. `New(fsys fs.FS) FileSystem` stores it unchanged. The equivalent literal is `locale.FileSystem{FS: fsys}`. Both expose the embedded filesystem's `Open` method. The constructor does not validate roots or reject nil. Supply a nonnil filesystem; nil can panic when a loader later calls it.

For disk-backed data, use `locale.New(os.DirFS("locales"))` in the provider instead. For an existing `fs.FS`, use `fs.Sub` when its root is above the JSON files and return the error. The service reads files during construction; disk changes do not refresh an existing translation service. Supply trusted language configuration because the loader appends `.json` to the literal configured stem.

## Contributions and collisions

The loader attempts one `<language>.json` per filesystem. It merges successful files by domain and key, retaining unrelated keys. A later filesystem in its input slice overwrites duplicate keys. Fx value groups do not promise slice order, so provider declaration order is not an override policy. Keep contributions disjoint or construct an explicitly ordered slice directly with the [service APIs](../service/usage.md).

Unreadable files and invalid JSON are skipped with debug logs. A successful file, even an empty object, counts as that locale being present. If some module lacks a requested locale while another supplies it, `Load` returns only the successful requested-language data, not a per-module English merge. `TranslationService` separately supplies English key fallback. Do not use loader `Load` alone as proof of a complete bilingual key set.
