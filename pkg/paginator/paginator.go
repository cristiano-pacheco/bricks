package paginator

import (
	"errors"
	"math"
	"net/url"
	"strconv"
	"strings"
)

type Params struct {
	Page    int
	PerPage int
}

func (p Params) Offset() int {
	if p.Page < 1 || p.PerPage < 1 {
		return 0
	}

	return (p.Page - 1) * p.PerPage
}

func (p Params) Limit() int {
	if p.PerPage < 1 {
		return 0
	}

	return p.PerPage
}

func ParseQueryParams(query url.Values, defaultPage int, defaultPerPage int) (Params, error) {
	page, err := parsePositiveIntQueryValue(query.Get("page"), defaultPage)
	if err != nil {
		return Params{}, ErrInvalidPage
	}

	perPage, err := parsePositiveIntQueryValue(query.Get("per_page"), defaultPerPage)
	if err != nil {
		return Params{}, ErrInvalidPerPage
	}

	return Params{
		Page:    page,
		PerPage: perPage,
	}, nil
}

func NormalizeParams(
	page int,
	perPage int,
	defaultPage int,
	defaultPerPage int,
	maxPerPage int,
) Params {
	normalizedPage := page
	if normalizedPage < 1 {
		normalizedPage = defaultPage
	}

	normalizedPerPage := perPage
	if normalizedPerPage < 1 {
		normalizedPerPage = defaultPerPage
	}
	if maxPerPage > 0 && normalizedPerPage > maxPerPage {
		normalizedPerPage = maxPerPage
	}

	return Params{
		Page:    normalizedPage,
		PerPage: normalizedPerPage,
	}
}

func OptionalTrimmedStringPtr(value string) *string {
	trimmedValue := strings.TrimSpace(value)
	if trimmedValue == "" {
		return nil
	}

	return &trimmedValue
}

func parsePositiveIntQueryValue(value string, fallback int) (int, error) {
	trimmedValue := strings.TrimSpace(value)
	if trimmedValue == "" {
		return fallback, nil
	}

	parsedValue, err := strconv.Atoi(trimmedValue)
	if err != nil {
		return 0, err
	}
	if parsedValue <= 0 {
		return 0, errors.New("value must be greater than zero")
	}

	return parsedValue, nil
}

type Metadata struct {
	TotalCount int64 `json:"total_count"`
	Page       int   `json:"page"`
	PerPage    int   `json:"per_page"`
	TotalPages int   `json:"total_pages"`
}

func TotalPages(totalCount int64, perPage int) int {
	if totalCount <= 0 || perPage <= 0 {
		return 0
	}

	return int(math.Ceil(float64(totalCount) / float64(perPage)))
}
