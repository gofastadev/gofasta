package scheduler

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockJob implements the Job interface for testing.
type mockJob struct {
	name     string
	runCount atomic.Int32
	runErr   error // returned by Run when non-nil
}

func (m *mockJob) Name() string { return m.name }

func (m *mockJob) Run(_ context.Context) error {
	m.runCount.Add(1)
	return m.runErr
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestNew(t *testing.T) {
	s := New(testLogger())
	assert.NotNil(t, s)
	assert.NotNil(t, s.cron)
	assert.NotNil(t, s.logger)
}

func TestScheduler_Register(t *testing.T) {
	tests := []struct {
		name      string
		schedule  string
		expectErr bool
	}{
		{
			name:      "valid schedule with seconds",
			schedule:  "0 0 * * * *",
			expectErr: false,
		},
		{
			name:      "valid every-second schedule",
			schedule:  "* * * * * *",
			expectErr: false,
		},
		{
			name:      "invalid schedule expression",
			schedule:  "not-a-cron",
			expectErr: true,
		},
		{
			name:      "empty schedule",
			schedule:  "",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := New(testLogger())
			job := &mockJob{name: "test-job"}

			err := s.Register(tt.schedule, job)
			if tt.expectErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "failed to register job")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestScheduler_StartAndStop(t *testing.T) {
	s := New(testLogger())
	job := &mockJob{name: "frequent-job"}

	// Run every second
	err := s.Register("* * * * * *", job)
	require.NoError(t, err)

	s.Start()

	// Wait enough for at least one execution
	time.Sleep(1500 * time.Millisecond)

	s.Stop()

	count := job.runCount.Load()
	assert.GreaterOrEqual(t, count, int32(1), "job should have run at least once")
}

func TestScheduler_RegisterMultipleJobs(t *testing.T) {
	s := New(testLogger())
	job1 := &mockJob{name: "job-1"}
	job2 := &mockJob{name: "job-2"}

	err := s.Register("* * * * * *", job1)
	require.NoError(t, err)

	err = s.Register("* * * * * *", job2)
	require.NoError(t, err)

	s.Start()
	time.Sleep(1500 * time.Millisecond)
	s.Stop()

	assert.GreaterOrEqual(t, job1.runCount.Load(), int32(1))
	assert.GreaterOrEqual(t, job2.runCount.Load(), int32(1))
}

// TestScheduler_FailingJobIsLoggedNotPanicked — when Run returns an
// error, the scheduler logs it and continues. The job still executed,
// so runCount increments; subsequent ticks keep firing.
func TestScheduler_FailingJobIsLoggedNotPanicked(t *testing.T) {
	s := New(testLogger())
	job := &mockJob{name: "failing-job", runErr: errors.New("synthetic failure")}

	require.NoError(t, s.Register("* * * * * *", job))
	s.Start()
	time.Sleep(1500 * time.Millisecond)
	s.Stop()

	assert.GreaterOrEqual(t, job.runCount.Load(), int32(1),
		"a failing job's Run should still execute; the scheduler logs the error and continues")
}

// TestScheduler_StopCancelsJobContext — jobs that observe ctx.Done()
// can wind down on Stop(). Verifies the scheduler propagates
// cancellation through the ctx it passes to Run.
func TestScheduler_StopCancelsJobContext(t *testing.T) {
	var observedCancel atomic.Bool
	gotCtx := make(chan context.Context, 1)
	jobStarted := make(chan struct{}, 1)

	s := New(testLogger())
	job := &ctxObservingJob{
		name:     "ctx-job",
		started:  jobStarted,
		gotCtx:   gotCtx,
		canceled: &observedCancel,
	}
	require.NoError(t, s.Register("* * * * * *", job))
	s.Start()
	<-jobStarted
	s.Stop()
	// The job's goroutine sets observedCancel when its ctx is canceled.
	// Stop returned only after cron.Stop's wait-group released, so by
	// the time we get here the job's <-ctx.Done() has unblocked.
	assert.True(t, observedCancel.Load(),
		"job ctx must observe cancellation when scheduler.Stop is called")
}

type ctxObservingJob struct {
	name     string
	started  chan struct{}
	gotCtx   chan context.Context
	canceled *atomic.Bool
}

func (j *ctxObservingJob) Name() string { return j.name }
func (j *ctxObservingJob) Run(ctx context.Context) error {
	select {
	case j.started <- struct{}{}:
	default:
	}
	select {
	case j.gotCtx <- ctx:
	default:
	}
	<-ctx.Done()
	j.canceled.Store(true)
	return nil
}
