package payment

import (
	"encoding/json"
	"fmt"
	"strings"
)

type responseEnvelope struct {
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func decodeResponse(statusCode int, body []byte, out any) error {
	var envelope responseEnvelope
	if len(body) > 0 {
		if err := json.Unmarshal(body, &envelope); err != nil {
			if statusCode < 200 || statusCode >= 300 {
				return &APIError{StatusCode: statusCode, Body: body}
			}
			return fmt.Errorf("payment: decode response envelope: %w", err)
		}
	}

	if statusCode < 200 || statusCode >= 300 {
		return &APIError{
			StatusCode: statusCode,
			Message:    strings.TrimSpace(envelope.Message),
			Body:       body,
		}
	}
	if out == nil || len(envelope.Data) == 0 || string(envelope.Data) == "null" {
		return nil
	}
	if err := json.Unmarshal(envelope.Data, out); err != nil {
		return fmt.Errorf("payment: decode response data: %w", err)
	}
	return nil
}
