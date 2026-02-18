package ucdecorator

import "context"

type translationDecorator[T any, R any] struct {
	base       UseCase[T, R]
	translator ErrorTranslator
}

func withTranslation[T any, R any](base UseCase[T, R], translator ErrorTranslator) UseCase[T, R] {
	if translator == nil {
		return base
	}

	return &translationDecorator[T, R]{
		base:       base,
		translator: translator,
	}
}

func (decorator *translationDecorator[T, R]) Execute(ctx context.Context, input T) (R, error) {
	output, err := decorator.base.Execute(ctx, input)
	if err != nil {
		return output, decorator.translator.TranslateError(err)
	}

	return output, nil
}
