package validator_test

import (
	"fmt"

	"github.com/cristiano-pacheco/bricks/pkg/validator"
)

func ExampleNew() {
	v, _ := validator.New()

	type User struct {
		Name  string `validate:"required,min=3"`
		Email string `validate:"required,email"`
	}

	user := User{Name: "John", Email: "john@example.com"}
	err := v.Validate(user)
	fmt.Println(err)
	// Output: <nil>
}

func ExampleValidator_Validate() {
	v, _ := validator.New()

	type Product struct {
		Name  string  `validate:"required"`
		Price float64 `validate:"required,gt=0"`
	}

	product := Product{Name: "Widget", Price: 9.99}
	err := v.Validate(product)
	fmt.Println(err)
	// Output: <nil>
}

func ExampleValidator_ValidateVar() {
	v, _ := validator.New()

	email := "test@example.com"
	err := v.ValidateVar(email, "email")
	fmt.Println(err)
	// Output: <nil>
}
