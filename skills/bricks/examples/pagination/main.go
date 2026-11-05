package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"

	"github.com/cristiano-pacheco/bricks/pkg/paginator"
)

func run() error {
	for _, rawQuery := range []string{
		"",
		"page=2&per_page=250",
		"page=%20%202%20&per_page=%2020%20",
		"page=0",
		"page=-1",
		"per_page=0",
		"per_page=-2",
		"page=1.5",
		"page=x&per_page=x",
		"page=999999999999999999999999999999999999",
		"page=2&page=3&per_page=20",
	} {
		query, err := url.ParseQuery(rawQuery)
		if err != nil {
			return fmt.Errorf("parse query %q: %w", rawQuery, err)
		}

		params, err := paginator.ParseQueryParams(query, 1, 20)
		if err != nil {
			fmt.Printf("query=%q pageError=%t perPageError=%t\n", rawQuery,
				errors.Is(err, paginator.ErrInvalidPage),
				errors.Is(err, paginator.ErrInvalidPerPage))

			continue
		}

		params = paginator.NormalizeParams(params.Page, params.PerPage, 1, 20, 100)
		fmt.Printf("query=%q params=%+v offset=%d limit=%d\n",
			rawQuery, params, params.Offset(), params.Limit())
	}

	uncheckedDefaults, err := paginator.ParseQueryParams(url.Values{}, 0, -1)
	if err != nil {
		return fmt.Errorf("parse empty query with supplied defaults: %w", err)
	}

	fmt.Printf("unchecked defaults=%+v\n", uncheckedDefaults)
	fmt.Printf("fallback and cap=%+v\n", paginator.NormalizeParams(0, -1, 1, 200, 100))
	fmt.Printf("cap disabled=%+v\n", paginator.NormalizeParams(1, 250, 1, 20, 0))

	invalidPage := paginator.Params{Page: 0, PerPage: 20}
	fmt.Printf("invalid page offset=%d limit=%d\n", invalidPage.Offset(), invalidPage.Limit())

	params := paginator.Params{Page: 3, PerPage: 20}
	metadata := paginator.Metadata{
		TotalCount: 41,
		Page:       params.Page,
		PerPage:    params.PerPage,
		TotalPages: paginator.TotalPages(41, params.PerPage),
	}

	encoded, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("encode pagination metadata: %w", err)
	}

	fmt.Println(string(encoded))
	fmt.Printf("zero count pages=%d invalid size pages=%d\n",
		paginator.TotalPages(0, 20), paginator.TotalPages(41, 0))
	fmt.Printf("blank filter is nil=%t\n", paginator.OptionalTrimmedStringPtr("   ") == nil)

	filter := paginator.OptionalTrimmedStringPtr("  active  ")
	if filter != nil {
		fmt.Printf("filter=%q\n", *filter)
	}

	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
