package ucdecorator

import (
	"context"

	"github.com/cristiano-pacheco/bricks/pkg/logger"
)

type loggingDecorator[T any, R any] struct {
	base   UseCase[T, R]
	logger logger.Logger
	name   string
}

func withLogging[T any, R any](base UseCase[T, R], log logger.Logger, name string) UseCase[T, R] {
	if log == nil {
		return base
	}
	return &loggingDecorator[T, R]{
		base:   base,
		logger: log,
		name:   name,
	}
}

func (decorator *loggingDecorator[T, R]) Execute(ctx context.Context, input T) (R, error) {
	var output R
	var err error

	defer func() {
		if err != nil {
			decorator.logger.Error(decorator.name+" failed", logger.Error(err))
		} else {
			decorator.logger.Info(decorator.name + " succeeded")
		}
	}()

	output, err = decorator.base.Execute(ctx, input)
	return output, err
}
