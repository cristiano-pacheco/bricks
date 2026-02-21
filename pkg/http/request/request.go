package request

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/cristiano-pacheco/bricks/pkg/errs"
)

const (
	// DefaultMaxBodySize is 1MB - can be overridden with ReadJSONWithMaxSize
	DefaultMaxBodySize = 1_048_576
)

// ReadJSON reads and decodes JSON from request body with default max size of 1MB.
// Validates Content-Type, limits body size, and provides detailed error messages.
func ReadJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	return ReadJSONWithMaxSize(w, r, dst, DefaultMaxBodySize)
}

// ReadJSONWithMaxSize reads and decodes JSON with a custom maximum body size.
func ReadJSONWithMaxSize(w http.ResponseWriter, r *http.Request, dst any, maxBytes int64) error {
	// Security: Validate Content-Type to prevent CSRF attacks
	if err := validateJSONContentType(r.Header.Get("Content-Type")); err != nil {
		return err
	}
	// Security: Limit request body size to prevent DoS attacks
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)

	// Performance: Reuse decoder instead of creating multiple times
	dec := json.NewDecoder(r.Body)
	// Security: Reject unknown fields to prevent injection of unexpected data
	dec.DisallowUnknownFields()

	return decodeJSONPayload(dec, dst)
}

func validateJSONContentType(contentType string) error {
	if contentType == "" {
		return nil
	}
	// Remove charset and other parameters for comparison
	if idx := strings.IndexByte(contentType, ';'); idx != -1 {
		contentType = contentType[:idx]
	}
	contentType = strings.TrimSpace(contentType)
	if contentType != "application/json" {
		return errs.New(
			"UNSUPPORTED_MEDIA_TYPE",
			"Content-Type header is not application/json",
			http.StatusUnsupportedMediaType,
			nil,
		)
	}
	return nil
}

func decodeJSONPayload(dec *json.Decoder, dst any) error {
	err := dec.Decode(dst)
	if err != nil {
		return parseJSONDecodeError(err)
	}

	// Security: Ensure only one JSON value to prevent processing of trailing data
	// Performance: Use Token() instead of Decode() to avoid allocating a struct
	if dec.More() {
		return errs.New("BAD_REQUEST", "request body must contain only a single JSON value", http.StatusBadRequest, nil)
	}

	return nil
}

func parseJSONDecodeError(err error) error {
	var syntaxError *json.SyntaxError
	var unmarshalTypeError *json.UnmarshalTypeError
	var invalidUnmarshalError *json.InvalidUnmarshalError
	var maxBytesError *http.MaxBytesError

	switch {
	case errors.As(err, &syntaxError):
		// Security: Don't expose exact position to avoid information leakage
		return errs.New("BAD_REQUEST", "request body contains malformed JSON", http.StatusBadRequest, nil)

	case errors.Is(err, io.ErrUnexpectedEOF):
		return errs.New("BAD_REQUEST", "request body contains malformed JSON", http.StatusBadRequest, nil)

	case errors.As(err, &unmarshalTypeError):
		if unmarshalTypeError.Field != "" {
			return errs.New(
				"BAD_REQUEST",
				fmt.Sprintf("request body contains invalid value for field %q", unmarshalTypeError.Field),
				http.StatusBadRequest,
				nil,
			)
		}
		return errs.New("BAD_REQUEST", "request body contains invalid JSON type", http.StatusBadRequest, nil)

	case errors.Is(err, io.EOF):
		return errs.New("BAD_REQUEST", "request body must not be empty", http.StatusBadRequest, nil)

	// Performance: Use strings.Cut (Go 1.18+) which is faster than HasPrefix + TrimPrefix
	case strings.HasPrefix(err.Error(), "json: unknown field "):
		fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
		return errs.New(
			"BAD_REQUEST",
			fmt.Sprintf("request body contains unknown field %s", fieldName),
			http.StatusBadRequest,
			nil,
		)

	case errors.As(err, &maxBytesError):
		return errs.New(
			"REQUEST_ENTITY_TOO_LARGE",
			fmt.Sprintf("request body must not exceed %d bytes", maxBytesError.Limit),
			http.StatusRequestEntityTooLarge,
			nil,
		)

	case errors.As(err, &invalidUnmarshalError):
		// This indicates a programming error, not a client error
		panic(err)

	default:
		// Security: Don't expose internal error details
		return errs.New("BAD_REQUEST", "error parsing request body", http.StatusBadRequest, nil)
	}
}
