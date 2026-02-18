package ucdecorator

import (
	"context"
	"errors"
	"testing"

	"github.com/cristiano-pacheco/bricks/test/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type LoggingDecoratorTestSuite struct {
	suite.Suite
	sut        UseCase[string, string]
	baseMock   *mocks.MockUseCase[string, string]
	loggerMock *mocks.MockLogger
}

func (s *LoggingDecoratorTestSuite) SetupTest() {
	s.baseMock = mocks.NewMockUseCase[string, string](s.T())
	s.loggerMock = mocks.NewMockLogger(s.T())
	s.sut = withLogging(s.baseMock, s.loggerMock, "TestUseCase.Execute")
}

func TestLoggingDecoratorSuite(t *testing.T) {
	suite.Run(t, new(LoggingDecoratorTestSuite))
}

func (s *LoggingDecoratorTestSuite) TestExecute_Success_DoesNotLogError() {
	// Arrange
	ctx := context.Background()
	s.baseMock.On("Execute", mock.Anything, "input").Return("output", nil)
	// No loggerMock.On("Error", ...) — any unexpected call will fail the test.

	// Act
	result, err := s.sut.Execute(ctx, "input")

	// Assert
	s.Require().NoError(err)
	s.Equal("output", result)
}

func (s *LoggingDecoratorTestSuite) TestExecute_Error_LogsErrorWithUseCaseName() {
	// Arrange
	ctx := context.Background()
	expectedErr := errors.New("use case failed")
	s.baseMock.On("Execute", mock.Anything, "input").Return("", expectedErr)
	s.loggerMock.On("Error", "TestUseCase.Execute failed", mock.Anything).Return()

	// Act
	result, err := s.sut.Execute(ctx, "input")

	// Assert
	s.Require().ErrorIs(err, expectedErr)
	s.Empty(result)
}
