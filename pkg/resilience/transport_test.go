package resilience

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

// countingHandler records how many times it was called and answers with the
// scripted statuses in order, repeating the last one once exhausted.
func countingHandler(calls *atomic.Int32, statuses ...int) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		n := int(calls.Add(1))
		status := statuses[len(statuses)-1]
		if n <= len(statuses) {
			status = statuses[n-1]
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte("body"))
	}
}

func clientAgainst(t *testing.T, h http.Handler) (client *http.Client, url string, stop func()) {
	t.Helper()
	srv := httptest.NewServer(h)
	return &http.Client{Transport: NewRetryTransport(nil)}, srv.URL, srv.Close
}

func TestRetryTransport_RetriesTransientStatuses(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(countingHandler(&calls, http.StatusServiceUnavailable, http.StatusOK))
	defer srv.Close()

	client := &http.Client{Transport: NewRetryTransport(nil)}
	resp, err := client.Get(srv.URL)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200 after a retry", resp.StatusCode)
	}
	if got := calls.Load(); got != 2 {
		t.Errorf("upstream called %d times, want 2", got)
	}
}

func TestRetryTransport_DoesNotRetry500(t *testing.T) {
	// An unhandled error upstream will still be unhandled on the next attempt.
	// Retrying it only multiplies load on a service already in trouble.
	var calls atomic.Int32
	srv := httptest.NewServer(countingHandler(&calls, http.StatusInternalServerError))
	defer srv.Close()

	client := &http.Client{Transport: NewRetryTransport(nil)}
	resp, err := client.Get(srv.URL)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("status = %d, want the 500 passed through", resp.StatusCode)
	}
	if got := calls.Load(); got != 1 {
		t.Errorf("upstream called %d times, want 1", got)
	}
}

func TestRetryTransport_NeverRetriesPOST(t *testing.T) {
	// The property that matters most here: a retried POST is how one payment
	// becomes two and one email becomes three. A 503 on a POST must reach the
	// caller unretried.
	var calls atomic.Int32
	srv := httptest.NewServer(countingHandler(&calls, http.StatusServiceUnavailable, http.StatusOK))
	defer srv.Close()

	client := &http.Client{Transport: NewRetryTransport(nil)}
	resp, err := client.Post(srv.URL, "application/json", strings.NewReader(`{}`))
	if err != nil {
		t.Fatalf("Post: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want the 503 passed through unretried", resp.StatusCode)
	}
	if got := calls.Load(); got != 1 {
		t.Errorf("POST was sent %d times, want exactly 1", got)
	}
}

func TestRetryTransport_ReplaysTheBodyOnRetry(t *testing.T) {
	// A retried PUT must carry its body again. Without GetBody the second
	// attempt would send an empty body and quietly wipe the resource.
	var calls atomic.Int32
	var bodies []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		bodies = append(bodies, string(b))
		if calls.Add(1) == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	req, err := http.NewRequest(http.MethodPut, srv.URL, strings.NewReader(`{"name":"x"}`))
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	client := &http.Client{Transport: NewRetryTransport(nil)}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if len(bodies) != 2 {
		t.Fatalf("upstream saw %d requests, want 2", len(bodies))
	}
	for i, b := range bodies {
		if b != `{"name":"x"}` {
			t.Errorf("attempt %d body = %q, want the original payload", i+1, b)
		}
	}
}

func TestRetryTransport_OpensTheCircuit(t *testing.T) {
	// After enough failures the transport stops calling out at all. The
	// assertion is that the upstream stops being called, not merely that the
	// caller sees errors.
	var calls atomic.Int32
	srv := httptest.NewServer(countingHandler(&calls, http.StatusServiceUnavailable))
	defer srv.Close()

	client := &http.Client{Transport: NewRetryTransport(nil)}

	var sawOpen bool
	for range 10 {
		resp, err := client.Get(srv.URL)
		if resp != nil {
			_ = resp.Body.Close()
		}
		if err != nil && errors.Is(err, ErrCircuitOpen) {
			sawOpen = true
			break
		}
	}
	if !sawOpen {
		t.Fatal("circuit never opened against a permanently failing upstream")
	}

	before := calls.Load()
	for range 3 {
		resp, err := client.Get(srv.URL)
		if resp != nil {
			_ = resp.Body.Close()
		}
		if !errors.Is(err, ErrCircuitOpen) {
			t.Fatalf("call after the circuit opened returned %v, want ErrCircuitOpen", err)
		}
	}
	if after := calls.Load(); after != before {
		t.Errorf("upstream called %d more times while the circuit was open", after-before)
	}
}

func TestRetryTransport_PassesSuccessThroughUntouched(t *testing.T) {
	client, url, closeSrv := clientAgainst(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Marker", "present")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("payload"))
	}))
	defer closeSrv()

	resp, err := client.Get(url)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.Header.Get("X-Marker") != "present" {
		t.Error("response headers did not survive the transport")
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "payload" {
		t.Errorf("body = %q, want the upstream payload — the transport must not consume it", body)
	}
}

