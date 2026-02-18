package ucdecorator

import "context"

// UseCase is the base interface that all use cases implement.
type UseCase[T any, R any] interface {
	Execute(ctx context.Context, input T) (R, error)
}

// ErrorTranslator translates an error into a localized error.
type ErrorTranslator interface {
	TranslateError(err error) error
}
