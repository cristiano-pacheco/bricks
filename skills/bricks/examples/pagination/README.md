# Run the pagination example

This program shows how strict query parsing differs from page-size capping. It also prints metadata, offsets, limits, and optional filters. It needs no Fx graph, YAML, database, HTTP server, or lifecycle resources.

## 1. Copy the program

In a consumer module that already depends on Bricks, copy [main.go](main.go) into an empty executable directory. It imports only Bricks and the Go standard library.

For a shorter example you can read and copy in one place, use the [pagination guide](../../references/paginator/usage.md).

## 2. Run it

From the directory containing the copied file, run:

```sh
go run .
```

From a Bricks checkout, run this instead:

```sh
go run ./skills/bricks/examples/pagination
```

## 3. Check parsing and capping

The program loops over query strings. For each valid query, it parses with default page `1` and size `20`, then caps the size at `100`.

Find these expected lines in the output:

```text
query="" params={Page:1 PerPage:20} offset=0 limit=20
query="page=2&per_page=250" params={Page:2 PerPage:100} offset=100 limit=100
query="page=0" pageError=true perPageError=false
query="per_page=0" pageError=false perPageError=true
```

The distinction matters. An explicit size of `250` is valid and then capped. An explicit size of `0` is rejected before normalization.

The remaining queries demonstrate surrounding whitespace, repeated keys, negative values, fractions, text, and integer overflow. Repeated keys use the first value. If both fields are invalid, the page error wins.

## 4. Check defaults, offsets, and metadata

After the query loop, the program prints additional cases:

- Missing values with defaults `0` and `-1` return those defaults without a parse error. Your application must choose positive defaults.
- Normalization replaces invalid internal integer inputs with defaults and caps the resulting size. A maximum of `0` disables capping.
- A page of `0` with size `20` has offset `0` but limit `20`. These helper results do not prove the parameters are valid.

The metadata example uses count `41`, page `3`, and size `20`. Its expected JSON is:

```json
{"total_count":41,"page":3,"per_page":20,"total_pages":3}
```

A zero count or nonpositive page size produces zero total pages. A whitespace-only filter becomes nil; `"  active  "` becomes `"active"`.

## 5. Apply it to a list endpoint

Pass `r.URL.Query()` to `ParseQueryParams` in the HTTP adapter. Handle parsing errors before using the parameters. Normalize valid parameters when your policy caps sizes, then pass the offset and limit to your application's query.

Count all matching records before pagination when building metadata. This program does not demonstrate a database query or a complete HTTP response. Those are separate composition tasks.
