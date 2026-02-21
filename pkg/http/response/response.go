package response

import (
	"encoding/json"
	"net/http"
	"strconv"
)

func JSON[T any](w http.ResponseWriter, status int, data T, headers http.Header) error {
	body, err := json.Marshal(NewEnvelope(data))
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	for key, values := range headers {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	w.WriteHeader(status)
	_, err = w.Write(body)
	return err
}

func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

func JSONRaw[T any](w http.ResponseWriter, status int, data T, headers http.Header) error {
	body, err := json.Marshal(data)
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	for key, values := range headers {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	w.WriteHeader(status)
	_, err = w.Write(body)
	return err
}
