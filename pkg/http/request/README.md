# HTTP Request

High-performance JSON request parser with built-in security features for Go HTTP handlers.

## Features

- 🔒 **Security**: Content-Type validation, configurable body size limits, unknown field rejection
- ⚡ **Performance**: Direct streaming decode, optimized error handling
- 🛡️ **Protection**: DoS prevention, CSRF mitigation, single JSON value enforcement
- 📝 **Developer-friendly**: Clear error messages, detailed validation

## Installation

```bash
go get github.com/cristiano-pacheco/bricks
```

## Usage

### Basic Usage

```go
package main

import (
    "net/http"
    
    "github.com/cristiano-pacheco/bricks/pkg/http/request"
)

type CreateUserRequest struct {
    Name  string `json:"name"`
    Email string `json:"email"`
}

func createUserHandler(w http.ResponseWriter, r *http.Request) {
    var req CreateUserRequest
    
    // Parse and validate JSON request with default 1MB limit
    if err := request.ReadJSON(w, r, &req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    
    // Process request...
}
```

### Custom Body Size Limit

```go
func uploadHandler(w http.ResponseWriter, r *http.Request) {
    var data UploadData
    
    // Allow up to 10MB for this specific endpoint
    maxSize := int64(10 * 1024 * 1024) // 10MB
    if err := request.ReadJSONWithMaxSize(w, r, &data, maxSize); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    
    // Process upload...
}
```

## Security Features

### Content-Type Validation

Automatically validates that requests have `Content-Type: application/json` header to prevent CSRF attacks:

```bash
# ✅ Valid
curl -H "Content-Type: application/json" -d '{"name":"John"}' http://localhost:8080/api/users

# ✅ Valid with charset
curl -H "Content-Type: application/json; charset=utf-8" -d '{"name":"John"}' http://localhost:8080/api/users

# ❌ Rejected
curl -H "Content-Type: application/x-www-form-urlencoded" -d '{"name":"John"}' http://localhost:8080/api/users
```

### Body Size Limits

Prevents DoS attacks by limiting request body size:

- Default: 1MB (`DefaultMaxBodySize`)
- Configurable per endpoint with `ReadJSONWithMaxSize()`

### Unknown Fields Rejection

Rejects requests with unexpected fields to prevent data injection:

```go
type User struct {
    Name string `json:"name"`
}

// ❌ This will be rejected:
// {"name": "John", "admin": true}
```

### Single JSON Value

Ensures only one JSON value is processed, preventing injection of trailing data:

```bash
# ❌ Rejected - multiple JSON values
{"name":"John"}{"admin":true}
```

## Error Handling

The package provides detailed, security-conscious error messages:

| Situation | Error Message |
|-----------|---------------|
| Malformed JSON | `request body contains malformed JSON` |
| Empty body | `request body must not be empty` |
| Wrong Content-Type | `Content-Type header is not application/json` |
| Body too large | `request body must not exceed X bytes` |
| Unknown field | `request body contains unknown field "fieldname"` |
| Invalid type | `request body contains invalid value for field "fieldname"` |
| Multiple values | `request body must contain only a single JSON value` |

## Configuration

### Constants

```go
const (
    // DefaultMaxBodySize is 1MB
    DefaultMaxBodySize = 1_048_576
)
```

### Functions

- `ReadJSON(w, r, dst)` - Parse JSON with default 1MB limit
- `ReadJSONWithMaxSize(w, r, dst, maxBytes)` - Parse JSON with custom size limit

## Best Practices

1. **Always check errors**: Never ignore the error returned by `ReadJSON`
2. **Use appropriate limits**: Adjust max body size based on endpoint needs
3. **Validate business logic**: This package handles JSON parsing; add your own validation for business rules
4. **Use with structured types**: Define clear struct types with JSON tags

## Example with Validation

```go
import (
    "net/http"
    
    "github.com/cristiano-pacheco/bricks/pkg/http/request"
    "github.com/cristiano-pacheco/bricks/pkg/http/response"
)

type CreateUserRequest struct {
    Name  string `json:"name"`
    Email string `json:"email"`
}

func (r *CreateUserRequest) Validate() error {
    if r.Name == "" {
        return errors.New("name is required")
    }
    if !strings.Contains(r.Email, "@") {
        return errors.New("invalid email format")
    }
    return nil
}

func createUserHandler(w http.ResponseWriter, r *http.Request) {
    var req CreateUserRequest
    
    // Parse JSON
    if err := request.ReadJSON(w, r, &req); err != nil {
        response.Error(w, err)
        return
    }
    
    // Validate business logic
    if err := req.Validate(); err != nil {
        response.Error(w, err)
        return
    }
    
    // Process...
}
```

## License

MIT
