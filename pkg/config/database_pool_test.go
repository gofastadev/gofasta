package config

import (
	"testing"
	"time"
)

func TestPoolOrDefaults_FillsUnsetLimits(t *testing.T) {
	// database/sql reads a zero MaxOpenConns as unlimited, so a caller that
	// builds a DatabaseConfig literal rather than going through LoadConfig
	// would otherwise get an unbounded pool.
	got := poolOrDefaults(&DatabaseConfig{})

	if got.idle != defaultMaxIdleConns {
		t.Errorf("idle = %d, want %d", got.idle, defaultMaxIdleConns)
	}
	if got.open != defaultMaxOpenConns {
		t.Errorf("open = %d, want %d — zero means unlimited", got.open, defaultMaxOpenConns)
	}
	if got.life != defaultConnMaxLifetime {
		t.Errorf("life = %v, want %v", got.life, defaultConnMaxLifetime)
	}
}

func TestPoolOrDefaults_KeepsExplicitLimits(t *testing.T) {
	got := poolOrDefaults(&DatabaseConfig{MaxIdle: 3, MaxOpen: 7, MaxLife: time.Minute})

	if got.idle != 3 || got.open != 7 || got.life != time.Minute {
		t.Errorf("explicit limits were overridden: %+v", got)
	}
}
