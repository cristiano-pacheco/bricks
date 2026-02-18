package ucdecorator

import (
	"context"
	"time"

	"github.com/cristiano-pacheco/bricks/pkg/metrics"
)

type metricsDecorator[T any, R any] struct {
	base       UseCase[T, R]
	metrics    metrics.UseCaseMetrics
	metricName string
}

func withMetrics[T any, R any](
	base UseCase[T, R],
	useCaseMetrics metrics.UseCaseMetrics,
	metricName string,
) UseCase[T, R] {
	return &metricsDecorator[T, R]{
		base:       base,
		metrics:    useCaseMetrics,
		metricName: metricName,
	}
}

func (decorator *metricsDecorator[T, R]) Execute(ctx context.Context, input T) (R, error) {
	start := time.Now()
	output, err := decorator.base.Execute(ctx, input)

	decorator.metrics.ObserveDuration(decorator.metricName, time.Since(start))
	if err != nil {
		decorator.metrics.IncError(decorator.metricName)
		return output, err
	}

	decorator.metrics.IncSuccess(decorator.metricName)
	return output, nil
}
