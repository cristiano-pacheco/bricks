package response

import (
	"encoding/json"
	"errors"
	"maps"
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
	envelope := NewEnvelope(data)
	js, err := json.MarshalIndent(envelope, "", "\t")
	if err != nil {
		return err
	}

	js = append(js, '\n')

	maps.Copy(w.Header(), headers)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(js)

	return nil
}

func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}
