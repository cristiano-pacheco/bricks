package service

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"strings"

	"github.com/cristiano-pacheco/bricks/pkg/i18n/locale"
	"github.com/cristiano-pacheco/bricks/pkg/i18n/ports"
	"github.com/cristiano-pacheco/bricks/pkg/logger"
)

const defaultLocaleCode = "en"

type LocaleLoaderService struct {
	logger      logger.Logger
	FileSystems []locale.FileSystem
}

var _ ports.LocaleLoaderService = (*LocaleLoaderService)(nil)

func NewLocaleLoaderService(log logger.Logger, fss []locale.FileSystem) *LocaleLoaderService {
	return &LocaleLoaderService{logger: log, FileSystems: fss}
}

func (s *LocaleLoaderService) Load(locale string) (map[string]map[string]string, error) {
	requestedLocale := strings.TrimSpace(locale)
	if requestedLocale == "" {
		requestedLocale = defaultLocaleCode
	}

	merged, found := s.loadAndMergeAll(requestedLocale)
	if found {
		return merged, nil
	}

	if requestedLocale == defaultLocaleCode {
		return nil, fmt.Errorf("no locale files found for %q", requestedLocale)
	}

	s.logger.Warn(
		"no locale files found for requested locale, falling back to default",
		logger.String("locale", requestedLocale),
		logger.String("fallback_locale", defaultLocaleCode),
	)

	fallback, fallbackFound := s.loadAndMergeAll(defaultLocaleCode)
	if !fallbackFound {
		return nil, fmt.Errorf("no locale files found for fallback locale %q", defaultLocaleCode)
	}

	return fallback, nil
}

func (s *LocaleLoaderService) loadAndMergeAll(locale string) (map[string]map[string]string, bool) {
	merged := make(map[string]map[string]string)
	found := false
	for _, fileSystem := range s.FileSystems {
		translations, err := s.loadLocaleFile(fileSystem, locale)
		if err != nil {
			s.logger.Debug(
				"locale file not found in filesystem, skipping",
				logger.String("locale", locale),
				logger.Error(err),
			)
			continue
		}
		mergeInto(merged, translations)
		found = true
	}
	return merged, found
}

func mergeInto(dst, src map[string]map[string]string) {
	for domain, keys := range src {
		if dst[domain] == nil {
			dst[domain] = make(map[string]string)
		}
		for k, v := range keys {
			dst[domain][k] = v
		}
	}
}

func (s *LocaleLoaderService) loadLocaleFile(
	fileSystem locale.FileSystem,
	locale string,
) (map[string]map[string]string, error) {
	content, err := fs.ReadFile(fileSystem.FS, locale+".json")
	if err != nil {
		return nil, fmt.Errorf("read locale %q: %w", locale, err)
	}

	translations := make(map[string]map[string]string)
	if unmarshalErr := json.Unmarshal(content, &translations); unmarshalErr != nil {
		return nil, fmt.Errorf("parse locale %q: %w", locale, unmarshalErr)
	}

	return translations, nil
}
