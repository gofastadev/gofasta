package mailer

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

// errReadCloser fails on the first Read, to exercise the response-read error path.
type errReadCloser struct{}

func (errReadCloser) Read([]byte) (int, error) { return 0, errors.New("read boom") }
func (errReadCloser) Close() error             { return nil }

func TestPostJSON_Success(t *testing.T) {
	var capturedReq *http.Request
	var capturedBody []byte

	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		capturedReq = req
		var err error
		capturedBody, err = io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(`{"id":"msg-1"}`)),
		}, nil
	})}

	body, err := postJSON(context.Background(), client, "acme", "https://api.acme.test/send",
		map[string]string{"Authorization": "Bearer key"}, map[string]string{"subject": "hi"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(body) != `{"id":"msg-1"}` {
		t.Errorf("body = %q, want %q", body, `{"id":"msg-1"}`)
	}
	if capturedReq.Method != http.MethodPost {
		t.Errorf("method = %q, want POST", capturedReq.Method)
	}
	if got := capturedReq.Header.Get("Authorization"); got != "Bearer key" {
		t.Errorf("Authorization = %q, want %q", got, "Bearer key")
	}
	if got := capturedReq.Header.Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", got)
	}
	if got := capturedReq.Header.Get("Accept"); got != "application/json" {
		t.Errorf("Accept = %q, want application/json", got)
	}
	if string(capturedBody) != `{"subject":"hi"}` {
		t.Errorf("request body = %q, want %q", capturedBody, `{"subject":"hi"}`)
	}
}

func TestPostJSON_MarshalError(t *testing.T) {
	_, err := postJSON(context.Background(), &http.Client{}, "acme", "https://api.acme.test/send",
		nil, make(chan int))
	if err == nil {
		t.Fatal("expected error for unmarshalable payload")
	}
	if !strings.Contains(err.Error(), "acme marshal:") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "acme marshal:")
	}
}

func TestPostJSON_RequestError(t *testing.T) {
	_, err := postJSON(context.Background(), &http.Client{}, "acme", "http://bad url\x7f",
		nil, map[string]string{})
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
	if !strings.Contains(err.Error(), "acme request:") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "acme request:")
	}
}

func TestPostJSON_ClientError(t *testing.T) {
	clientErr := errors.New("connection refused")
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, clientErr
	})}

	_, err := postJSON(context.Background(), client, "acme", "https://api.acme.test/send", nil, map[string]string{})
	if err == nil {
		t.Fatal("expected error when client fails")
	}
	if !errors.Is(err, clientErr) {
		t.Errorf("error = %q, want it to wrap %q", err.Error(), clientErr.Error())
	}
	if !strings.Contains(err.Error(), "acme send:") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "acme send:")
	}
}

func TestPostJSON_ResponseReadError(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: errReadCloser{}}, nil
	})}

	_, err := postJSON(context.Background(), client, "acme", "https://api.acme.test/send", nil, map[string]string{})
	if err == nil {
		t.Fatal("expected error when response body read fails")
	}
	if !strings.Contains(err.Error(), "acme response:") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "acme response:")
	}
}

func TestPostJSON_APIError(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 422,
			Body:       io.NopCloser(strings.NewReader(`{"message":"invalid recipient"}`)),
		}, nil
	})}

	_, err := postJSON(context.Background(), client, "acme", "https://api.acme.test/send", nil, map[string]string{})
	if err == nil {
		t.Fatal("expected error for 422 status")
	}
	for _, want := range []string{"acme error:", "status 422", "invalid recipient"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error = %q, want it to contain %q", err.Error(), want)
		}
	}
}
