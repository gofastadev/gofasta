package seeds

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// resetSeeders clears the package-level seeders slice between tests.
func resetSeeders() {
	seeders = nil
}

func TestRegister(t *testing.T) {
	resetSeeders()

	assert.Empty(t, seeders)

	Register(func(db *gorm.DB) error { return nil })
	assert.Len(t, seeders, 1)

	Register(func(db *gorm.DB) error { return nil })
	assert.Len(t, seeders, 2)
}

func TestRunAll(t *testing.T) {
	tests := []struct {
		name       string
		seedFuncs  []func(*gorm.DB) error
		expectErr  bool
		errContain string
	}{
		{
			name:      "no seeders registered",
			seedFuncs: nil,
			expectErr: false,
		},
		{
			name: "all seeders succeed",
			seedFuncs: []func(*gorm.DB) error{
				func(db *gorm.DB) error { return nil },
				func(db *gorm.DB) error { return nil },
			},
			expectErr: false,
		},
		{
			name: "first seeder fails",
			seedFuncs: []func(*gorm.DB) error{
				func(db *gorm.DB) error { return errors.New("seed error") },
				func(db *gorm.DB) error { return nil },
			},
			expectErr:  true,
			errContain: "seed #1 failed",
		},
		{
			name: "second seeder fails",
			seedFuncs: []func(*gorm.DB) error{
				func(db *gorm.DB) error { return nil },
				func(db *gorm.DB) error { return errors.New("seed error") },
			},
			expectErr:  true,
			errContain: "seed #2 failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetSeeders()
			for _, fn := range tt.seedFuncs {
				Register(fn)
			}

			err := RunAll(nil) // db is nil since mock functions don't use it
			if tt.expectErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContain)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRunAll_ExecutionOrder(t *testing.T) {
	resetSeeders()

	order := []int{}
	Register(func(db *gorm.DB) error { order = append(order, 1); return nil })
	Register(func(db *gorm.DB) error { order = append(order, 2); return nil })
	Register(func(db *gorm.DB) error { order = append(order, 3); return nil })

	err := RunAll(nil)
	require.NoError(t, err)
	assert.Equal(t, []int{1, 2, 3}, order)
}

func TestRunAll_StopsOnFirstError(t *testing.T) {
	resetSeeders()

	executed := []int{}
	Register(func(db *gorm.DB) error { executed = append(executed, 1); return nil })
	Register(func(db *gorm.DB) error { return errors.New("fail") })
	Register(func(db *gorm.DB) error { executed = append(executed, 3); return nil })

	err := RunAll(nil)
	require.Error(t, err)
	assert.Equal(t, []int{1}, executed, "third seeder should not have run")
}
