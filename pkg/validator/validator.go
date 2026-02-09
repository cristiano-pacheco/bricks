package validator

import (
	"errors"

	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	lib_validator "github.com/go-playground/validator/v10"
	en_translations "github.com/go-playground/validator/v10/translations/en"
)

// Validator provides struct validation with translation support
type Validator interface {
	// Validate validates a struct
	Validate(s any) error
	// ValidateVar validates a single variable
	ValidateVar(field any, tag string) error
	// Struct validates a struct (alias for Validate)
	Struct(s any) error
	// Var validates a single variable (alias for ValidateVar)
	Var(field any, tag string) error
	// Engine returns the underlying validator engine
	Engine() *lib_validator.Validate
	// Translator returns the universal translator
	Translator() ut.Translator
}

type validator struct {
	engine     *lib_validator.Validate
	translator ut.Translator
}

// New creates a new Validator with English translations pre-configured
func New() (*validator, error) {
	// Create validator instance
	v := lib_validator.New(lib_validator.WithRequiredStructEnabled())

	// Setup English translator
	en := en.New()
	uni := ut.New(en, en)
	trans, found := uni.GetTranslator("en")
	if !found {
		return nil, errors.New("translator not found")
	}

	// Register default English translations
	if err := en_translations.RegisterDefaultTranslations(v, trans); err != nil {
		return nil, err
	}

	return &validator{
		engine:     v,
		translator: trans,
	}, nil
}

// Validate validates a struct and returns translated error messages
func (v *validator) Validate(s any) error {
	return v.engine.Struct(s)
}

// ValidateVar validates a single variable with translated error messages
func (v *validator) ValidateVar(field any, tag string) error {
	return v.engine.Var(field, tag)
}

// Struct validates a struct (alias for Validate)
func (v *validator) Struct(s any) error {
	return v.engine.Struct(s)
}

// Var validates a single variable (alias for ValidateVar)
func (v *validator) Var(field any, tag string) error {
	return v.engine.Var(field, tag)
}

// Engine returns the underlying validator engine for advanced usage
func (v *validator) Engine() *lib_validator.Validate {
	return v.engine
}

// Translator returns the universal translator for custom translations
func (v *validator) Translator() ut.Translator {
	return v.translator
}
