# HTTP Response

Optimized JSON response helpers for Go HTTP handlers with high performance and minimal allocations.

## Features

- ⚡ **High Performance**: Direct streaming encoding, zero-copy header handling
- 🎯 **Flexible**: With or without envelope wrapper, custom headers support
- 🔧 **Error Handling**: `ErrorHandler` interface for structured error responses (validation, errs.Error, unknown)
- 📦 **Framework Agnostic**: Works with standard `http.ResponseWriter`
- 🔌 **FX**: `response.Module` for Uber FX dependency injection

## Installation

```bash
go get github.com/cristiano-pacheco/bricks
```

## Usage

### Basic JSON Response

```go
package main

import (
    "net/http"
    
    "github.com/cristiano-pacheco/bricks/pkg/http/response"
)

type User struct {
    ID   int    `json:"id"`
    Name string `json:"name"`
}

func getUserHandler(w http.ResponseWriter, r *http.Request) {
    user := User{ID: 1, Name: "John Doe"}
    
    // Send JSON response with envelope
    response.JSON(w, http.StatusOK, user, nil)
}
```

**Response:**
```json
{
  "data": {
    "id": 1,
    "name": "John Doe"
  }
}
```

### Raw JSON Response (Without Envelope)

For better performance when envelope is not needed:

```go
func getUserHandler(w http.ResponseWriter, r *http.Request) {
    user := User{ID: 1, Name: "John Doe"}
    
    // Send JSON response without envelope wrapper
    response.JSONRaw(w, http.StatusOK, user, nil)
}
```

**Response:**
```json
{
  "id": 1,
  "name": "John Doe"
}
```

### Custom Headers

```go
func downloadHandler(w http.ResponseWriter, r *http.Request) {
    data := map[string]string{"file": "document.pdf"}
    
    headers := http.Header{}
    headers.Set("X-Request-ID", "abc123")
    headers.Set("X-Rate-Limit", "100")
    
    response.JSON(w, http.StatusOK, data, headers)
}
```

### Error Responses

```go
import (
    "net/http"
    
    "github.com/cristiano-pacheco/bricks/pkg/errs"
    "github.com/cristiano-pacheco/bricks/pkg/http/response"
)

// Create at startup or inject via response.Module
handler := response.NewErrorHandler(validator, log)

func createUserHandler(w http.ResponseWriter, r *http.Request) {
    err := errs.New("INVALID_EMAIL", "invalid email format", http.StatusBadRequest, nil)
    handler.Error(w, err)
}
```

**Response (errs.Error):**
```json
{
  "error": {
    "code": "INVALID_EMAIL",
    "message": "invalid email format"
  }
}
```

**Response (validation errors):**
```json
{
  "error": {
    "code": "INVALID_ARGUMENT",
    "message": "request has invalid fields",
    "details": [
      {"field": "email", "message": "must be a valid email address"}
    ]
  }
}
```

**Response (unknown error):**
```json
{
  "error": {
    "code": "internal_server_error",
    "message": "Internal server error"
  }
}
```

### FX Module

```go
import (
    "github.com/cristiano-pacheco/bricks/pkg/http/response"
    "github.com/cristiano-pacheco/bricks/pkg/validator"
    "go.uber.org/fx"
)

app := fx.New(
    response.Module, // requires validator.Module and logger.Logger
)
```

### No Content Response

```go
func deleteUserHandler(w http.ResponseWriter, r *http.Request) {
    // Delete user...
    
    response.NoContent(w)
}
```

## API Reference

### Functions

#### `JSON[T any](w http.ResponseWriter, status int, data T, headers http.Header) error`

Sends a JSON response with envelope wrapper `{"data": ...}`.

#### `JSONRaw[T any](w http.ResponseWriter, status int, data T, headers http.Header) error`

Sends a JSON response without envelope wrapper.

#### `NoContent(w http.ResponseWriter)`

Sends a 204 No Content response.

#### `NewErrorHandler(validate validator.Validator, log logger.Logger) ErrorHandler`

Creates an ErrorHandler. Validator and logger may be nil. If logger is nil, `log.Default()` is used for marshal/write failures.

