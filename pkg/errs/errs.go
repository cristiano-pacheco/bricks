package errs

import "fmt"

type Error struct {
	Status        int      `json:"-"`
	Code          string   `json:"code"`
	Message       string   `json:"message"`
	Details       []Detail `json:"details,omitempty"`
	OriginalError error    `json:"-"`
}

type Detail struct {
	Field   string `json:"field,omitempty"`
	Message string `json:"message,omitempty"`
}

// Error implements the error interface
func (e *Error) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func New(code, message string, status int, details []Detail) *Error {
	return &Error{
		Status:  status,
		Code:    code,
		Message: message,
		Details: details,
	}
}
