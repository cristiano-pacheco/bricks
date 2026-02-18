package ucdecorator

import (
	"context"
	"time"

	"github.com/cristiano-pacheco/bricks/pkg/logger"
)

type debugDecorator[T any, R any] struct {
	base          UseCase[T, R]
	log           logger.Logger
	useCaseName   string
	decoratorName string
}

func withDebug[T any, R any](
	base UseCase[T, R],
	log logger.Logger,
	useCaseName string,
	decoratorName string,
) UseCase[T, R] {
	if log == nil {
		return base
	}

	return &debugDecorator[T, R]{
		base:          base,
		log:           log,
		useCaseName:   useCaseName,
		decoratorName: decoratorName,
	}
}

func (decorator *debugDecorator[T, R]) Execute(ctx context.Context, input T) (R, error) {
	startedAt := time.Now()
	decorator.log.Debug(
		"decorator execute start",
		logger.String("use_case", decorator.useCaseName),
		logger.String("decorator", decorator.decoratorName),
	)

	output, err := decorator.base.Execute(ctx, input)
	durationMs := time.Since(startedAt).Milliseconds()
	if err != nil {
		decorator.log.Debug(
			"decorator execute error",
			logger.String("use_case", decorator.useCaseName),
			logger.String("decorator", decorator.decoratorName),
			logger.Int64("duration_ms", durationMs),
			logger.Error(err),
		)
		return output, err
	}

	decorator.log.Debug(
		"decorator execute success",
		logger.String("use_case", decorator.useCaseName),
		logger.String("decorator", decorator.decoratorName),
		logger.Int64("duration_ms", durationMs),
	)
	return output, nil
}
