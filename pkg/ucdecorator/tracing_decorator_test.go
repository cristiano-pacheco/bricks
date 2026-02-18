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

type TracingDecoratorTestSuite struct {
	suite.Suite
	sut      ucdecorator.UseCase[string, string]
	baseMock *mocks.MockUseCase[string, string]
}

func (s *TracingDecoratorTestSuite) SetupTest() {
	s.baseMock = mocks.NewMockUseCase[string, string](s.T())
	s.sut = ucdecorator.WithTracing(s.baseMock, "TestUseCase.Execute")
}

func TestTracingDecoratorSuite(t *testing.T) {
	suite.Run(t, new(TracingDecoratorTestSuite))
}

func (s *TracingDecoratorTestSuite) TestExecute_Success_PropagatesOutputToCallerAndCallsBase() {
	// Arrange
	ctx := context.Background()
	s.baseMock.On("Execute", mock.Anything, "input").Return("output", nil)

	// Act
	result, err := s.sut.Execute(ctx, "input")

	// Assert
	s.Require().NoError(err)
	s.Equal("output", result)
}

func (s *TracingDecoratorTestSuite) TestExecute_Error_PropagatesErrorToCallerAndCallsBase() {
	// Arrange
	ctx := context.Background()
	expectedErr := errors.New("use case failed")
	s.baseMock.On("Execute", mock.Anything, "input").Return("", expectedErr)

	// Act
	result, err := s.sut.Execute(ctx, "input")

	// Assert
	s.Require().ErrorIs(err, expectedErr)
	s.Empty(result)
}
