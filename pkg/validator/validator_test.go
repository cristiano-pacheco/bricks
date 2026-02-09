package validator_test

import (
	"testing"

	"github.com/cristiano-pacheco/bricks/pkg/validator"
)

type User struct {
	Name  string `validate:"required,min=3"`
	Email string `validate:"required,email"`
	Age   int    `validate:"gte=18"`
}

func TestNew(t *testing.T) {
	v, err := validator.New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if v == nil {
		t.Fatal("New() returned nil validator")
	}
}

func TestValidate(t *testing.T) {
	v, err := validator.New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	tests := []struct {
		name    string
		user    User
		wantErr bool
	}{
		{
			name:    "valid user",
			user:    User{Name: "John Doe", Email: "john@example.com", Age: 25},
			wantErr: false,
		},
		{
			name:    "invalid name",
			user:    User{Name: "Jo", Email: "john@example.com", Age: 25},
			wantErr: true,
		},
		{
			name:    "invalid email",
			user:    User{Name: "John", Email: "invalid", Age: 25},
			wantErr: true,
		},
		{
			name:    "invalid age",
			user:    User{Name: "John", Email: "john@example.com", Age: 15},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotErr := v.Validate(tt.user)
			if (gotErr != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", gotErr, tt.wantErr)
			}
		})
	}
}

func TestValidateVar(t *testing.T) {
	v, err := validator.New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	tests := []struct {
		name    string
		value   any
		tag     string
		wantErr bool
	}{
		{
			name:    "valid email",
			value:   "test@example.com",
			tag:     "email",
			wantErr: false,
		},
		{
			name:    "invalid email",
			value:   "invalid",
			tag:     "email",
			wantErr: true,
		},
		{
			name:    "valid min",
			value:   "hello",
			tag:     "min=3",
			wantErr: false,
		},
		{
			name:    "invalid min",
			value:   "hi",
			tag:     "min=3",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotErr := v.ValidateVar(tt.value, tt.tag)
			if (gotErr != nil) != tt.wantErr {
				t.Errorf("ValidateVar() error = %v, wantErr %v", gotErr, tt.wantErr)
			}
		})
	}
}

func TestEngine(t *testing.T) {
	v, err := validator.New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	engine := v.Engine()
	if engine == nil {
		t.Fatal("Engine() returned nil")
	}
}

func TestTranslator(t *testing.T) {
	v, err := validator.New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	trans := v.Translator()
	if trans == nil {
		t.Fatal("Translator() returned nil")
	}
}
