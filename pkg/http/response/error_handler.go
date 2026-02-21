package response

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"strings"

	"github.com/cristiano-pacheco/bricks/pkg/errs"
	"github.com/cristiano-pacheco/bricks/pkg/logger"
	"github.com/cristiano-pacheco/bricks/pkg/validator"
	lib_validator "github.com/go-playground/validator/v10"
)

var camelToSnakeRe = regexp.MustCompile("([a-z0-9])([A-Z])")

var genericError = Envelope{
	"error": map[string]string{
		"code":    "internal_server_error",
		"message": "Internal server error",
	},
}

type ErrorHandler interface {
	Error(w http.ResponseWriter, err error)
}

type ErrorHandlerImpl struct {
	validate validator.Validator
	logger   logger.Logger
}

func NewErrorHandler(validate validator.Validator, log logger.Logger) *ErrorHandlerImpl {
	return &ErrorHandlerImpl{
		validate: validate,
		logger:   log,
	}
}

func (h *ErrorHandlerImpl) logError(msg string, err error) {
	if h.logger != nil {
		h.logger.Error(msg, logger.Error(err))
	} else {
		slog.Default().Error(msg, "err", err)
	}
}

func (h *ErrorHandlerImpl) Error(w http.ResponseWriter, err error) {
	var validationErrors lib_validator.ValidationErrors
	if errors.As(err, &validationErrors) {
		details := make([]errs.Detail, 0, len(validationErrors))
		for _, e := range validationErrors {
			var msg string
			if h.validate != nil {
				msg = e.Translate(h.validate.Translator())
			} else {
				msg = fmt.Sprintf("%s: %s", camelToSnake(e.Field()), e.Tag())
			}
			details = append(details, errs.Detail{
				Field:   camelToSnake(e.Field()),
				Message: msg,
			})
		}

		validationError := errs.New(
			"INVALID_ARGUMENT",
			"request has invalid fields",
			http.StatusUnprocessableEntity,
			details,
		)

		h.writeResponse(w, validationError.Status, Envelope{"error": validationError})
		return
	}

	rError := &errs.Error{}
	ok := errors.As(err, &rError)
	if !ok {
		h.writeResponse(w, http.StatusInternalServerError, genericError)
		return
	}

	if rError.Status == 0 {
		rError.Status = http.StatusInternalServerError
	}

	h.writeResponse(w, rError.Status, Envelope{"error": rError})
}

func (h *ErrorHandlerImpl) writeResponse(w http.ResponseWriter, status int, payload Envelope) {
	body, err := json.Marshal(payload)
	if err != nil {
		h.logError("failed to marshal error response", err)
		h.writeGenericError(w)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if _, err = w.Write(body); err != nil {
		h.logError("failed to write error response", err)
	}
}

func (h *ErrorHandlerImpl) writeGenericError(w http.ResponseWriter) {
	body, err := json.Marshal(genericError)
	if err != nil {
		h.logError("failed to marshal generic error response", err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	if _, err = w.Write(body); err != nil {
		h.logError("failed to write generic error response", err)
	}
}

func camelToSnake(s string) string {
	snake := camelToSnakeRe.ReplaceAllString(s, "${1}_${2}")
	return strings.ToLower(snake)
}
