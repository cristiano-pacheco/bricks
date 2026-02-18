package service_test

import (
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/cristiano-pacheco/bricks/pkg/i18n/service"
	"github.com/cristiano-pacheco/bricks/pkg/logger"
	"github.com/stretchr/testify/suite"
)

type LocaleLoaderServiceTestSuite struct {
	suite.Suite
	sut  *service.LocaleLoaderService
	fsys fstest.MapFS
}

func TestLocaleLoaderServiceSuite(t *testing.T) {
	suite.Run(t, new(LocaleLoaderServiceTestSuite))
}

func (s *LocaleLoaderServiceTestSuite) SetupTest() {
	s.fsys = fstest.MapFS{
		"en.json": &fstest.MapFile{
			Data: []byte(`{
				"admin": {
					"nav.products": "Products"
				}
			}`),
		},
		"pt_BR.json": &fstest.MapFile{
			Data: []byte(`{
				"errors": {
					"EXPORT_01": "Número de telefone é obrigatório"
				}
			}`),
		},
	}

	s.sut = service.NewLocaleLoaderService(logger.MustNew(logger.DefaultConfig()), s.fsys)
}

func (s *LocaleLoaderServiceTestSuite) TestLoadValidLocale() {
	translations, err := s.sut.Load("en")

	s.Require().NoError(err)
	s.Require().Contains(translations, "admin")
	s.Require().Contains(translations["admin"], "nav.products")
	s.Equal("Products", translations["admin"]["nav.products"])
}

func (s *LocaleLoaderServiceTestSuite) TestLoadInvalidLocaleFallsBackToEnglish() {
	translations, err := s.sut.Load("invalid-locale")

	s.Require().NoError(err)
	s.Require().Contains(translations, "admin")
	s.Equal("Products", translations["admin"]["nav.products"])
}

func (s *LocaleLoaderServiceTestSuite) TestParseJSONStructure() {
	translations, err := s.sut.Load("pt_BR")

	s.Require().NoError(err)
	s.Require().Contains(translations, "errors")
	s.Require().Contains(translations["errors"], "EXPORT_01")
	s.Equal("Número de telefone é obrigatório", translations["errors"]["EXPORT_01"])
}

var _ fs.FS = fstest.MapFS{}
