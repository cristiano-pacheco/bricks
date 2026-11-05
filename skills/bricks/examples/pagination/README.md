# Pagination example

Load this example when you need a complete executable demonstration of strict pagination parsing and separate size policy. For ordinary API usage, read the [pagination guide](../../references/paginator/usage.md).

[main.go](main.go) uses Bricks and standard-library imports only. It has no Fx graph, configuration files, database, HTTP server, or lifecycle resources. It prints successful parsing followed by a 100-item cap. It also prints page-specific errors for zero, negative, fractional, text, and overflowing inputs. Other branches show whitespace, first-value selection for repeated parameters, unchecked defaults, fallback, disabled capping, invalid-page offset versus limit, metadata, empty totals, and trimmed filters.

In a consumer module that already depends on Bricks, copy `main.go` into an otherwise empty executable directory and run `go run .`. No framework checkout or external skill is needed. From a Bricks checkout, use `go run ./skills/bricks/examples/pagination`.

Expected examples include these results:

- `page=2&per_page=250` becomes page 2 and size 100, with offset 100 and limit 100.
- `page=0` produces a page error. `per_page=0` produces a size error.
- Missing values with defaults 0 and -1 return those values without a parse error.
- An invalid page with size 20 has offset 0 but limit 20.
- Metadata for count 41, page 3, and size 20 contains total pages 3.
- Whitespace-only filters return nil; `"  active  "` becomes `"active"`.

These are source-derived expectations, not captured execution output. This program prints behavior; it is not an assertion-based test suite. Compilation and execution were waived by the user for ticket 04. No checks were run. The complete HTTP response composition remains later-ticket work.
