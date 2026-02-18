package service

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/cristiano-pacheco/bricks/pkg/i18n/config"
	"github.com/cristiano-pacheco/bricks/pkg/i18n/ports"
	"github.com/cristiano-pacheco/bricks/pkg/logger"
)

type TranslationService struct {
	logger               logger.Logger
	locale               string
	translations         map[string]map[string]string
	fallbackTranslations map[string]map[string]string
}

var _ ports.TranslationService = (*TranslationService)(nil)

func NewTranslationService(
	localeLoaderService ports.LocaleLoaderService,
	cfg config.Config,
	log logger.Logger,
) (*TranslationService, error) {
	fallbackTranslations, err := localeLoaderService.Load(defaultLocaleCode)
	if err != nil {
		return nil, fmt.Errorf("load fallback locale %q: %w", defaultLocaleCode, err)
	}

	configuredLocale := strings.TrimSpace(cfg.Language)
	if configuredLocale == "" {
		configuredLocale = defaultLocaleCode
	}

	translations, err := localeLoaderService.Load(configuredLocale)
	if err != nil {
		return nil, fmt.Errorf("load configured locale %q: %w", configuredLocale, err)
	}

	return &TranslationService{
		logger:               log,
		locale:               configuredLocale,
		translations:         translations,
		fallbackTranslations: fallbackTranslations,
	}, nil
}

func (s *TranslationService) Translate(domain, key string) string {
	resolvedDomain := strings.TrimSpace(domain)
	resolvedKey := strings.TrimSpace(key)
	if resolvedDomain == "" || resolvedKey == "" {
		return resolvedKey
	}

	if value, ok := s.findTranslationValue(s.translations, resolvedDomain, resolvedKey); ok {
		return value
	}

	if fallbackValue, ok := s.findTranslationValue(s.fallbackTranslations, resolvedDomain, resolvedKey); ok {
		s.logger.Warn(
			"translation key missing in configured locale, using fallback",
			logger.String("locale", s.locale),
			logger.String("fallback_locale", defaultLocaleCode),
			logger.String("domain", resolvedDomain),
			logger.String("key", resolvedKey),
		)
		return fallbackValue
	}

	s.logger.Warn(
		"translation key not found",
		logger.String("locale", s.locale),
		logger.String("domain", resolvedDomain),
		logger.String("key", resolvedKey),
	)

	return resolvedKey
}

func (s *TranslationService) TranslateWithData(domain, key string, data map[string]string) string {
	value := s.Translate(domain, key)
	if len(data) == 0 {
		return value
	}

	tmpl, parseErr := template.New("translation").Option("missingkey=default").Parse(value)
	if parseErr != nil {
		s.logger.Warn(
			"failed to parse translation template",
			logger.String("domain", domain),
			logger.String("key", key),
			logger.Error(parseErr),
		)
		return value
	}

	buffer := bytes.NewBuffer(nil)
	if executeErr := tmpl.Execute(buffer, data); executeErr != nil {
		s.logger.Warn(
			"failed to execute translation template",
			logger.String("domain", domain),
			logger.String("key", key),
			logger.Error(executeErr),
		)
		return value
	}

	return buffer.String()
}

func (s *TranslationService) GetAllForDomain(domain string) map[string]string {
	resolvedDomain := strings.TrimSpace(domain)
	if resolvedDomain == "" {
		return map[string]string{}
	}

	result := make(map[string]string)
	for key, value := range s.fallbackTranslations[resolvedDomain] {
		result[key] = value
	}
	for key, value := range s.translations[resolvedDomain] {
		result[key] = value
	}

	return result
}

func (s *TranslationService) findTranslationValue(all map[string]map[string]string, domain, key string) (string, bool) {
	domainMap, ok := all[domain]
	if !ok {
		return "", false
	}

	value, ok := domainMap[key]
	if !ok {
		return "", false
	}

	return value, true
}
