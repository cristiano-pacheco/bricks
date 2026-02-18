package ucdecorator_test

import (
	"context"
	"errors"
	"testing"

	"github.com/cristiano-pacheco/bricks/pkg/ucdecorator"
	"github.com/cristiano-pacheco/bricks/test/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// testHandlerUseCase is a named UseCase for predictable name-inference testing.
type testHandlerUseCase struct{}

func (h *testHandlerUseCase) Execute(_ context.Context, _ string) (string, error) {
	return "ok", nil
}

type FactoryTestSuite struct {
	suite.Suite
	loggerMock     *mocks.MockLogger
	metricsMock    *mocks.MockUseCaseMetrics
	translatorMock *mocks.MockErrorTranslator
}

func (s *FactoryTestSuite) SetupTest() {
	s.loggerMock = mocks.NewMockLogger(s.T())
	s.metricsMock = mocks.NewMockUseCaseMetrics(s.T())
	s.translatorMock = mocks.NewMockErrorTranslator(s.T())
}

func TestFactorySuite(t *testing.T) {
	suite.Run(t, new(FactoryTestSuite))
}

func (s *FactoryTestSuite) TestWrap_Disabled_ReturnsHandlerUnchanged() {
	// Arrange
	factory := ucdecorator.NewTestFactory(ucdecorator.Config{Enabled: false}, nil, nil, nil)
	handlerMock := mocks.NewMockUseCase[string, string](s.T())

	// Act
	result := ucdecorator.Wrap(factory, handlerMock)

	// Assert
	s.Same(handlerMock, result)
}

func (s *FactoryTestSuite) TestWrap_OnlyMetricsEnabled_Success_IncrementsSuccessAndObservesDuration() {
	// Arrange
	ctx := context.Background()
	input := "input"
	factory := ucdecorator.NewTestFactory(
		ucdecorator.Config{Enabled: true, Metrics: true},
		s.metricsMock,
		nil,
		nil,
	)
	handlerMock := mocks.NewMockUseCase[string, string](s.T())
	handlerMock.On("Execute", mock.Anything, input).Return("output", nil)
	s.metricsMock.On("ObserveDuration", mock.Anything, mock.Anything).Return()
	s.metricsMock.On("IncSuccess", mock.Anything).Return()

	sut := ucdecorator.Wrap(factory, handlerMock)

	// Act
	result, err := sut.Execute(ctx, input)

	// Assert
	s.Require().NoError(err)
	s.Equal("output", result)
}

func (s *FactoryTestSuite) TestWrap_OnlyMetricsEnabled_Error_IncrementsErrorAndObservesDuration() {
	// Arrange
	ctx := context.Background()
	input := "input"
	expectedErr := errors.New("handler error")
	factory := ucdecorator.NewTestFactory(
		ucdecorator.Config{Enabled: true, Metrics: true},
		s.metricsMock,
		nil,
		nil,
	)
	handlerMock := mocks.NewMockUseCase[string, string](s.T())
	handlerMock.On("Execute", mock.Anything, input).Return("", expectedErr)
	s.metricsMock.On("ObserveDuration", mock.Anything, mock.Anything).Return()
	s.metricsMock.On("IncError", mock.Anything).Return()

	sut := ucdecorator.Wrap(factory, handlerMock)

	// Act
	result, err := sut.Execute(ctx, input)

	// Assert
	s.Require().ErrorIs(err, expectedErr)
	s.Empty(result)
}

func (s *FactoryTestSuite) TestWrap_OnlyLoggingEnabled_Error_LogsError() {
	// Arrange
	ctx := context.Background()
	input := "input"
	expectedErr := errors.New("handler error")
	factory := ucdecorator.NewTestFactory(
		ucdecorator.Config{Enabled: true, Logging: true},
		nil,
		s.loggerMock,
		nil,
	)
	handlerMock := mocks.NewMockUseCase[string, string](s.T())
	handlerMock.On("Execute", mock.Anything, input).Return("", expectedErr)
	s.loggerMock.On("Error", mock.Anything, mock.Anything).Return()

	sut := ucdecorator.Wrap(factory, handlerMock)

	// Act
	result, err := sut.Execute(ctx, input)

	// Assert
	s.Require().ErrorIs(err, expectedErr)
	s.Empty(result)
}

func (s *FactoryTestSuite) TestWrap_OnlyLoggingEnabled_Success_DoesNotLogError() {
	// Arrange
	ctx := context.Background()
	input := "input"
	factory := ucdecorator.NewTestFactory(
		ucdecorator.Config{Enabled: true, Logging: true},
		nil,
		s.loggerMock,
		nil,
	)
	handlerMock := mocks.NewMockUseCase[string, string](s.T())
	handlerMock.On("Execute", mock.Anything, input).Return("output", nil)
	// No loggerMock.On("Error", ...) setup — any unexpected Error call will fail the test.

	sut := ucdecorator.Wrap(factory, handlerMock)

	// Act
	result, err := sut.Execute(ctx, input)

	// Assert
	s.Require().NoError(err)
	s.Equal("output", result)
}

func (s *FactoryTestSuite) TestWrap_OnlyTranslationEnabled_Error_TranslatesError() {
	// Arrange
	ctx := context.Background()
	input := "input"
	originalErr := errors.New("original error")
	translatedErr := errors.New("translated error")
	factory := ucdecorator.NewTestFactory(
		ucdecorator.Config{Enabled: true, Translation: true},
		nil,
		nil,
		s.translatorMock,
	)
	handlerMock := mocks.NewMockUseCase[string, string](s.T())
	handlerMock.On("Execute", mock.Anything, input).Return("", originalErr)
	s.translatorMock.On("TranslateError", originalErr).Return(translatedErr)

	sut := ucdecorator.Wrap(factory, handlerMock)

	// Act
	result, err := sut.Execute(ctx, input)

	// Assert
	s.Require().ErrorIs(err, translatedErr)
	s.Empty(result)
}

func (s *FactoryTestSuite) TestWrap_OnlyTranslationEnabled_Success_DoesNotTranslate() {
	// Arrange
	ctx := context.Background()
	input := "input"
	factory := ucdecorator.NewTestFactory(
		ucdecorator.Config{Enabled: true, Translation: true},
		nil,
		nil,
		s.translatorMock,
	)
	handlerMock := mocks.NewMockUseCase[string, string](s.T())
	handlerMock.On("Execute", mock.Anything, input).Return("output", nil)
	// No translatorMock.On("TranslateError", ...) setup — any unexpected call will fail.

	sut := ucdecorator.Wrap(factory, handlerMock)

	// Act
	result, err := sut.Execute(ctx, input)

	// Assert
	s.Require().NoError(err)
	s.Equal("output", result)
}

func (s *FactoryTestSuite) TestInferUseCaseName_PointerToNamedStruct_ReturnsNameDotExecute() {
	// Arrange
	f := ucdecorator.NewTestFactory(ucdecorator.Config{}, nil, nil, nil)
	handler := &testHandlerUseCase{}

	// Act
	name := f.InferUseCaseName(handler)

	// Assert
	s.Equal("testHandlerUseCase.Execute", name)
}

func (s *FactoryTestSuite) TestInferUseCaseName_NonPointerNamedStruct_ReturnsNameDotExecute() {
	// Arrange
	f := ucdecorator.NewTestFactory(ucdecorator.Config{}, nil, nil, nil)
	handler := testHandlerUseCase{}

	// Act
	name := f.InferUseCaseName(handler)

	// Assert
	s.Equal("testHandlerUseCase.Execute", name)
}

func (s *FactoryTestSuite) TestInferUseCaseName_AnonymousStruct_ReturnsFallback() {
	// Arrange
	f := ucdecorator.NewTestFactory(ucdecorator.Config{}, nil, nil, nil)
	handler := struct{}{}

	// Act
	name := f.InferUseCaseName(handler)

	// Assert
	s.Equal("UseCase.Execute", name)
}

func (s *FactoryTestSuite) TestInferUseCaseName_Nil_ReturnsFallback() {
	// Arrange
	f := ucdecorator.NewTestFactory(ucdecorator.Config{}, nil, nil, nil)

	// Act
	name := f.InferUseCaseName(nil)

	// Assert
	s.Equal("UseCase.Execute", name)
}

func (s *FactoryTestSuite) TestInferMetricName_WithUseCaseSuffix_RemovesSuffixAndSnakeCases() {
	// Arrange
	f := ucdecorator.NewTestFactory(ucdecorator.Config{}, nil, nil, nil)

	// Act
	name := f.InferMetricName("testHandlerUseCase.Execute")

	// Assert
	s.Equal("test_handler", name)
}

func (s *FactoryTestSuite) TestInferMetricName_WithoutUseCaseSuffix_SnakeCasesTypeName() {
	// Arrange
	f := ucdecorator.NewTestFactory(ucdecorator.Config{}, nil, nil, nil)

	// Act
	name := f.InferMetricName("testSimpleHandler.Execute")

	// Assert
	s.Equal("test_simple_handler", name)
}

func (s *FactoryTestSuite) TestInferMetricName_OnlyUseCaseSuffix_ReturnsFallback() {
	// Arrange
	f := ucdecorator.NewTestFactory(ucdecorator.Config{}, nil, nil, nil)

	// Act
	name := f.InferMetricName("UseCase.Execute")

	// Assert
	s.Equal("use_case", name)
}

func TestToSnakeCase(t *testing.T) {
	f := ucdecorator.NewTestFactory(ucdecorator.Config{}, nil, nil, nil)

	t.Run("camel case converts to snake case", func(t *testing.T) {
		require.Equal(t, "create_user", f.ToSnakeCase("CreateUser"))
	})

	t.Run("consecutive uppercase acronym followed by word", func(t *testing.T) {
		require.Equal(t, "xml_parser", f.ToSnakeCase("XMLParser"))
	})

	t.Run("https acronym followed by word", func(t *testing.T) {
		require.Equal(t, "https_request", f.ToSnakeCase("HTTPSRequest"))
	})

	t.Run("all lowercase returns unchanged", func(t *testing.T) {
		require.Equal(t, "simple", f.ToSnakeCase("simple"))
	})

	t.Run("empty string returns empty string", func(t *testing.T) {
		require.Equal(t, "", f.ToSnakeCase(""))
	})

	t.Run("mixed case with id suffix", func(t *testing.T) {
		require.Equal(t, "user_id", f.ToSnakeCase("UserID"))
	})
}
