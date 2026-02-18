package service_test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/cristiano-pacheco/bricks/pkg/errs"
	"github.com/cristiano-pacheco/bricks/pkg/i18n/service"
	"github.com/stretchr/testify/suite"
)

type ErrorTranslatorServiceTestSuite struct {
	suite.Suite
}

func TestErrorTranslatorServiceSuite(t *testing.T) {
	suite.Run(t, new(ErrorTranslatorServiceTestSuite))
}

func (s *ErrorTranslatorServiceTestSuite) TestTranslateBricksError() {
	translator := service.NewErrorTranslatorService(&stubTranslationService{
		translations: map[string]map[string]string{
			"errors": {
				"CATALOG_01": "Tipo de conteudo de imagem nao suportado",
			},
		},
	})

	originalErr := errs.New("CATALOG_01", "unsupported image content type", http.StatusBadRequest, nil)

	translatedErr := translator.TranslateError(originalErr)

	var translatedBricksErr *errs.Error
	s.Require().True(errors.As(translatedErr, &translatedBricksErr))
	s.Equal("CATALOG_01", translatedBricksErr.Code)
	s.Equal(http.StatusBadRequest, translatedBricksErr.Status)
	s.Equal("Tipo de conteudo de imagem nao suportado", translatedBricksErr.Message)
}

func (s *ErrorTranslatorServiceTestSuite) TestTranslateNonBricksError() {
	translator := service.NewErrorTranslatorService(
		&stubTranslationService{translations: map[string]map[string]string{}},
	)
	originalErr := errors.New("plain error")

	translatedErr := translator.TranslateError(originalErr)

	s.Same(originalErr, translatedErr)
}

func (s *ErrorTranslatorServiceTestSuite) TestTranslateMissingErrorCode() {
	translator := service.NewErrorTranslatorService(
		&stubTranslationService{translations: map[string]map[string]string{}},
	)
	originalErr := errs.New("UNKNOWN_CODE", "original message", http.StatusBadRequest, nil)

	translatedErr := translator.TranslateError(originalErr)

	s.Same(originalErr, translatedErr)
}

type stubTranslationService struct {
	translations map[string]map[string]string
}

func (s *stubTranslationService) Translate(domain, key string) string {
	if domainMap, ok := s.translations[domain]; ok {
		if value, exists := domainMap[key]; exists {
			return value
		}
	}

	return key
}

func (s *stubTranslationService) TranslateWithData(domain, key string, _ map[string]string) string {
	return s.Translate(domain, key)
}

func (s *stubTranslationService) GetAllForDomain(domain string) map[string]string {
	domainMap := s.translations[domain]
	output := make(map[string]string, len(domainMap))
	for key, value := range domainMap {
		output[key] = value
	}

	return output
}
