package auditlog

import (
	"context"
	"fmt"
	"testing"
	"time"

	"gorm.io/gorm"
)

// countRows returns the number of rows the table holds, soft-deleted ones
// included. Retention has to be measured with Unscoped: Entry carries a
// gorm.DeletedAt, so a "delete" that a normal query cannot see may still be
// occupying the disk this whole mechanism exists to reclaim.
func countRows(t *testing.T, db *gorm.DB) int64 {
	t.Helper()
	var n int64
	if err := db.Unscoped().Model(&Entry{}).Count(&n).Error; err != nil {
		t.Fatalf("counting audit_logs: %v", err)
	}
	return n
}

// countLive returns the rows a normal query still sees.
func countLive(t *testing.T, db *gorm.DB) int64 {
	t.Helper()
	var n int64
	if err := db.Model(&Entry{}).Count(&n).Error; err != nil {
		t.Fatalf("counting live audit_logs: %v", err)
	}
	return n
}

func seedAged(t *testing.T, db *gorm.DB, prefix string, count int, age time.Duration) {
	t.Helper()
	for i := 0; i < count; i++ {
		entry := Entry{
			ID:          fmt.Sprintf("%s-%d", prefix, i),
			EventType:   Logout,
			ServiceName: "solago",
			CreatedAt:   time.Now().Add(-age),
		}
		if err := db.Create(&entry).Error; err != nil {
			t.Fatalf("seeding %s-%d: %v", prefix, i, err)
		}
	}
}

func TestDefaultRetentionConfig(t *testing.T) {
	cfg := DefaultRetentionConfig()

	if cfg.MaxAge != 90*24*time.Hour {
		t.Errorf("MaxAge = %v, want 90 days", cfg.MaxAge)
	}
	if cfg.CleanupInterval != 24*time.Hour {
		t.Errorf("CleanupInterval = %v, want 24h", cfg.CleanupInterval)
	}
	if cfg.BatchSize != 1000 {
		t.Errorf("BatchSize = %d, want 1000", cfg.BatchSize)
	}
}

func TestRunCleanup_RemovesOnlyEntriesPastTheCutoff(t *testing.T) {
	db := newAuditDB(t)
	s := NewAuditService(db, "solago")

	seedAged(t, db, "old", 3, 100*24*time.Hour)
	seedAged(t, db, "recent", 2, 1*time.Hour)

	s.runCleanup(RetentionConfig{MaxAge: 90 * 24 * time.Hour, BatchSize: 100})

	if got := countLive(t, db); got != 2 {
		t.Errorf("%d rows still queryable, want the 2 inside the retention window", got)
	}
}

func TestRunCleanup_DrainsInBatches(t *testing.T) {
	// The batch limit is what keeps a first run against years of history from
	// taking a single lock over the whole table. It only works if the loop
	// keeps going after a full batch — a single-pass implementation would
	// leave everything past the first 2 rows behind, and the backlog would
	// never shrink.
	db := newAuditDB(t)
	s := NewAuditService(db, "solago")

	seedAged(t, db, "old", 7, 100*24*time.Hour)

	s.runCleanup(RetentionConfig{MaxAge: 90 * 24 * time.Hour, BatchSize: 2})

	if got := countLive(t, db); got != 0 {
		t.Errorf("%d rows survived a batched cleanup, want 0", got)
	}
}

func TestRunCleanup_StopsOnADatabaseError(t *testing.T) {
	// No audit_logs table. The cleanup runs on a ticker with nobody watching
	// its return value, so the only requirement is that it gives up rather
	// than spinning on the same failing statement forever.
	s := NewAuditService(newEmptyDB(t), "solago")

	done := make(chan struct{})
	go func() {
		defer close(done)
		s.runCleanup(RetentionConfig{MaxAge: time.Hour, BatchSize: 10})
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("runCleanup did not return after a database error")
	}
}

func TestRunCleanup_NothingToDoIsSilent(t *testing.T) {
	db := newAuditDB(t)
	s := NewAuditService(db, "solago")

	seedAged(t, db, "recent", 2, time.Hour)

	s.runCleanup(RetentionConfig{MaxAge: 90 * 24 * time.Hour, BatchSize: 100})

	if got := countRows(t, db); got != 2 {
		t.Errorf("%d rows, want both left alone", got)
	}
}

