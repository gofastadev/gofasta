package scheduler

import (
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
}

func (m *mockJob) Name() string { return m.name }
func (m *mockJob) Run()         { m.runCount.Add(1) }

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
