package service_test

import (
	"errors"
	"testing"

	"github.com/cristiano-pacheco/bricks/pkg/i18n/config"
	"github.com/cristiano-pacheco/bricks/pkg/i18n/service"
	"github.com/cristiano-pacheco/bricks/pkg/logger"
	"github.com/stretchr/testify/suite"
)

type TranslationServiceTestSuite struct {
	suite.Suite
	sut *service.TranslationService
}

func TestTranslationServiceSuite(t *testing.T) {
	suite.Run(t, new(TranslationServiceTestSuite))
}

func (s *TranslationServiceTestSuite) SetupTest() {
	loader := &stubLocaleLoaderService{
		locales: map[string]map[string]map[string]string{
			"en": {
				"admin": {
					"products.title": "Products",
					"welcome":        "Hello {{.Name}}",
					"only_en":        "English fallback",
				},
			},
			"pt_BR": {
				"admin": {
					"products.title": "Produtos",
					"welcome":        "Ola {{.Name}}",
				},
			},
		},
	}

	cfg := config.Config{
		Language: "pt_BR",
	}

	sut, err := service.NewTranslationService(loader, cfg, logger.MustNew(logger.DefaultConfig()))
	s.Require().NoError(err)
	s.sut = sut
}

func (s *TranslationServiceTestSuite) TestTranslateExistingKey() {
	value := s.sut.Translate("admin", "products.title")

	s.Equal("Produtos", value)
}

func (s *TranslationServiceTestSuite) TestTranslateMissingKeyFallsBackToEnglish() {
	value := s.sut.Translate("admin", "only_en")

	s.Equal("English fallback", value)
}

func (s *TranslationServiceTestSuite) TestTranslateWithPlaceholders() {
	value := s.sut.TranslateWithData("admin", "welcome", map[string]string{"Name": "Carla"})

	s.Equal("Ola Carla", value)
}

func (s *TranslationServiceTestSuite) TestGetAllForDomain() {
	values := s.sut.GetAllForDomain("admin")

	s.Equal("Produtos", values["products.title"])
	s.Equal("English fallback", values["only_en"])
	s.Equal("Ola {{.Name}}", values["welcome"])
}

type stubLocaleLoaderService struct {
	locales map[string]map[string]map[string]string
}

func (s *stubLocaleLoaderService) Load(locale string) (map[string]map[string]string, error) {
	translations, ok := s.locales[locale]
	if !ok {
		return nil, errors.New("locale not found")
	}

	return cloneTranslations(translations), nil
}

func cloneTranslations(input map[string]map[string]string) map[string]map[string]string {
	copied := make(map[string]map[string]string, len(input))
	for domain, values := range input {
		domainCopy := make(map[string]string, len(values))
		for key, value := range values {
			domainCopy[key] = value
		}
		copied[domain] = domainCopy
	}

	return copied
}
