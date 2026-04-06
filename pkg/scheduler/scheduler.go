package scheduler

import (
	"fmt"
	"log/slog"

	"github.com/robfig/cron/v3"
)

// Job is the interface every cron job must implement.
type Job interface {
	Name() string
	Run()
}

// Scheduler manages cron jobs. Register jobs, then call Start().
// Call Stop() on graceful shutdown to wait for running jobs to finish.
type Scheduler struct {
	cron   *cron.Cron
	logger *slog.Logger
}

// New creates a new scheduler with second-precision cron support.
func New(logger *slog.Logger) *Scheduler {
	return &Scheduler{
		cron: cron.New(cron.WithSeconds(), cron.WithLogger(cron.PrintfLogger(slog.NewLogLogger(logger.Handler(), slog.LevelDebug)))),
		logger: logger,
	}
}

// Register adds a job with the given cron schedule expression.
// Standard cron: "0 */6 * * *" (every 6h), with seconds: "0 0 */6 * * *"
func (s *Scheduler) Register(schedule string, job Job) error {
	_, err := s.cron.AddFunc(schedule, func() {
		s.logger.Info("job started", "job", job.Name())
		job.Run()
		s.logger.Info("job completed", "job", job.Name())
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
func (s *Scheduler) Stop() {
	s.logger.Info("scheduler stopping, waiting for running jobs...")
	ctx := s.cron.Stop()
	<-ctx.Done()
	s.logger.Info("scheduler stopped")
}
