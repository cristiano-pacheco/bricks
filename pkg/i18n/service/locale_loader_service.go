package service

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"strings"

	"github.com/cristiano-pacheco/bricks/pkg/i18n/ports"
	"github.com/cristiano-pacheco/bricks/pkg/logger"
)

const defaultLocaleCode = "en"

type LocaleLoaderService struct {
	logger   logger.Logger
	localeFS fs.FS
}

var _ ports.LocaleLoaderService = (*LocaleLoaderService)(nil)

func NewLocaleLoaderService(log logger.Logger, localeFS fs.FS) *LocaleLoaderService {
	return &LocaleLoaderService{logger: log, localeFS: localeFS}
}

func (s *LocaleLoaderService) Load(locale string) (map[string]map[string]string, error) {
	requestedLocale := strings.TrimSpace(locale)
	if requestedLocale == "" {
		requestedLocale = defaultLocaleCode
	}

	translations, err := s.loadLocaleFile(requestedLocale)
	if err == nil {
		return translations, nil
	}

	if requestedLocale == defaultLocaleCode {
		return nil, err
	}

	s.logger.Warn(
		"failed to load configured locale, using fallback locale",
		logger.String("locale", requestedLocale),
		logger.String("fallback_locale", defaultLocaleCode),
		logger.Error(err),
	)

	fallbackTranslations, fallbackErr := s.loadLocaleFile(defaultLocaleCode)
	if fallbackErr != nil {
		return nil, fmt.Errorf("load locale fallback %q: %w", defaultLocaleCode, fallbackErr)
	}

	return fallbackTranslations, nil
}

func (s *LocaleLoaderService) loadLocaleFile(locale string) (map[string]map[string]string, error) {
	content, err := fs.ReadFile(s.localeFS, locale+".json")
	if err != nil {
		return nil, fmt.Errorf("read locale %q: %w", locale, err)
	}

	translations := make(map[string]map[string]string)
	if unmarshalErr := json.Unmarshal(content, &translations); unmarshalErr != nil {
		return nil, fmt.Errorf("parse locale %q: %w", locale, unmarshalErr)
	}

	return translations, nil
}
