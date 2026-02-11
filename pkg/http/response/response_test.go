package response_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cristiano-pacheco/bricks/pkg/http/response"
	"github.com/stretchr/testify/require"
)

func TestJSON_SetsHeadersAndEnvelopes(t *testing.T) {
	// Arrange
	rr := httptest.NewRecorder()
	headers := http.Header{}
	headers.Add("X-Custom", "v1")
	data := map[string]string{"hello": "world"}

	// Act
	err := response.JSON(rr, http.StatusAccepted, data, headers)

	// Assert
	require.NoError(t, err)
	require.Equal(t, http.StatusAccepted, rr.Code)
	require.Equal(t, "application/json", rr.Header().Get("Content-Type"))
	require.Equal(t, "v1", rr.Header().Get("X-Custom"))

	var out struct {
		Data map[string]string `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &out))
	require.Equal(t, data, out.Data)
}

func TestNoContent_WritesNoContentStatus(t *testing.T) {
	// Arrange
	rr := httptest.NewRecorder()

	// Act
	response.NoContent(rr)

	// Assert
	require.Equal(t, http.StatusNoContent, rr.Code)
	require.Equal(t, 0, rr.Body.Len())
}

func TestJSONRaw_WritesRawJSONAndHeaders(t *testing.T) {
	// Arrange
	rr := httptest.NewRecorder()
	headers := http.Header{}
	headers.Add("X-Trace", "t1")
	payload := struct {
		Msg string `json:"msg"`
	}{Msg: "ok"}

	// Act
	err := response.JSONRaw(rr, http.StatusOK, payload, headers)

	// Assert
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rr.Code)
	require.Equal(t, "application/json", rr.Header().Get("Content-Type"))
	require.Equal(t, "t1", rr.Header().Get("X-Trace"))

	var out struct {
		Msg string `json:"msg"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &out))
	require.Equal(t, "ok", out.Msg)
}
