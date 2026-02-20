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

type DebugDecoratorTestSuite struct {
	suite.Suite
	sut        ucdecorator.UseCase[string, string]
	baseMock   *mocks.MockUseCase[string, string]
	loggerMock *mocks.MockLogger
}

func (s *DebugDecoratorTestSuite) SetupTest() {
	s.baseMock = mocks.NewMockUseCase[string, string](s.T())
	s.loggerMock = mocks.NewMockLogger(s.T())
	s.sut = ucdecorator.WithDebug(s.baseMock, s.loggerMock, "TestUseCase.Execute", "translation")
}

func TestDebugDecoratorSuite(t *testing.T) {
	suite.Run(t, new(DebugDecoratorTestSuite))
}

func (s *DebugDecoratorTestSuite) TestWithDebug_NilLogger_ReturnsBaseUnchanged() {
	// Arrange
	baseMock := mocks.NewMockUseCase[string, string](s.T())

	// Act
	result := ucdecorator.WithDebug(baseMock, nil, "TestUseCase.Execute", "translation")

	// Assert
	s.Same(baseMock, result)
}

func (s *DebugDecoratorTestSuite) TestExecute_Success_LogsStartAndSuccess() {
	// Arrange
	ctx := context.Background()
	s.baseMock.On("Execute", mock.Anything, "input").Return("output", nil)
	s.loggerMock.On("Debug", "decorator execute start", mock.Anything, mock.Anything).Once().Return()
	s.loggerMock.On("Debug", "decorator execute success", mock.Anything, mock.Anything, mock.Anything).Once().Return()

	// Act
	result, err := s.sut.Execute(ctx, "input")

	// Assert
	s.Require().NoError(err)
	s.Equal("output", result)
}

func (s *DebugDecoratorTestSuite) TestExecute_Error_LogsStartAndError() {
	// Arrange
	ctx := context.Background()
	expectedErr := errors.New("use case failed")
	s.baseMock.On("Execute", mock.Anything, "input").Return("", expectedErr)
	s.loggerMock.On("Debug", "decorator execute start", mock.Anything, mock.Anything).Once().Return()
	s.loggerMock.On("Debug", "decorator execute error", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Once().
		Return()

	// Act
	result, err := s.sut.Execute(ctx, "input")

	// Assert
	s.Require().ErrorIs(err, expectedErr)
	s.Empty(result)
}
