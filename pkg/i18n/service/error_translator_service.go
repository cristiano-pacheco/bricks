package service

import (
	"errors"

	brickserrs "github.com/cristiano-pacheco/bricks/pkg/errs"
	"github.com/cristiano-pacheco/bricks/pkg/i18n/ports"
)

type ErrorTranslatorService struct {
	translationService ports.TranslationService
}

var _ ports.ErrorTranslatorService = (*ErrorTranslatorService)(nil)

func NewErrorTranslatorService(translationService ports.TranslationService) *ErrorTranslatorService {
	return &ErrorTranslatorService{translationService: translationService}
}

func (s *ErrorTranslatorService) TranslateError(err error) error {
	var bricksErr *brickserrs.Error
	if !errors.As(err, &bricksErr) {
		return err
	}

	translatedMessage := s.translationService.Translate("errors", bricksErr.Code)
	if translatedMessage == "" || translatedMessage == bricksErr.Code {
		return err
	}

	return brickserrs.New(bricksErr.Code, translatedMessage, bricksErr.Status, bricksErr.Details)
}
