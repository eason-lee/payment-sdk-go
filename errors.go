package payment

import (
	"fmt"

	"github.com/go-playground/validator/v10"
)

var errInvalidBaseURL = fmt.Errorf("payment: base url must include scheme and host")

// APIError is returned for non-2xx gateway responses.
type APIError struct {
	StatusCode int
	Message    string
	Body       []byte
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("payment: api error: status=%d message=%q", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("payment: api error: status=%d body=%q", e.StatusCode, string(e.Body))
}

func ValidStruct(obj any) error {
	return validator.New().Struct(obj)
}
