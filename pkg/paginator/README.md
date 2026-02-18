# Paginator Package

A pagination utilities package for handling pagination parameters in HTTP requests with validation, normalization, and metadata generation.

## Installation

```bash
go get github.com/cristiano-pacheco/bricks/pkg/paginator
```

## Usage

### Parsing Query Parameters

```go
package main

import (
    "net/http"
    "github.com/cristiano-pacheco/bricks/pkg/paginator"
)

func ListUsersHandler(w http.ResponseWriter, r *http.Request) {
    // Parse pagination from query params with defaults
    params, err := paginator.ParseQueryParams(r.URL.Query(), 1, 20)
    if err != nil {
        // Handle ErrInvalidPage or ErrInvalidPerPage
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    
    // Use params.Offset() and params.Limit() for database queries
    users := repository.ListUsers(params.Offset(), params.Limit())
    
    // ... return response
}
```

### Normalizing Parameters

```go
params := paginator.NormalizeParams(
    page,           // user-provided page
    perPage,        // user-provided per_page
    1,              // default page
    20,             // default per_page
    100,            // max per_page
)
```

### Generating Pagination Metadata

```go
totalCount := int64(150)
perPage := 20

metadata := paginator.Metadata{
    TotalCount: totalCount,
    Page:       params.Page,
    PerPage:    params.PerPage,
    TotalPages: paginator.TotalPages(totalCount, perPage),
}
```

## Features

- 📄 **Query Parameter Parsing**: Parse `page` and `per_page` from URL query strings
- ✅ **Validation**: Built-in validation with sentinel errors
- 🔧 **Normalization**: Normalize parameters with defaults and max limits
- 📊 **Metadata Generation**: Generate pagination metadata for API responses
- 🛠️ **Helper Functions**: Utility functions for offset/limit calculations

## API

### Types

#### `Params`

```go
type Params struct {
    Page    int
    PerPage int
}
```

#### `Metadata`

```go
type Metadata struct {
    TotalCount int64 `json:"total_count"`
    Page       int   `json:"page"`
    PerPage    int   `json:"per_page"`
    TotalPages int   `json:"total_pages"`
}
```

### Functions

#### `ParseQueryParams(query url.Values, defaultPage int, defaultPerPage int) (Params, error)`

Parses pagination parameters from URL query values. Returns `ErrInvalidPage` or `ErrInvalidPerPage` if validation fails.

```go
query := url.Values{"page": []string{"2"}, "per_page": []string{"50"}}
params, err := paginator.ParseQueryParams(query, 1, 20)
// params.Page = 2, params.PerPage = 50
```

#### `NormalizeParams(page int, perPage int, defaultPage int, defaultPerPage int, maxPerPage int) Params`

Normalizes pagination parameters with defaults and maximum limits.

```go
params := paginator.NormalizeParams(0, 1000, 1, 20, 100)
// params.Page = 1, params.PerPage = 100 (capped at max)
```

#### `TotalPages(totalCount int64, perPage int) int`

Calculates the total number of pages based on total count and items per page.

```go
totalPages := paginator.TotalPages(41, 20) // returns 3
```

#### `OptionalTrimmedStringPtr(value string) *string`

Returns a trimmed string pointer, or nil if the string is empty/whitespace only.

```go
ptr := paginator.OptionalTrimmedStringPtr("  hello ") // returns *string("hello")
ptr := paginator.OptionalTrimmedStringPtr("   ")     // returns nil
```

### Methods

#### `params.Offset() int`

Returns the database offset for pagination (0-based).

```go
params := paginator.Params{Page: 3, PerPage: 20}
offset := params.Offset() // returns 40
```

#### `params.Limit() int`

Returns the database limit for pagination.

```go
params := paginator.Params{Page: 3, PerPage: 20}
limit := params.Limit() // returns 20
```

## Errors

```go
var (
    ErrInvalidPage    = errors.New("invalid page")
    ErrInvalidPerPage = errors.New("invalid per_page")
)
```

## HTTP Handler Example

```go
package main

import (
    "encoding/json"
    "net/http"
    
    "github.com/cristiano-pacheco/bricks/pkg/paginator"
)

type ListResponse struct {
    Data       []User            `json:"data"`
    Pagination paginator.Metadata `json:"pagination"`
}

type User struct {
    ID   int    `json:"id"`
    Name string `json:"name"`
}

type Handler struct {
    repository *UserRepository
}

func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
    // Parse and validate pagination params
    params, err := paginator.ParseQueryParams(r.URL.Query(), 1, 20)
    if err != nil {
        respondError(w, http.StatusBadRequest, err)
        return
    }
    
    // Normalize with max limit
    params = paginator.NormalizeParams(params.Page, params.PerPage, 1, 20, 100)
    
    // Get paginated data
    users, totalCount, err := h.repository.List(params.Offset(), params.Limit())
    if err != nil {
        respondError(w, http.StatusInternalServerError, err)
        return
    }
    
    // Build response with metadata
    response := ListResponse{
        Data: users,
        Pagination: paginator.Metadata{
            TotalCount: totalCount,
            Page:       params.Page,
            PerPage:    params.PerPage,
            TotalPages: paginator.TotalPages(totalCount, params.PerPage),
        },
    }
    
    json.NewEncoder(w).Encode(response)
}
```

## Query Parameter Format

The package expects pagination parameters in the query string:

```
GET /api/users?page=2&per_page=50
```

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `page` | int | 1 | The page number (1-indexed) |
| `per_page` | int | 20 | Items per page |

## Response Format

```json
{
    "data": [...],
    "pagination": {
        "total_count": 150,
        "page": 2,
        "per_page": 20,
        "total_pages": 8
    }
}
```
