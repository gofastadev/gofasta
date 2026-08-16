package mailer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// postJSON delivers one provider API call: it marshals payload, POSTs it to
// url with JSON content headers plus the provider's auth headers, and returns
// the raw response body. A non-2xx status is an error carrying the provider
// name, status code, and response body, so an API-backed sender only supplies
// its payload mapping and auth header.
func postJSON(ctx context.Context, client *http.Client, provider, url string, headers map[string]string, payload any) ([]byte, error) {
	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("%s marshal: %w", provider, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("%s request: %w", provider, err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s send: %w", provider, err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%s response: %w", provider, err)
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("%s error: status %d, body: %s", provider, resp.StatusCode, string(body))
	}
	return body, nil
}
