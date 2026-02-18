package ucdecorator

import (
	"context"

	"github.com/cristiano-pacheco/bricks/pkg/otel/trace"
)

type tracingDecorator[T any, R any] struct {
	base UseCase[T, R]
	name string
}

func withTracing[T any, R any](base UseCase[T, R], name string) UseCase[T, R] {
	return &tracingDecorator[T, R]{
		base: base,
		name: name,
	}
}

func (decorator *tracingDecorator[T, R]) Execute(ctx context.Context, input T) (R, error) {
	ctx, span := trace.Span(ctx, decorator.name)
	defer span.End()

	return decorator.base.Execute(ctx, input)
}