func TestRetryable(t *testing.T) {
	cases := []struct {
		method string
		body   io.Reader
		want   bool
	}{
		{http.MethodGet, nil, true},
		{http.MethodHead, nil, true},
		{http.MethodOptions, nil, true},
		{http.MethodPut, strings.NewReader("x"), true},
		{http.MethodDelete, nil, true},
		{http.MethodPost, strings.NewReader("x"), false},
		{http.MethodPatch, strings.NewReader("x"), false},
	}
	for _, tc := range cases {
		t.Run(tc.method, func(t *testing.T) {
			req, err := http.NewRequest(tc.method, "http://example.test/", tc.body)
			if err != nil {
				t.Fatalf("NewRequest: %v", err)
			}
			if got := retryable(req); got != tc.want {
				t.Errorf("retryable(%s) = %v, want %v", tc.method, got, tc.want)
			}
		})
	}
}

func TestRetryable_UnreplayableBodyIsNotRetried(t *testing.T) {
	// A streaming body has been consumed by the first attempt; sending it
	// again would send nothing at all.
	req, err := http.NewRequest(http.MethodPut, "http://example.test/", io.LimitReader(strings.NewReader("x"), 1))
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	if retryable(req) {
		t.Error("a PUT with an unrewindable body was marked retryable")
	}
}

// stubTransport answers every attempt with the same canned result.
type stubTransport struct {
	calls atomic.Int32
	resp  *http.Response
	err   error
}

func (s *stubTransport) RoundTrip(_ *http.Request) (*http.Response, error) {
	s.calls.Add(1)
	if s.err != nil {
		return nil, s.err
	}
	return s.resp, nil
}

func TestRetryTransport_ReportsATransportError(t *testing.T) {
	// A connection that never comes up is retried and then reported. Returning
	// a nil error with a nil response here would hand the caller a response to
	// dereference.
	wire := errors.New("dial tcp: connection refused")
	base := &stubTransport{err: wire}

	req, err := http.NewRequest(http.MethodGet, "http://upstream.invalid/health", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	resp, err := NewRetryTransport(base).RoundTrip(req)

	if err == nil {
		t.Fatal("RoundTrip returned nil for a transport that never answered")
	}
	if !errors.Is(err, wire) {
		t.Errorf("err = %v, want the underlying transport error", err)
	}
	if resp != nil {
		t.Errorf("resp = %v, want nil alongside the error", resp)
	}
	if got := base.calls.Load(); got != defaultMaxRetries+1 {
		t.Errorf("base called %d times, want %d", got, defaultMaxRetries+1)
	}
}

func TestRetryTransport_ReportsAFailureToRewindTheBody(t *testing.T) {
	// GetBody is what makes a retry possible at all: each attempt needs its
	// own reader. If it fails, the attempt cannot be made, and sending the
	// request with the previous attempt's drained body would silently PUT
	// nothing.
	rewind := errors.New("cannot reopen body")
	base := &stubTransport{err: errors.New("never reached")}

	req, err := http.NewRequest(http.MethodPut, "http://upstream.invalid/thing",
		strings.NewReader("payload"))
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.GetBody = func() (io.ReadCloser, error) { return nil, rewind }

	resp, err := NewRetryTransport(base).RoundTrip(req)

	if err == nil {
		t.Fatal("RoundTrip returned nil for a body that could not be rewound")
	}
	if !errors.Is(err, rewind) {
		t.Errorf("err = %v, want the rewind failure", err)
	}
	if resp != nil {
		t.Errorf("resp = %v, want nil alongside the error", resp)
	}
	if got := base.calls.Load(); got != 0 {
		t.Errorf("base was called %d times; the request must not be sent at all", got)
	}
}

func TestTransientError_NamesTheStatus(t *testing.T) {
	// This is the error a caller sees when a transient status never recovered,
	// and it is the only thing left saying which status caused the retries —
	// the response body has been drained by then.
	cases := map[int]string{
		http.StatusTooManyRequests:    "Too Many Requests",
		http.StatusBadGateway:         "Bad Gateway",
		http.StatusServiceUnavailable: "Service Unavailable",
		http.StatusGatewayTimeout:     "Gateway Timeout",
	}

	for status, want := range cases {
		err := &transientError{status: status}
		if got := err.Error(); got != want {
			t.Errorf("transientError{%d}.Error() = %q, want %q", status, got, want)
		}
	}
}

func TestRetryTransport_ExhaustedRetriesSurfaceTheStatus(t *testing.T) {
	// End to end: an upstream that answers 503 forever produces the transient
	// error above rather than a response whose body has already been closed.
	var calls atomic.Int32
	srv := httptest.NewServer(countingHandler(&calls, http.StatusServiceUnavailable))
	defer srv.Close()

	client := &http.Client{Transport: NewRetryTransport(nil)}
	resp, err := client.Get(srv.URL)
	if err == nil {
		_ = resp.Body.Close()
		t.Fatal("Get returned a response for an upstream that only ever answered 503")
	}
	if !strings.Contains(err.Error(), "Service Unavailable") {
		t.Errorf("err = %v, want it to name the status that caused the retries", err)
	}
}

func TestTransientStatus(t *testing.T) {
	// 500 is deliberately absent: an unhandled error upstream will keep being
	// unhandled, and retrying it just multiplies load on a service already in
	// trouble.
	retryable := []int{
		http.StatusTooManyRequests,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout,
	}
	for _, code := range retryable {
		if !transientStatus(code) {
			t.Errorf("transientStatus(%d) = false, want true", code)
		}
	}

	for _, code := range []int{
		http.StatusOK,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusInternalServerError,
		http.StatusNotImplemented,
	} {
		if transientStatus(code) {
			t.Errorf("transientStatus(%d) = true, want false", code)
		}
	}
}
