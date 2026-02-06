# HTTP Response

Optimized JSON response helpers for Go HTTP handlers with high performance and minimal allocations.

## Features

- ⚡ **High Performance**: Direct streaming encoding, zero-copy header handling
- 🎯 **Flexible**: With or without envelope wrapper, custom headers support
- 🔧 **Error Handling**: Structured error responses with HTTP status codes
- 📦 **Framework Agnostic**: Works with standard `http.ResponseWriter`

## Installation

```bash
go get github.com/cristiano-pacheco/bricks/pkg/http/response
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
    "github.com/cristiano-pacheco/bricks/pkg/errs"
    "github.com/cristiano-pacheco/bricks/pkg/http/response"
)

func createUserHandler(w http.ResponseWriter, r *http.Request) {
    // Structured error with status code
    err := errs.New("invalid email format").
        WithStatus(http.StatusBadRequest).
        WithCode("invalid_email")
    
    response.Error(w, err)
}
```

**Response:**
```json
{
  "error": {
    "code": "invalid_email",
    "message": "invalid email format",
    "status": 400
  }
}
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

- **Performance**: Uses `json.NewEncoder` for direct streaming
- **Headers**: Custom headers are added efficiently
- **Use case**: Standard API responses with consistent structure

#### `JSONRaw[T any](w http.ResponseWriter, status int, data T, headers http.Header) error`

Sends a JSON response without envelope wrapper.

- **Performance**: Faster than `JSON()` as it skips envelope creation
- **Use case**: When you need maximum performance or control over response structure

#### `Error(w http.ResponseWriter, err error)`

Sends an error response with proper HTTP status code.

- **Smart detection**: Automatically detects `errs.Error` types
- **Fallback**: Returns 500 Internal Server Error for unknown errors
- **Security**: Hides internal error details for non-structured errors

#### `NoContent(w http.ResponseWriter)`

Sends a 204 No Content response.

- **Use case**: DELETE operations, successful updates without response body

### Types

#### `Envelope`

```go
type Envelope map[string]any
```

Wrapper for JSON responses. Created automatically by `JSON()` function.

## Performance Characteristics

| Operation | Allocations | Notes |
|-----------|-------------|-------|
| `JSON()` | ~2-3 | Envelope map + encoder |
| `JSONRaw()` | ~1-2 | Just encoder, no envelope |
| `Error()` | ~2-3 | Similar to JSON |
| Header copy | 0 | Zero-copy iteration |

## Best Practices

### 1. Choose the Right Function

```go
// ✅ Use JSON() for consistent API structure
response.JSON(w, http.StatusOK, users, nil)

// ✅ Use JSONRaw() for high-frequency endpoints
response.JSONRaw(w, http.StatusOK, metrics, nil)

// ✅ Use Error() for all error responses
response.Error(w, err)

// ✅ Use NoContent() for operations without response body
response.NoContent(w)
```

### 2. Always Pass Headers (even if nil)

```go
// ✅ Good - explicit nil
response.JSON(w, http.StatusOK, data, nil)

// ✅ Good - with headers
headers := http.Header{}
headers.Set("X-Custom", "value")
response.JSON(w, http.StatusOK, data, headers)
```

### 3. Use Structured Errors

```go
// ❌ Bad - loses context
response.Error(w, errors.New("user not found"))

// ✅ Good - structured with status
err := errs.New("user not found").
    WithStatus(http.StatusNotFound).
    WithCode("user_not_found")
response.Error(w, err)
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
        response.Error(w, err)
        return
    }
    
    user, err := createUser(req)
    if err != nil {
        response.Error(w, err)
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
    
    "github.com/cristiano-pacheco/bricks/pkg/errs"
    "github.com/cristiano-pacheco/bricks/pkg/http/request"
    "github.com/cristiano-pacheco/bricks/pkg/http/response"
    "github.com/go-chi/chi/v5"
)

type User struct {
    ID    int    `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

func main() {
    r := chi.NewRouter()
    
    r.Get("/users/{id}", getUser)
    r.Post("/users", createUser)
    r.Delete("/users/{id}", deleteUser)
    
    http.ListenAndServe(":8080", r)
}

func getUser(w http.ResponseWriter, r *http.Request) {
    // Simulate database query
    user := User{ID: 1, Name: "John", Email: "john@example.com"}
    
    response.JSON(w, http.StatusOK, user, nil)
}

func createUser(w http.ResponseWriter, r *http.Request) {
    var user User
    
    if err := request.ReadJSON(w, r, &user); err != nil {
        response.Error(w, err)
        return
    }
    
    // Simulate save
    user.ID = 1
    
    response.JSON(w, http.StatusCreated, user, nil)
}

func deleteUser(w http.ResponseWriter, r *http.Request) {
    // Simulate delete
    
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
