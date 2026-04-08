package resilience

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRetryPolicy(t *testing.T) {
	tests := []struct {
		name       string
		maxRetries int
		delay      time.Duration
	}{
		{
			name:       "standard retry policy",
			maxRetries: 3,
			delay:      100 * time.Millisecond,
		},
		{
			name:       "single retry",
			maxRetries: 1,
			delay:      50 * time.Millisecond,
		},
		{
			name:       "zero retries",
			maxRetries: 0,
			delay:      10 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rp := NewRetryPolicy[string](tt.maxRetries, tt.delay)
			assert.NotNil(t, rp)
		})
	}
}

func TestNewCircuitBreaker(t *testing.T) {
	cb := NewCircuitBreaker[string](3, time.Second)
	assert.NotNil(t, cb)
}

func TestExecute_Success(t *testing.T) {
	rp := NewRetryPolicy[string](3, 10*time.Millisecond)

	result, err := Execute(func() (string, error) {
		return "hello", nil
	}, rp)

	require.NoError(t, err)
	assert.Equal(t, "hello", result)
}

func TestExecute_RetryThenSuccess(t *testing.T) {
	rp := NewRetryPolicy[string](3, 10*time.Millisecond)

	var attempts atomic.Int32
	result, err := Execute(func() (string, error) {
		count := attempts.Add(1)
		if count < 3 {
			return "", errors.New("temporary error")
		}
		return "recovered", nil
	}, rp)

	require.NoError(t, err)
	assert.Equal(t, "recovered", result)
	assert.Equal(t, int32(3), attempts.Load())
}

func TestExecute_AllRetriesExhausted(t *testing.T) {
	rp := NewRetryPolicy[string](2, 10*time.Millisecond)

	var attempts atomic.Int32
	_, err := Execute(func() (string, error) {
		attempts.Add(1)
		return "", errors.New("persistent error")
	}, rp)

	assert.Error(t, err)
	// maxRetries=2 means initial attempt + 2 retries = 3 total attempts
	assert.Equal(t, int32(3), attempts.Load())
}

func TestExecute_WithCircuitBreaker(t *testing.T) {
	cb := NewCircuitBreaker[string](2, 5*time.Second)

	// Trigger failures to open the circuit
	for i := 0; i < 2; i++ {
		_, _ = Execute(func() (string, error) {
			return "", errors.New("fail")
		}, cb)
	}

	// Circuit should now be open, subsequent calls should fail immediately
	_, err := Execute(func() (string, error) {
		return "should not reach", nil
	}, cb)
	assert.Error(t, err)
}

func TestExecute_SuccessWithCircuitBreaker(t *testing.T) {
	cb := NewCircuitBreaker[int](5, time.Second)

	result, err := Execute(func() (int, error) {
		return 42, nil
	}, cb)

	require.NoError(t, err)
	assert.Equal(t, 42, result)
}
