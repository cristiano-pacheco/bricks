# Validator

Struct validation with `go-playground/validator`.

## Installation

```bash
go get github.com/cristiano-pacheco/bricks/pkg/validator
```

## Usage

### Standalone

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/cristiano-pacheco/bricks/pkg/validator"
)

type User struct {
    Name  string `validate:"required,min=3,max=50"`
    Email string `validate:"required,email"`
    Age   int    `validate:"required,gte=18,lte=120"`
}

func main() {
    // Create validator instance
    v, err := validator.New()
    if err != nil {
        log.Fatal(err)
    }
    
    // Validate struct
    user := User{
        Name:  "Jo",
        Email: "invalid-email",
        Age:   15,
    }
    
    if err := v.Validate(user); err != nil {
        fmt.Println("Validation errors:", err)
        return
    }
}
```

### With Uber FX

```go
package main

import (
    "context"
    "fmt"
    
    "github.com/cristiano-pacheco/bricks/pkg/validator"
    "go.uber.org/fx"
)

type User struct {
    Name  string `validate:"required,min=3"`
    Email string `validate:"required,email"`
}

type UserService struct {
    validator validator.Validator
}

func NewUserService(v validator.Validator) *UserService {
    return &UserService{validator: v}
}

func (s *UserService) CreateUser(user User) error {
    if err := s.validator.Validate(user); err != nil {
        return fmt.Errorf("invalid user: %w", err)
    }
    return nil
}

func main() {
    fx.New(
        validator.Module,
        fx.Provide(NewUserService),
        fx.Invoke(func(service *UserService) error {
            return service.CreateUser(User{Name: "John", Email: "john@example.com"})
        }),
    ).Run()
}
```

## API

```go
type Validator interface {
    Validate(s any) error
    ValidateVar(field any, tag string) error
    Struct(s any) error
    Var(field any, tag string) error
    Engine() *lib_validator.Validate
    Translator() ut.Translator
}
```

## Tags

Common tags: `required`, `email`, `url`, `min`, `max`, `len`, `eq`, `ne`, `gt`, `gte`, `lt`, `lte`, `alpha`, `alphanum`, `numeric`, `omitempty`, `dive`.

[Full list](https://pkg.go.dev/github.com/go-playground/validator/v10#hdr-Baked_In_Validators_and_Tags)

## Examples

### Nested Struct Validation

```go
type Address struct {
    Street string `validate:"required"`
    City   string `validate:"required"`
    ZIP    string `validate:"required,len=5"`
}

type User struct {
    Name    string  `validate:"required"`
    Address Address `validate:"required,dive"`
}

user := User{
    Name: "John",
    Address: Address{Street: "123 Main St", City: "", ZIP: "123"},
}
err := v.Validate(user)
```

### Slice Validation

```go
type Product struct {
    Tags []string `validate:"required,min=1,dive,required,min=2"`
}

product := Product{Tags: []string{"", "ab", "valid-tag"}}
err := v.Validate(product)
```

### Error Handling

```go
err := v.Struct(user)
if validationErrs, ok := err.(lib_validator.ValidationErrors); ok {
    for _, e := range validationErrs {
        fmt.Printf("Field: %s, Tag: %s\n", e.Field(), e.Tag())
    }
}
```

### Custom Validator

```go
v, _ := validator.New()
v.Engine().RegisterValidation("is_awesome", func(fl lib_validator.FieldLevel) bool {
    return fl.Field().String() == "awesome"
})

type Thing struct {
    Value string `validate:"required,is_awesome"`
}
```

## HTTP Integration

```go
import (
    "encoding/json"
    "net/http"
    
    "github.com/cristiano-pacheco/bricks/pkg/validator"
)

type Handler struct {
    validator validator.Validator
}

func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
    var user User
    if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
        http.Error(w, "Invalid JSON", http.StatusBadRequest)
        return
    }
    
    if err := h.validator.Validate(user); err != nil {
        w.WriteHeader(http.StatusBadRequest)
        json.NewEncoder(w).Encode(map[string]any{"error": err})
        return
    }
    
    w.WriteHeader(http.StatusCreated)
}
```

## License

MIT
