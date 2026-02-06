package response

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/cristiano-pacheco/bricks/pkg/errs"
)

func Error(w http.ResponseWriter, err error) {
	rError := &errs.Error{}
	ok := errors.As(err, &rError)
	if !ok {
		httpStatus := http.StatusInternalServerError
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(httpStatus)
		_ = json.NewEncoder(w).Encode(Envelope{
			"error": map[string]string{
				"code":    "internal_server_error",
				"message": "Internal server error",
			},
		})
		return
	}

	if rError.Status == 0 {
		rError.Status = http.StatusInternalServerError
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(rError.Status)
	_ = json.NewEncoder(w).Encode(Envelope{
		"error": rError,
	})
}

func JSON[T any](w http.ResponseWriter, status int, data T, headers http.Header) error {
	// Set Content-Type first
	w.Header().Set("Content-Type", "application/json")

	// Copy custom headers efficiently
	for key, values := range headers {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	w.WriteHeader(status)

	// Use encoder to write directly to response writer (more efficient)
	envelope := NewEnvelope(data)
	return json.NewEncoder(w).Encode(envelope)
}

func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// JSONRaw sends a JSON response without envelope wrapper for better performance.
// Use this when you don't need the "data" wrapper.
func JSONRaw[T any](w http.ResponseWriter, status int, data T, headers http.Header) error {
	w.Header().Set("Content-Type", "application/json")

	for key, values := range headers {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(data)
}
