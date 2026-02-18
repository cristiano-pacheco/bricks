package ucdecorator_test

import (
	"context"
	"errors"
	"testing"

	"github.com/cristiano-pacheco/bricks/pkg/ucdecorator"
	"github.com/cristiano-pacheco/bricks/test/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type MetricsDecoratorTestSuite struct {
	suite.Suite
	sut         ucdecorator.UseCase[string, string]
	baseMock    *mocks.MockUseCase[string, string]
	metricsMock *mocks.MockUseCaseMetrics
}

func (s *MetricsDecoratorTestSuite) SetupTest() {
	s.baseMock = mocks.NewMockUseCase[string, string](s.T())
	s.metricsMock = mocks.NewMockUseCaseMetrics(s.T())
	s.sut = ucdecorator.WithMetrics(s.baseMock, s.metricsMock, "create_user")
}

func TestMetricsDecoratorSuite(t *testing.T) {
	suite.Run(t, new(MetricsDecoratorTestSuite))
}

func (s *MetricsDecoratorTestSuite) TestExecute_Success_ObservesDurationAndIncrementsSuccess() {
	// Arrange
	ctx := context.Background()
	s.baseMock.On("Execute", mock.Anything, "input").Return("output", nil)
	s.metricsMock.On("ObserveDuration", "create_user", mock.Anything).Return()
	s.metricsMock.On("IncSuccess", "create_user").Return()

	// Act
	result, err := s.sut.Execute(ctx, "input")

	// Assert
	s.Require().NoError(err)
	s.Equal("output", result)
}

func (s *MetricsDecoratorTestSuite) TestExecute_Error_ObservesDurationAndIncrementsError() {
	// Arrange
	ctx := context.Background()
	expectedErr := errors.New("use case failed")
	s.baseMock.On("Execute", mock.Anything, "input").Return("", expectedErr)
	s.metricsMock.On("ObserveDuration", "create_user", mock.Anything).Return()
	s.metricsMock.On("IncError", "create_user").Return()

	// Act
	result, err := s.sut.Execute(ctx, "input")

	// Assert
	s.Require().ErrorIs(err, expectedErr)
	s.Empty(result)
}
