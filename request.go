package payment

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func (c *Client) doJSON(ctx context.Context, method string, path string, body any, out any) error {
	var rawBody []byte
	var err error
	if body != nil {
		rawBody, err = json.Marshal(body)
		if err != nil {
			return fmt.Errorf("payment: marshal request body: %w", err)
		}
	}

	urlValue := *c.baseURL
	urlValue.Path = joinPath(c.baseURL.Path, path)
	req, err := http.NewRequestWithContext(ctx, method, urlValue.String(), bytes.NewReader(rawBody))
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)

	timestamp := c.now()
	nonce, err := c.nonce()
	if err != nil {
		return fmt.Errorf("payment: generate nonce: %w", err)
	}
	signature := SignRequest(c.secret, SignRequestInput{
		Method:    method,
		Path:      path,
		Timestamp: timestamp,
		Nonce:     nonce,
		Body:      rawBody,
	})

	req.Header.Set(headerMerchantID, c.merchantID)
	req.Header.Set(headerTimestamp, timestamp)
	req.Header.Set(headerNonce, nonce)
	req.Header.Set(headerSignature, signature)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("payment: read response body: %w", err)
	}
	return decodeResponse(resp.StatusCode, respBody, out)
}

func joinPath(basePath string, path string) string {
	if basePath == "" || basePath == "/" {
		return path
	}
	if path == "" || path == "/" {
		return basePath
	}
	if basePath[len(basePath)-1] == '/' && path[0] == '/' {
		return basePath + path[1:]
	}
	if basePath[len(basePath)-1] != '/' && path[0] != '/' {
		return basePath + "/" + path
	}
	return basePath + path
}
