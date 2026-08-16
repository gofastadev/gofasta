package auditlog

import (
	"context"
	"log"
	"time"
)

// RetentionConfig configures automatic cleanup of old audit log entries.
type RetentionConfig struct {
	MaxAge          time.Duration // How old entries must be before deletion. Default: 90 days.
	CleanupInterval time.Duration // How often to run cleanup. Default: 24 hours.
	BatchSize       int           // Max rows to delete per batch. Default: 1000.
}

// DefaultRetentionConfig returns a RetentionConfig with sensible defaults.
func DefaultRetentionConfig() RetentionConfig {
	return RetentionConfig{
		MaxAge:          90 * 24 * time.Hour,
		CleanupInterval: 24 * time.Hour,
		BatchSize:       1000,
	}
}

// StartRetentionCleanup runs a background goroutine that periodically
// hard-deletes audit log entries older than MaxAge. It stops when ctx
// is canceled. Safe to call with a nil context (uses background context).
func (s *Service) StartRetentionCleanup(ctx context.Context, cfg RetentionConfig) {
	if cfg.MaxAge <= 0 {
		cfg.MaxAge = 90 * 24 * time.Hour
	}
	if cfg.CleanupInterval <= 0 {
		cfg.CleanupInterval = 24 * time.Hour
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 1000
	}
	if ctx == nil {
		ctx = context.Background()
	}

	go func() {
		ticker := time.NewTicker(cfg.CleanupInterval)
		defer ticker.Stop()

		log.Printf("[Entry] Retention cleanup started for %s (max age: %v, interval: %v, batch: %d)",
			s.ServiceName, cfg.MaxAge, cfg.CleanupInterval, cfg.BatchSize)

		for {
			select {
			case <-ctx.Done():
				log.Printf("[Entry] Retention cleanup stopped for %s", s.ServiceName)
				return
			case <-ticker.C:
				s.runCleanup(cfg)
			}
		}
	}()
}

func (s *Service) runCleanup(cfg RetentionConfig) {
	cutoff := time.Now().Add(-cfg.MaxAge)
	totalDeleted := int64(0)

	for {
		result := s.DB.
			Where("created_at < ? AND deleted_at IS NULL", cutoff).
			Limit(cfg.BatchSize).
			Delete(&Entry{})

		if result.Error != nil {
			log.Printf("[Entry] Retention cleanup error for %s: %v", s.ServiceName, result.Error)
			return
		}

		totalDeleted += result.RowsAffected

		// If fewer rows deleted than batch size, we're done
		if result.RowsAffected < int64(cfg.BatchSize) {
			break
		}
	}

	if totalDeleted > 0 {
		log.Printf("[Entry] Retention cleanup for %s: deleted %d entries older than %v",
			s.ServiceName, totalDeleted, cfg.MaxAge)
	}
}
