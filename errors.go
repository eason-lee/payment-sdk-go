package payment

import "fmt"

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
