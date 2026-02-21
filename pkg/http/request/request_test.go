package request_test

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cristiano-pacheco/bricks/pkg/errs"
	"github.com/cristiano-pacheco/bricks/pkg/http/request"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadJSON(t *testing.T) {
	t.Run("Valid JSON decodes into struct", func(t *testing.T) {
		// Arrange
		type Payload struct {
			Name string `json:"name"`
		}

		var dst Payload

		body := bytes.NewBufferString(`{"name":"alice"}`)
		r := httptest.NewRequest(http.MethodPost, "/", body)
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		// Act
		err := request.ReadJSON(w, r, &dst)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, "alice", dst.Name)
		_ = w
	})

	t.Run("Invalid Content-Type returns error", func(t *testing.T) {
		// Arrange
		type Payload struct {
			Name string `json:"name"`
		}

		var dst Payload
		body := bytes.NewBufferString(`{"name":"bob"}`)
		r := httptest.NewRequest(http.MethodPost, "/", body)
		r.Header.Set("Content-Type", "text/plain")
		w := httptest.NewRecorder()

		// Act
		err := request.ReadJSON(w, r, &dst)

		// Assert
		require.Error(t, err)
		var reqErr *errs.Error
		require.True(t, errors.As(err, &reqErr))
		assert.Equal(t, http.StatusUnsupportedMediaType, reqErr.Status)
		assert.Equal(t, "UNSUPPORTED_MEDIA_TYPE", reqErr.Code)
		assert.Equal(t, "Content-Type header is not application/json", reqErr.Message)
		_ = w
	})

	t.Run("Malformed JSON returns descriptive error", func(t *testing.T) {
		// Arrange
		type Payload struct {
			Name string `json:"name"`
		}

		var dst Payload
		body := bytes.NewBufferString(`{"name":}`)
		r := httptest.NewRequest(http.MethodPost, "/", body)
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		// Act
		err := request.ReadJSON(w, r, &dst)

		// Assert
		require.Error(t, err)
		var reqErr *errs.Error
		require.True(t, errors.As(err, &reqErr))
		assert.Equal(t, http.StatusBadRequest, reqErr.Status)
		assert.Equal(t, "BAD_REQUEST", reqErr.Code)
		assert.Contains(t, reqErr.Message, "malformed JSON")
		_ = w
	})

	t.Run("Empty body returns not empty error", func(t *testing.T) {
		// Arrange
		type Payload struct {
			Name string `json:"name"`
		}

		var dst Payload
		body := bytes.NewBufferString("")
		r := httptest.NewRequest(http.MethodPost, "/", body)
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		// Act
		err := request.ReadJSON(w, r, &dst)

		// Assert
		require.Error(t, err)
		var reqErr *errs.Error
		require.True(t, errors.As(err, &reqErr))
		assert.Equal(t, http.StatusBadRequest, reqErr.Status)
		assert.Equal(t, "BAD_REQUEST", reqErr.Code)
		assert.Equal(t, "request body must not be empty", reqErr.Message)
		_ = w
	})

	t.Run("Unknown field returns unknown field error", func(t *testing.T) {
		// Arrange
		type Payload struct {
			Name string `json:"name"`
		}

		var dst Payload
		body := bytes.NewBufferString(`{"unknown":1}`)
		r := httptest.NewRequest(http.MethodPost, "/", body)
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		// Act
		err := request.ReadJSON(w, r, &dst)

		// Assert
		require.Error(t, err)
		var reqErr *errs.Error
		require.True(t, errors.As(err, &reqErr))
		assert.Equal(t, http.StatusBadRequest, reqErr.Status)
		assert.Equal(t, "BAD_REQUEST", reqErr.Code)
		assert.Contains(t, reqErr.Message, "unknown field")
		_ = w
	})

	t.Run("Request body exceeds max bytes returns size error", func(t *testing.T) {
		// Arrange
		type Payload struct {
			Name string `json:"name"`
		}

		var dst Payload
		// create a valid JSON with a large string value so decoder reads past limit
		longVal := bytes.Repeat([]byte("a"), 256)
		jsonBuf := bytes.NewBufferString("{\"name\":\"")
		jsonBuf.Write(longVal)
		jsonBuf.WriteString("\"}")

		r := httptest.NewRequest(http.MethodPost, "/", jsonBuf)
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		// Act
		err := request.ReadJSONWithMaxSize(w, r, &dst, 10)

		// Assert
		require.Error(t, err)
		var reqErr *errs.Error
		require.True(t, errors.As(err, &reqErr))
		assert.Equal(t, http.StatusRequestEntityTooLarge, reqErr.Status)
		assert.Equal(t, "REQUEST_ENTITY_TOO_LARGE", reqErr.Code)
		assert.Contains(t, reqErr.Message, "must not exceed")
		_ = w
	})

	t.Run("Multiple JSON values returns single value error", func(t *testing.T) {
		// Arrange
		type Payload struct {
			Name string `json:"name"`
		}

		var dst Payload
		body := bytes.NewBufferString(`{} {}`)
		r := httptest.NewRequest(http.MethodPost, "/", body)
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		// Act
		err := request.ReadJSON(w, r, &dst)

		// Assert
		require.Error(t, err)
		var reqErr *errs.Error
		require.True(t, errors.As(err, &reqErr))
		assert.Equal(t, http.StatusBadRequest, reqErr.Status)
		assert.Equal(t, "BAD_REQUEST", reqErr.Code)
		assert.Equal(t, "request body must contain only a single JSON value", reqErr.Message)
		_ = w
	})
}
