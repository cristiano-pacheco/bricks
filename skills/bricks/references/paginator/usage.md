# Pagination

Use `github.com/cristiano-pacheco/bricks/pkg/paginator` for page-number pagination. The package has no constructor, Fx module, configuration loader, or lifecycle. The application chooses positive defaults and its maximum page size.

## Strict query parsing

`ParseQueryParams(query url.Values, defaultPage, defaultPerPage int) (Params, error)` reads `page` and `per_page` through `url.Values.Get`, so repeated parameters use the first value. It trims surrounding whitespace. Missing, empty, and whitespace-only values use the supplied defaults without checking whether those defaults are positive.

Explicit values must parse through `strconv.Atoi` and be greater than zero. Text, fractions, zero, negatives, and integers outside the platform's `int` range fail. A leading plus sign is accepted by `Atoi`; this is integer parsing, not a digits-only grammar. URL-encode a literal plus sign because query decoding normally converts `+` to a space.

The function returns `ErrInvalidPage` or `ErrInvalidPerPage`, with zero-valued `Params` on failure. It discards the underlying parse error. Page parsing runs first, so an invalid page wins when both inputs are invalid. Compare with `errors.Is`.

Strict-parsing fragment, with standard-library `errors` and `net/url` imports:

```go
query := url.Values{"page": {" 2 "}, "per_page": {"250"}}
params, err := paginator.ParseQueryParams(query, 1, 20)
if errors.Is(err, paginator.ErrInvalidPage) {
	return err
}
if err != nil {
	return err
}
// params is Page=2, PerPage=250. Parsing has not capped it.
```

In an HTTP adapter, pass `r.URL.Query()` rather than constructing `url.Values`. Successful JSON decoding or query parsing does not validate other application input. This package does not invoke the Bricks validator.

## Normalization and page-size policy

`NormalizeParams(page, perPage, defaultPage, defaultPerPage, maxPerPage int) Params` is forgiving. It replaces page or size below one with the corresponding default. A positive `maxPerPage` caps the resulting size, including a size taken from the default. A zero or negative maximum disables capping. Defaults themselves are not repaired, so invalid defaults can still produce invalid parameters.

To reject malformed query input but cap valid sizes, parse first and then normalize:

```go
params, err := paginator.ParseQueryParams(query, 1, 20)
if err != nil {
	return err
}
params = paginator.NormalizeParams(params.Page, params.PerPage, 1, 20, 100)
// An explicit per_page=250 becomes 100. An explicit per_page=0 was rejected.
```

This is a fragment with `query` already supplied. Normalization alone is an alternative for internal integer inputs when fallback is intentional. It does not parse strings and must not silently replace strict parsing at an HTTP boundary. If product policy requires rejecting oversize pages instead of capping them, compare the parsed size with the application maximum and return an application error before using it. Parsing imposes no maximum size.

## Offsets, limits, and metadata

`Params` has integer `Page` and `PerPage` fields. With positive values, `Offset()` returns `(Page-1)*PerPage`, and `Limit()` returns `PerPage`. Offset is zero when either field is below one. Limit is zero only when `PerPage` is below one, regardless of Page. These defensive returns do not prove the parameters are valid.

Multiplication uses `int` without overflow detection. Capping page size alone does not bound the offset. Apply an application page or offset limit before multiplying if callers can supply extreme page numbers.

`TotalPages(totalCount int64, perPage int) int` returns zero when count or size is nonpositive. Otherwise it calculates the ceiling through `float64` division and converts it to `int`. It is not exact integer arithmetic for all extreme counts; keep totals within your supported numeric range.

`Metadata` is a data struct, not a count query or response writer. Populate it with the total matching records before page slicing. Its JSON fields are `total_count`, `page`, `per_page`, and `total_pages`. It does not clamp a requested page to the last page or prevent an empty result beyond the end.

Metadata fragment, with normalized `params` and a count from the application's query:

```go
metadata := paginator.Metadata{
	TotalCount: totalCount,
	Page:       params.Page,
	PerPage:    params.PerPage,
	TotalPages: paginator.TotalPages(totalCount, params.PerPage),
}
```

For page 3, size 20, and count 41, offset is 40, limit is 20, and total pages is 3. For zero matching records, total pages is 0 while the requested page remains unchanged. Selecting an HTTP response envelope is a separate composition concern.

## Optional string filters

`OptionalTrimmedStringPtr(value string) *string` trims whitespace and returns nil for an empty result. Otherwise it returns a pointer to the trimmed string. It neither mutates the original string nor checks filter validity. It intentionally treats an absent value and a blank value alike.

```go
filter := paginator.OptionalTrimmedStringPtr("  active  ")
// filter points to "active". Whitespace-only input returns nil.
```

For a complete program demonstrating invalid query input, zero and negative boundaries, fallback defaults, capping, offset/limit differences, metadata, and optional filters, read [the pagination example](../../examples/pagination/README.md). Load it only when a complete executable demonstration is needed. The snippets above are fragments. The complete example has not been compiled or executed; those checks were waived for this ticket.
