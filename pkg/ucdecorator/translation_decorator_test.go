package ucdecorator

import (
	"context"
	"errors"
	"testing"

	"github.com/cristiano-pacheco/bricks/test/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type TranslationDecoratorTestSuite struct {
	suite.Suite
	sut            UseCase[string, string]
	baseMock       *mocks.MockUseCase[string, string]
	translatorMock *mocks.MockErrorTranslator
}

func (s *TranslationDecoratorTestSuite) SetupTest() {
	s.baseMock = mocks.NewMockUseCase[string, string](s.T())
	s.translatorMock = mocks.NewMockErrorTranslator(s.T())
	s.sut = withTranslation(s.baseMock, s.translatorMock)
}

func TestTranslationDecoratorSuite(t *testing.T) {
	suite.Run(t, new(TranslationDecoratorTestSuite))
}

func (s *TranslationDecoratorTestSuite) TestWithTranslation_NilTranslator_ReturnsBaseUnchanged() {
	// Arrange
	baseMock := mocks.NewMockUseCase[string, string](s.T())

	// Act
	result := withTranslation(baseMock, nil)

	// Assert
	s.Same(baseMock, result)
}

func (s *TranslationDecoratorTestSuite) TestExecute_Success_DoesNotCallTranslator() {
	// Arrange
	ctx := context.Background()
	s.baseMock.On("Execute", mock.Anything, "input").Return("output", nil)
	// No translatorMock.On("TranslateError", ...) — any unexpected call will fail the test.

	// Act
	result, err := s.sut.Execute(ctx, "input")

	// Assert
	s.Require().NoError(err)
	s.Equal("output", result)
}

func (s *TranslationDecoratorTestSuite) TestExecute_Error_TranslatesError() {
	// Arrange
	ctx := context.Background()
	originalErr := errors.New("original error")
	translatedErr := errors.New("translated error")
	s.baseMock.On("Execute", mock.Anything, "input").Return("", originalErr)
	s.translatorMock.On("TranslateError", originalErr).Return(translatedErr)

	// Act
	result, err := s.sut.Execute(ctx, "input")

	// Assert
	s.Require().ErrorIs(err, translatedErr)
	s.Empty(result)
}
