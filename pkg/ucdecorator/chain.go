package ucdecorator

import (
	"github.com/cristiano-pacheco/bricks/pkg/logger"
	"github.com/cristiano-pacheco/bricks/pkg/metrics"
)

// Chain composes use case decorators in the expected execution order.
func Chain[T any, R any](
	handler UseCase[T, R],
	log logger.Logger,
	useCaseMetrics metrics.UseCaseMetrics,
	translator ErrorTranslator,
	metricName string,
	useCaseName string,
) UseCase[T, R] {
	return withLogging[T, R](
		withMetrics[T, R](
			withTracing[T, R](
				withTranslation[T, R](handler, translator),
				useCaseName,
			),
			useCaseMetrics,
			metricName,
		),
		log,
		useCaseName,
	)
}
