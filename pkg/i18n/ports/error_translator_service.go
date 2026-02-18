package ports

// ErrorTranslatorService translates typed module errors into localized messages.
type ErrorTranslatorService interface {
	TranslateError(err error) error
}
