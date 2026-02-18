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

const (
	chainTestMetricName  = "test_metric"
	chainTestUseCaseName = "TestUseCase.Execute"
	chainTestInput       = "input"
	chainTestOutput      = "output"
)

type ChainTestSuite struct {
	suite.Suite
	sut            ucdecorator.UseCase[string, string]
	handlerMock    *mocks.MockUseCase[string, string]
	loggerMock     *mocks.MockLogger
	metricsMock    *mocks.MockUseCaseMetrics
	translatorMock *mocks.MockErrorTranslator
}

func (s *ChainTestSuite) SetupTest() {
	s.handlerMock = mocks.NewMockUseCase[string, string](s.T())
	s.loggerMock = mocks.NewMockLogger(s.T())
	s.metricsMock = mocks.NewMockUseCaseMetrics(s.T())
	s.translatorMock = mocks.NewMockErrorTranslator(s.T())

	s.sut = ucdecorator.Chain(
		s.handlerMock,
		s.loggerMock,
		s.metricsMock,
		s.translatorMock,
		chainTestMetricName,
		chainTestUseCaseName,
	)
}

func TestChainSuite(t *testing.T) {
	suite.Run(t, new(ChainTestSuite))
}

func (s *ChainTestSuite) TestExecute_Success_ObservesDurationAndIncrementsSuccess() {
	// Arrange
	ctx := context.Background()
	s.handlerMock.On("Execute", mock.Anything, chainTestInput).Return(chainTestOutput, nil)
	s.metricsMock.On("ObserveDuration", chainTestMetricName, mock.Anything).Return()
	s.metricsMock.On("IncSuccess", chainTestMetricName).Return()

	// Act
	result, err := s.sut.Execute(ctx, chainTestInput)

	// Assert
	s.Require().NoError(err)
	s.Equal(chainTestOutput, result)
}

func (s *ChainTestSuite) TestExecute_HandlerError_IncrementsErrorTranslatesAndLogsError() {
	// Arrange
	ctx := context.Background()
	originalErr := errors.New("original error")
	translatedErr := errors.New("translated error")

	s.handlerMock.On("Execute", mock.Anything, chainTestInput).Return("", originalErr)
	s.translatorMock.On("TranslateError", originalErr).Return(translatedErr)
	s.metricsMock.On("ObserveDuration", chainTestMetricName, mock.Anything).Return()
	s.metricsMock.On("IncError", chainTestMetricName).Return()
	s.loggerMock.On("Error", chainTestUseCaseName+" failed", mock.Anything).Return()

	// Act
	result, err := s.sut.Execute(ctx, chainTestInput)

	// Assert
	s.Require().ErrorIs(err, translatedErr)
	s.Empty(result)
}

func (s *ChainTestSuite) TestExecute_NilTranslator_ReturnsOriginalError() {
	// Arrange
	ctx := context.Background()
	originalErr := errors.New("original error")

	handlerMock := mocks.NewMockUseCase[string, string](s.T())
	loggerMock := mocks.NewMockLogger(s.T())
	metricsMock := mocks.NewMockUseCaseMetrics(s.T())

	sut := ucdecorator.Chain(
		handlerMock,
		loggerMock,
		metricsMock,
		nil,
		chainTestMetricName,
		chainTestUseCaseName,
	)

	handlerMock.On("Execute", mock.Anything, chainTestInput).Return("", originalErr)
	metricsMock.On("ObserveDuration", chainTestMetricName, mock.Anything).Return()
	metricsMock.On("IncError", chainTestMetricName).Return()
	loggerMock.On("Error", chainTestUseCaseName+" failed", mock.Anything).Return()

	// Act
	result, err := sut.Execute(ctx, chainTestInput)

	// Assert
	s.Require().ErrorIs(err, originalErr)
	s.Empty(result)
}
