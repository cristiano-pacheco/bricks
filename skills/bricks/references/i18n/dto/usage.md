# Use the locale DTO without changing the file format

Represent locale metadata with `dto.Locale` in application-owned code. This DTO is not the JSON shape consumed by the Bricks locale loader. It has no YAML configuration or lifecycle.

## Construct a DTO

1. Create `cmd/locale-dto/main.go` in an application Go module with Bricks:

```go
package main

import (
    "encoding/json"
    "fmt"
    "os"

    "github.com/cristiano-pacheco/bricks/pkg/i18n/dto"
)

func run() error {
    value := dto.Locale{
        Code: "en",
        Domains: map[string]map[string]string{
            "catalog": {"title": "Products"},
        },
    }
    encoded, err := json.Marshal(value)
    if err != nil { return err }
    fmt.Println(string(encoded))
    return nil
}

func main() {
    if err := run(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}
```

2. Run `go run ./cmd/locale-dto`. Expect:

```json
{"code":"en","domains":{"catalog":{"title":"Products"}}}
```

3. When saving data for the Bricks loader, marshal `value.Domains`, not `value`. Its `en.json` expects `{"catalog":{"title":"Products"}}`. The filename supplies the language. There is no cleanup for these values.

## Public fields

`Locale.Code` is a string with JSON key `code`. `Locale.Domains` is `map[string]map[string]string` with JSON key `domains`. Both have ordinary Go zero values, an empty string and a nil map. Neither field has validation, a configuration tag, or a constructor. The loader never decodes this struct, so a DTO wrapper in a locale file can fail parsing and be skipped. Use [the translation guide](../usage.md) only when you need to load messages rather than carry metadata.