func TestStartRetentionCleanup_RunsOnTheTickerAndStopsWhenCanceled(t *testing.T) {
	db := newAuditDB(t)
	s := NewAuditService(db, "solago")

	seedAged(t, db, "old", 3, 100*24*time.Hour)

	ctx, cancel := context.WithCancel(context.Background())
	s.StartRetentionCleanup(ctx, RetentionConfig{
		MaxAge:          90 * 24 * time.Hour,
		CleanupInterval: 10 * time.Millisecond,
		BatchSize:       100,
	})

	eventually(t, "the first tick to clear the backlog", func() bool {
		return countLive(t, db) == 0
	})

	// Cancellation has to actually stop the loop: a retention goroutine that
	// outlives its context keeps holding a database handle for the life of the
	// process, and there is no second chance to stop it.
	cancel()
	time.Sleep(50 * time.Millisecond)

	seedAged(t, db, "after-cancel", 2, 100*24*time.Hour)
	time.Sleep(100 * time.Millisecond) // several ticks, had it still been running

	if got := countLive(t, db); got != 2 {
		t.Errorf("%d rows left after cancellation, want 2 — the loop kept running", got)
	}
}

func TestStartRetentionCleanup_FillsInAnEmptyConfig(t *testing.T) {
	// A caller passing RetentionConfig{} must get the documented defaults, not
	// a zero MaxAge that deletes everything and a zero interval that makes
	// time.NewTicker panic.
	db := newAuditDB(t)
	s := NewAuditService(db, "solago")

	seedAged(t, db, "recent", 2, time.Hour)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s.StartRetentionCleanup(ctx, RetentionConfig{})

	// The defaulted interval is 24h, so nothing can have run yet — which is
	// exactly the evidence that the zero interval was replaced rather than
	// used.
	time.Sleep(50 * time.Millisecond)
	if got := countRows(t, db); got != 2 {
		t.Errorf("%d rows, want 2 — a defaulted 24h interval cannot have ticked", got)
	}
}

func TestStartRetentionCleanup_NilContextIsAccepted(t *testing.T) {
	// Documented as safe, and worth holding to: a caller that has no context
	// to hand in gets a background one rather than a nil-pointer panic inside
	// the goroutine, where nothing would recover it.
	db := newAuditDB(t)
	s := NewAuditService(db, "solago")

	// Held in a variable rather than written as a literal nil at the call
	// site, which is the shape vet and staticcheck flag.
	var noContext context.Context

	s.StartRetentionCleanup(noContext, RetentionConfig{
		MaxAge:          90 * 24 * time.Hour,
		CleanupInterval: 24 * time.Hour,
		BatchSize:       100,
	})

	// The 24h interval means the goroutine parks on its ticker without ever
	// touching the database, so it cannot outlive the test in any way that
	// matters. What is being asserted is that it started at all.
	time.Sleep(20 * time.Millisecond)
	if got := countRows(t, db); got != 0 {
		t.Errorf("%d rows, want 0 — nothing was seeded", got)
	}
}

// The doc comment on StartRetentionCleanup says "hard-deletes", and this
// records what the code actually does instead: Entry carries a gorm.DeletedAt,
// so GORM issues a soft delete and the row stays on disk with deleted_at set.
// Retention that only hides rows does not reclaim anything, which matters for
// the table that grows fastest — flagged here rather than changed, because
// switching to Unscoped would start destroying data on every deployment that
// has this running.
func TestRunCleanup_SoftDeletesRatherThanReclaiming(t *testing.T) {
	db := newAuditDB(t)
	s := NewAuditService(db, "solago")

	seedAged(t, db, "old", 3, 100*24*time.Hour)

	s.runCleanup(RetentionConfig{MaxAge: 90 * 24 * time.Hour, BatchSize: 100})

	if live := countLive(t, db); live != 0 {
		t.Errorf("%d rows still queryable, want 0", live)
	}
	if stored := countRows(t, db); stored != 3 {
		t.Errorf("%d rows on disk, want 3 — see this test's comment", stored)
	}
}
