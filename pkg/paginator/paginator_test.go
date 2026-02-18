package paginator_test

import (
	"errors"
	"net/url"
	"testing"

	"github.com/cristiano-pacheco/bricks/pkg/paginator"
)

func TestParamsOffsetAndLimit(t *testing.T) {
	params := paginator.Params{Page: 3, PerPage: 20}
	if got := params.Offset(); got != 40 {
		t.Fatalf("Offset() = %d, want 40", got)
	}
	if got := params.Limit(); got != 20 {
		t.Fatalf("Limit() = %d, want 20", got)
	}
}

func TestParseQueryParams(t *testing.T) {
	query := url.Values{"page": []string{"2"}, "per_page": []string{"50"}}
	got, err := paginator.ParseQueryParams(query, 1, 20)
	if err != nil {
		t.Fatalf("ParseQueryParams() unexpected error = %v", err)
	}
	if got.Page != 2 || got.PerPage != 50 {
		t.Fatalf("ParseQueryParams() = %+v, want page=2 per_page=50", got)
	}
}

func TestParseQueryParamsInvalidErrors(t *testing.T) {
	_, err := paginator.ParseQueryParams(url.Values{"page": []string{"x"}}, 1, 20)
	if !errors.Is(err, paginator.ErrInvalidPage) {
		t.Fatalf("expected ErrInvalidPage, got %v", err)
	}

	_, err = paginator.ParseQueryParams(url.Values{"per_page": []string{"0"}}, 1, 20)
	if !errors.Is(err, paginator.ErrInvalidPerPage) {
		t.Fatalf("expected ErrInvalidPerPage, got %v", err)
	}
}

func TestNormalizeParams(t *testing.T) {
	got := paginator.NormalizeParams(0, 1000, 1, 20, 100)
	if got.Page != 1 || got.PerPage != 100 {
		t.Fatalf("NormalizeParams() = %+v, want page=1 per_page=100", got)
	}
}

func TestOptionalTrimmedStringPtr(t *testing.T) {
	if got := paginator.OptionalTrimmedStringPtr("   "); got != nil {
		t.Fatalf("OptionalTrimmedStringPtr(spaces) = %v, want nil", *got)
	}
	got := paginator.OptionalTrimmedStringPtr("  hello ")
	if got == nil || *got != "hello" {
		t.Fatalf("OptionalTrimmedStringPtr() = %v, want hello", got)
	}
}

func TestTotalPages(t *testing.T) {
	if got := paginator.TotalPages(41, 20); got != 3 {
		t.Fatalf("TotalPages() = %d, want 3", got)
	}
	if got := paginator.TotalPages(0, 20); got != 0 {
		t.Fatalf("TotalPages() zero total = %d, want 0", got)
	}
}
