package response

import (
	"encoding/json"
	"net/http"
)

func JSON[T any](w http.ResponseWriter, status int, data T, headers http.Header) error {
	w.Header().Set("Content-Type", "application/json")

	for key, values := range headers {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	w.WriteHeader(status)

	envelope := NewEnvelope(data)
	return json.NewEncoder(w).Encode(envelope)
}

func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

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
