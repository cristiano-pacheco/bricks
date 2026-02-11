package response_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cristiano-pacheco/bricks/pkg/errs"
	"github.com/cristiano-pacheco/bricks/pkg/http/response"
	"github.com/cristiano-pacheco/bricks/pkg/logger"
	"github.com/cristiano-pacheco/bricks/pkg/validator"
	"github.com/stretchr/testify/suite"
)

type ErrorHandlerTestSuite struct {
	suite.Suite
	sut response.ErrorHandler
	v   validator.Validator
	log logger.Logger
}

func (s *ErrorHandlerTestSuite) SetupTest() {
	var err error
	s.v, err = validator.New()
	s.Require().NoError(err)
	s.log = logger.MustNewWithOptions(logger.WithLevel("fatal"))
	s.sut = response.NewErrorHandler(s.v, s.log)
}

func TestErrorHandlerSuite(t *testing.T) {
	suite.Run(t, new(ErrorHandlerTestSuite))
}

func (s *ErrorHandlerTestSuite) TestError_NonTypedError_ReturnsInternalServerError() {
	// Arrange
	rr := httptest.NewRecorder()
	err := errors.New("boom")

	// Act
	s.sut.Error(rr, err)

	// Assert
	s.Equal(http.StatusInternalServerError, rr.Code)
	s.Equal("application/json", rr.Header().Get("Content-Type"))

	var body map[string]map[string]string
	s.Require().NoError(json.Unmarshal(rr.Body.Bytes(), &body))
	s.Equal("internal_server_error", body["error"]["code"])
	s.Equal("Internal server error", body["error"]["message"])
}

func (s *ErrorHandlerTestSuite) TestError_TypedError_WithStatusZero_SetsInternalServerError() {
	// Arrange
	rr := httptest.NewRecorder()
	rErr := errs.New("bad_request", "bad", 0, nil)

	// Act
	s.sut.Error(rr, rErr)

	// Assert
	s.Equal(http.StatusInternalServerError, rr.Code)
	s.Equal("application/json", rr.Header().Get("Content-Type"))

	var body map[string]map[string]interface{}
	s.Require().NoError(json.Unmarshal(rr.Body.Bytes(), &body))
	s.Equal("bad_request", body["error"]["code"])
	s.Equal("bad", body["error"]["message"])
}

func (s *ErrorHandlerTestSuite) TestError_TypedError_WithStatus_ReturnsGivenStatus() {
	// Arrange
	rr := httptest.NewRecorder()
	rErr := errs.New("validation_error", "invalid", http.StatusBadRequest, nil)

	// Act
	s.sut.Error(rr, rErr)

	// Assert
	s.Equal(http.StatusBadRequest, rr.Code)
	s.Equal("application/json", rr.Header().Get("Content-Type"))

	var body map[string]map[string]interface{}
	s.Require().NoError(json.Unmarshal(rr.Body.Bytes(), &body))
	s.Equal("validation_error", body["error"]["code"])
	s.Equal("invalid", body["error"]["message"])
}

func (s *ErrorHandlerTestSuite) TestError_ValidationErrors_ReturnsUnprocessableEntity() {
	// Arrange
	type invalidStruct struct {
		Email string `validate:"required,email"`
		Name  string `validate:"required"`
	}
	valErr := s.v.Validate(&invalidStruct{})
	s.Require().Error(valErr)

	rr := httptest.NewRecorder()

	// Act
	s.sut.Error(rr, valErr)

	// Assert
	s.Equal(http.StatusUnprocessableEntity, rr.Code)
	s.Equal("application/json", rr.Header().Get("Content-Type"))

	var body map[string]interface{}
	s.Require().NoError(json.Unmarshal(rr.Body.Bytes(), &body))
	errorObj := body["error"].(map[string]interface{})
	s.Equal("INVALID_ARGUMENT", errorObj["code"])
	s.Equal("request has invalid fields", errorObj["message"])

	details, ok := errorObj["details"].([]interface{})
	s.True(ok)
	s.GreaterOrEqual(len(details), 1)

	firstDetail := details[0].(map[string]interface{})
	s.Contains([]string{"email", "name"}, firstDetail["field"])
	s.NotEmpty(firstDetail["message"])
}

func (s *ErrorHandlerTestSuite) TestError_WithNilLogger_DoesNotPanic() {
	// Arrange: handler with nil logger uses log.Default() fallback
	rr := httptest.NewRecorder()
	handler := response.NewErrorHandler(nil, nil)
	err := errors.New("boom")

	// Act & Assert: should not panic
	handler.Error(rr, err)
	s.Equal(http.StatusInternalServerError, rr.Code)
	s.Equal("internal_server_error", s.parseError(rr)["code"])
}

func (s *ErrorHandlerTestSuite) parseError(rr *httptest.ResponseRecorder) map[string]interface{} {
	var body map[string]map[string]interface{}
	s.Require().NoError(json.Unmarshal(rr.Body.Bytes(), &body))
	return body["error"]
}

func (s *ErrorHandlerTestSuite) TestError_ValidationErrors_WithNilValidator_ReturnsFallbackFormat() {
	// Arrange
	type invalidStruct struct {
		Email string `validate:"required"`
	}
	valErr := s.v.Validate(&invalidStruct{})
	s.Require().Error(valErr)

	rr := httptest.NewRecorder()
	handler := response.NewErrorHandler(nil, s.log)

	// Act
	handler.Error(rr, valErr)

	// Assert
	s.Equal(http.StatusUnprocessableEntity, rr.Code)
	s.Equal("application/json", rr.Header().Get("Content-Type"))

	var body map[string]interface{}
	s.Require().NoError(json.Unmarshal(rr.Body.Bytes(), &body))
	errorObj := body["error"].(map[string]interface{})
	s.Equal("INVALID_ARGUMENT", errorObj["code"])
	s.Equal("request has invalid fields", errorObj["message"])

	details, ok := errorObj["details"].([]interface{})
	s.True(ok)
	s.Len(details, 1)
	firstDetail := details[0].(map[string]interface{})
	s.Equal("email", firstDetail["field"])
	s.Contains(firstDetail["message"], "email")
}
