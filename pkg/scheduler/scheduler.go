package scheduler

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/robfig/cron/v3"
)

// Job is the interface every cron job must implement.
//
// Run accepts a context that the scheduler cancels when Stop() is
// called — well-behaved jobs check ctx.Done() during long operations
// so a graceful shutdown doesn't have to wait for them. Run returns
// an error so failures are visible (the scheduler logs them) rather
// than silently swallowed inside the job body.
type Job interface {
	Name() string
	Run(ctx context.Context) error
}

// Scheduler manages cron jobs. Register jobs, then call Start().
// Call Stop() on graceful shutdown to wait for running jobs to finish
// and propagate cancellation to any that respect ctx.
type Scheduler struct {
	cron   *cron.Cron
	logger *slog.Logger
	ctx    context.Context // long-lived; canceled in Stop() to signal shutdown
	cancel context.CancelFunc
}

// New creates a new scheduler with second-precision cron support.
func New(logger *slog.Logger) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())
	return &Scheduler{
		cron:   cron.New(cron.WithSeconds(), cron.WithLogger(cron.PrintfLogger(slog.NewLogLogger(logger.Handler(), slog.LevelDebug)))),
		logger: logger,
		ctx:    ctx,
		cancel: cancel,
	}
}

// Register adds a job with the given cron schedule expression.
// Standard cron: "0 */6 * * *" (every 6h), with seconds: "0 0 */6 * * *"
//
// Each invocation passes the scheduler's long-lived context to Run, so
// jobs can observe shutdown via ctx.Done(). Errors returned by Run are
// logged at ERROR level with the job name attached.
func (s *Scheduler) Register(schedule string, job Job) error {
	_, err := s.cron.AddFunc(schedule, func() {
		s.logger.InfoContext(s.ctx, "job started", "job", job.Name())
		if err := job.Run(s.ctx); err != nil {
			s.logger.ErrorContext(s.ctx, "job failed", "job", job.Name(), "error", err)
			return
		}
		s.logger.InfoContext(s.ctx, "job completed", "job", job.Name())
	})
	if err != nil {
		return fmt.Errorf("failed to register job %q with schedule %q: %w", job.Name(), schedule, err)
	}
	s.logger.Info("job registered", "job", job.Name(), "schedule", schedule)
	return nil
}

// Start begins executing registered jobs on their schedules.
func (s *Scheduler) Start() {
	s.logger.Info("scheduler started")
	s.cron.Start()
}

// Stop waits for running jobs to finish, then stops the scheduler.
// Cancels the scheduler context first so in-flight jobs see ctx.Done()
// and can wind down promptly; cron.Stop() then waits for them to return.
func (s *Scheduler) Stop() {
	s.logger.Info("scheduler stopping, waiting for running jobs...")
	s.cancel()
	ctx := s.cron.Stop()
	<-ctx.Done()
	s.logger.Info("scheduler stopped")
}
