package resilience

import (
	"time"

	"github.com/failsafe-go/failsafe-go"
	"github.com/failsafe-go/failsafe-go/circuitbreaker"
	"github.com/failsafe-go/failsafe-go/retrypolicy"
)

// NewRetryPolicy creates a retry policy with exponential backoff.
func NewRetryPolicy[T any](maxRetries int, delay time.Duration) retrypolicy.RetryPolicy[T] {
	return retrypolicy.NewBuilder[T]().
		WithMaxRetries(maxRetries).
		WithBackoff(delay, delay*10).
		Build()
}

// NewCircuitBreaker creates a circuit breaker that opens after failureThreshold failures.
func NewCircuitBreaker[T any](failureThreshold uint, delay time.Duration) circuitbreaker.CircuitBreaker[T] {
	return circuitbreaker.NewBuilder[T]().
		WithFailureThreshold(failureThreshold).
		WithDelay(delay).
		Build()
}

// Execute runs a function with the given failsafe policies (retry, circuit breaker, etc).
func Execute[T any](fn func() (T, error), policies ...failsafe.Policy[T]) (T, error) {
	return failsafe.With[T](policies...).Get(fn)
}