### Types

#### `ErrorHandler`

```go
type ErrorHandler interface {
    Error(w http.ResponseWriter, err error)
}
```

Writes errors to HTTP responses as JSON. Handles: validation errors (422), `errs.Error` (custom status), unknown errors (500).

#### `Envelope`

```go
type Envelope map[string]any
```

Wrapper for JSON responses. Created automatically by `JSON()` function.

### FX

#### `Module`

`fx.Module` that provides `ErrorHandler`. Requires `validator.Module` and `logger.Logger` in the fx graph.

## Performance Characteristics

| Operation | Allocations | Notes |
|-----------|-------------|-------|
| `JSON()` | ~2-3 | Envelope map + encoder |
| `JSONRaw()` | ~1-2 | Just encoder, no envelope |
| `ErrorHandler.Error()` | ~2-3 | Similar to JSON |
| Header copy | 0 | Zero-copy iteration |

## Best Practices

### 1. Choose the Right Function

```go
response.JSON(w, http.StatusOK, users, nil)
response.JSONRaw(w, http.StatusOK, metrics, nil)
response.NoContent(w)
handler.Error(w, err)
```

### 2. Always Pass Headers (even if nil)

```go
response.JSON(w, http.StatusOK, data, nil)

headers := http.Header{}
headers.Set("X-Custom", "value")
response.JSON(w, http.StatusOK, data, headers)
```

### 3. Use Structured Errors

```go
// unknown errors return 500 with generic message
handler.Error(w, errors.New("user not found"))

// errs.Error preserves status and code
err := errs.New("USER_NOT_FOUND", "user not found", http.StatusNotFound, nil)
handler.Error(w, err)
```

### 4. Combine with Request Package

```go
import (
    "github.com/cristiano-pacheco/bricks/pkg/http/request"
    "github.com/cristiano-pacheco/bricks/pkg/http/response"
)

func handler(w http.ResponseWriter, r *http.Request) {
    var req CreateUserRequest
    
    if err := request.ReadJSON(w, r, &req); err != nil {
        handler.Error(w, err)
        return
    }
    
    user, err := createUser(req)
    if err != nil {
        handler.Error(w, err)
        return
    }
    
    response.JSON(w, http.StatusCreated, user, nil)
}
```

## Complete Example

```go
package main

import (
    "net/http"
    
    "github.com/cristiano-pacheco/bricks/pkg/http/request"
    "github.com/cristiano-pacheco/bricks/pkg/http/response"
    "github.com/cristiano-pacheco/bricks/pkg/logger"
    "github.com/cristiano-pacheco/bricks/pkg/validator"
    "github.com/go-chi/chi/v5"
)

type User struct {
    ID    int    `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

func main() {
    v, _ := validator.New()
    log := logger.MustNewWithOptions(logger.WithLevel("info"))
    errHandler := response.NewErrorHandler(v, log)
    
    r := chi.NewRouter()
    r.Get("/users/{id}", getUser)
    r.Post("/users", createUser(errHandler))
    r.Delete("/users/{id}", deleteUser)
    
    http.ListenAndServe(":8080", r)
}

func getUser(w http.ResponseWriter, r *http.Request) {
    user := User{ID: 1, Name: "John", Email: "john@example.com"}
    response.JSON(w, http.StatusOK, user, nil)
}

func createUser(h response.ErrorHandler) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        var user User
        
        if err := request.ReadJSON(w, r, &user); err != nil {
            h.Error(w, err)
            return
        }
        
        user.ID = 1
        response.JSON(w, http.StatusCreated, user, nil)
    }
}

func deleteUser(w http.ResponseWriter, r *http.Request) {
    response.NoContent(w)
}
```

## Framework Compatibility

This package works with any Go HTTP framework:

- ✅ Chi
- ✅ Gorilla Mux
- ✅ Echo (with `echo.Context.Response().Writer`)
- ✅ Fiber (using adaptor)
- ✅ Gin (with `gin.Context.Writer`)
- ✅ Standard library `net/http`

## License

MIT
