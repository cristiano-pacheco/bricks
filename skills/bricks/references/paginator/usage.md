# Paginate a list

This guide parses page-number query parameters, caps the page size, and builds pagination metadata. The package needs no constructor, Fx module, YAML, or lifecycle hooks. Your application chooses the defaults and maximum size.

## 1. Choose your page-size policy

The example uses page `1`, size `20`, and a maximum size of `100`.

First call `ParseQueryParams` to reject malformed or nonpositive explicit values. Then call `NormalizeParams` to cap a valid size. Parsing alone does not enforce a maximum.

## 2. Copy the complete example

Create `main.go` in an executable directory of a module that already depends on Bricks:

```go
package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"

	"github.com/cristiano-pacheco/bricks/pkg/paginator"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	query, err := url.ParseQuery("page=3&per_page=20")
	if err != nil {
		return err
	}
	params, err := paginator.ParseQueryParams(query, 1, 20)
	if err != nil {
		return err
	}
	params = paginator.NormalizeParams(params.Page, params.PerPage, 1, 20, 100)

	fmt.Printf("offset=%d limit=%d\n", params.Offset(), params.Limit())

	// Replace this example count with the total matching rows before pagination.
	totalCount := int64(41)
	metadata := paginator.Metadata{
		TotalCount: totalCount,
		Page:       params.Page,
		PerPage:    params.PerPage,
		TotalPages: paginator.TotalPages(totalCount, params.PerPage),
	}
	encoded, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	fmt.Println(string(encoded))
	return nil
}
```

This program calculates query inputs and metadata. It does not query a database or start an HTTP server.

## 3. Run it and check the result

Run `go run .` from the directory containing `main.go`. Expected output:

```text
offset=40 limit=20
{"total_count":41,"page":3,"per_page":20,"total_pages":3}
```

Change the query to `page=2&per_page=250`. The normalized size becomes `100`, with offset `100` and limit `100`. Change it to `page=0` or `per_page=0` to see parsing fail before normalization.

The [distributed pagination example](../../examples/pagination/README.md) demonstrates additional boundary cases.

## 4. Use the parameters in your application

In an HTTP adapter, pass `r.URL.Query()` to `ParseQueryParams`. On a parsing error, reject the request rather than continuing with zero-valued parameters. Use standard-library `errors.Is` to distinguish `paginator.ErrInvalidPage` from `paginator.ErrInvalidPerPage`.

After normalization, pass `params.Offset()` and `params.Limit()` to your application's list query. Count all matching records before applying the page slice, then use that count in `Metadata`. A count of the current page's rows is not the total count.

The package does not execute a count query, write a response, choose a response envelope, or invoke the Bricks validator. Successful query parsing does not validate other application input.

## Query fields and parsing behavior

| Query field | Accepted explicit values | When absent or blank |
| --- | --- | --- |
| `page` | An integer greater than zero | Uses the `defaultPage` argument. |
| `per_page` | An integer greater than zero | Uses the `defaultPerPage` argument. |

There are no package-wide defaults. The parser uses the arguments you supply without checking that they are positive.

`ParseQueryParams(query url.Values, defaultPage, defaultPerPage int) (Params, error)` reads the first value for each key through `url.Values.Get` and trims surrounding whitespace. Empty and whitespace-only values use the defaults.

Explicit values pass through `strconv.Atoi`. Text, fractions, zero, negatives, and integers outside the platform's `int` range fail. A leading plus sign is accepted. URL-encode a literal plus sign because query decoding normally converts `+` to a space.

On failure, the function returns zero-valued `Params` and either `ErrInvalidPage` or `ErrInvalidPerPage`. It discards the underlying parse error. It parses page first, so an invalid page wins when both fields are invalid.

## Choose capping, rejection, or fallback

`NormalizeParams(page, perPage, defaultPage, defaultPerPage, maxPerPage int) Params` replaces a page or size below one with its corresponding default. A positive maximum caps the resulting size, including a size taken from the default. A zero or negative maximum disables capping. Invalid defaults can still produce invalid parameters.

Use normalization alone for internal integer inputs when fallback is intentional. For example:

```go
params := paginator.NormalizeParams(0, -1, 1, 200, 100)
// params.Page is 1 and params.PerPage is 100.
```

At an HTTP boundary, parse first so invalid explicit strings remain errors. If your product requires rejecting oversized pages instead of capping them, compare the parsed `PerPage` with your maximum and return an application error before using it.

## Offsets and numeric limits

`Params` has integer `Page` and `PerPage` fields.

- With positive values, `Offset()` returns `(Page-1)*PerPage` and `Limit()` returns `PerPage`.
- `Offset()` returns zero when either field is below one.
- `Limit()` returns zero only when `PerPage` is below one. A bad page with size `20` still has limit `20`.

These defensive returns do not prove that parameters are valid. Offset multiplication uses `int` without overflow detection. Capping size alone does not bound the offset. If callers can supply extreme page numbers, enforce an application page or offset limit before multiplication.

`TotalPages(totalCount int64, perPage int) int` returns zero for a nonpositive count or size. Otherwise it calculates the ceiling through `float64` division, then converts to `int`. This is not exact integer arithmetic for all extreme counts. Keep counts within your application's supported numeric range.

## Metadata behavior

`Metadata` contains `TotalCount`, `Page`, `PerPage`, and `TotalPages`. Its JSON field names are `total_count`, `page`, `per_page`, and `total_pages`.

The struct does not clamp a requested page to the last page. For zero matching records, total pages is `0`, but the requested page remains unchanged. Requests beyond the end can produce an empty list.

## Optional string filters

Use `OptionalTrimmedStringPtr` when your application treats a missing filter and a blank filter alike:

```go
filter := paginator.OptionalTrimmedStringPtr("  active  ")
// filter points to "active".
blank := paginator.OptionalTrimmedStringPtr("   ")
// blank is nil.
```

The helper trims whitespace without changing the original string. It does not validate whether a nonempty filter is allowed.
