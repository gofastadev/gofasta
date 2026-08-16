package resilience

import (
	"errors"
	"net/http"
	"time"

	"github.com/failsafe-go/failsafe-go"
	"github.com/failsafe-go/failsafe-go/circuitbreaker"
)

// Defaults chosen for calls between services on one internal network, where a
// failure is far more often an instance restarting than a wire problem. Three
// attempts over roughly a second covers a restart; a fourth would mostly add
// latency to a request that is going to fail anyway.
const (
	defaultMaxRetries       = 2
	defaultRetryDelay       = 200 * time.Millisecond
	defaultFailureThreshold = 5
	defaultBreakerDelay     = 30 * time.Second
)

// ErrCircuitOpen reports that the transport is refusing calls because the
// upstream has been failing.
//
// Distinguished from the underlying failure so a caller can tell "the service
// is down" from "we did not even try" — the second means the caller is being
// shielded, not that this particular request found a problem.
var ErrCircuitOpen = errors.New("resilience: upstream circuit is open")

// retryTransport retries transient failures and trips a circuit breaker when
// an upstream stops answering.
type retryTransport struct {
	base    http.RoundTripper
	execute func(func() (*http.Response, error)) (*http.Response, error)
}

// NewRetryTransport wraps base with retry and circuit-breaker behavior.
//
// One transport per upstream. The breaker's state is the transport's, so
// Dinero being down opens Dinero's breaker and leaves Solago's alone; a single
// shared breaker would let one failing dependency stop calls to healthy ones.
//
// Pass nil for base to wrap http.DefaultTransport.
func NewRetryTransport(base http.RoundTripper) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}
	retry := NewRetryPolicy[*http.Response](defaultMaxRetries, defaultRetryDelay)
	breaker := NewCircuitBreaker[*http.Response](defaultFailureThreshold, defaultBreakerDelay)

	return &retryTransport{
		base: base,
		execute: func(fn func() (*http.Response, error)) (*http.Response, error) {
			// Breaker outermost: once it is open the retry policy should not
			// spend attempts on a call that will be rejected immediately.
			return Execute(fn, failsafe.Policy[*http.Response](breaker), retry)
		},
	}
}

// RoundTrip implements http.RoundTripper.
func (t *retryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if !retryable(req) {
		return t.base.RoundTrip(req)
	}

	resp, err := t.execute(func() (*http.Response, error) {
		// Each attempt needs its own body reader. GetBody is populated by
		// http.NewRequest for the in-memory body types; a request built by
		// hand around a streaming reader has none, which retryable() has
		// already excluded.
		attempt := req.Clone(req.Context())
		if req.GetBody != nil {
			body, err := req.GetBody()
			if err != nil {
				return nil, err
			}
			attempt.Body = body
		}

		resp, err := t.base.RoundTrip(attempt)
		if err != nil {
			return nil, err
		}
		if transientStatus(resp.StatusCode) {
			// The body must be drained and closed or the connection leaks,
			// and it must not be handed back or the caller would see a
			// response we are about to discard.
			_ = resp.Body.Close()
			return nil, &transientError{status: resp.StatusCode}
		}
		return resp, nil
	})

	switch {
	case err == nil:
		return resp, nil
	case errors.Is(err, circuitbreaker.ErrOpen):
		return nil, ErrCircuitOpen
	default:
		// A transient status that never recovered is reported as an error
		// rather than as the last response, because the response body has
		// been consumed by then.
		return nil, err
	}
}

// transientError carries the status that caused a retry.
type transientError struct{ status int }

func (e *transientError) Error() string {
	return http.StatusText(e.status)
}

// retryable reports whether a request may be sent more than once.
//
// Two conditions, both necessary:
//
// The method must be idempotent. Retrying a POST is how one payment becomes
// two and one email becomes three — the request may well have been processed
// before the failure was observed, and nothing in an HTTP error distinguishes
// "never arrived" from "arrived, succeeded, and the reply was lost".
//
// The body must be replayable. A request whose body is a stream has already
// been consumed by the first attempt; sending it again would send nothing.
func retryable(req *http.Request) bool {
	switch req.Method {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
		return true
	case http.MethodPut, http.MethodDelete:
		// Idempotent by definition, but only replayable with a rewindable
		// body. A nil Body means there is nothing to replay.
		return req.Body == nil || req.GetBody != nil
	default:
		return false
	}
}

// transientStatus reports whether a status is worth another attempt.
//
// 429 and the 5xx gateway statuses mean "not now"; 500 does not, because an
// unhandled error upstream will keep being unhandled and retrying it just
// multiplies the load on a service already in trouble.
func transientStatus(code int) bool {
	switch code {
	case http.StatusTooManyRequests,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}
