package database

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type widget struct {
	ID   uint `gorm:"primaryKey"`
	Name string
}

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&widget{}))
	return db
}

func count(t *testing.T, db *gorm.DB) int64 {
	t.Helper()
	var n int64
	require.NoError(t, db.Model(&widget{}).Count(&n).Error)
	return n
}

func TestWithTx_CommitsOnSuccess(t *testing.T) {
	db := newTestDB(t)

	err := WithTx(context.Background(), db, func(ctx context.Context) error {
		return FromContext(ctx, db).Create(&widget{Name: "a"}).Error
	})

	require.NoError(t, err)
	assert.Equal(t, int64(1), count(t, db))
}

// TestWithTx_RollsBackEveryWriteInTheGroup is the behavior the package exists
// for: without transaction propagation each repository call would commit on its
// own, so a later failure would leave the earlier writes behind.
func TestWithTx_RollsBackEveryWriteInTheGroup(t *testing.T) {
	db := newTestDB(t)
	sentinel := errors.New("second step failed")

	err := WithTx(context.Background(), db, func(ctx context.Context) error {
		if err := FromContext(ctx, db).Create(&widget{Name: "first"}).Error; err != nil {
			return err
		}
		if err := FromContext(ctx, db).Create(&widget{Name: "second"}).Error; err != nil {
			return err
		}
		return sentinel
	})

	assert.ErrorIs(t, err, sentinel)
	assert.Equal(t, int64(0), count(t, db), "both writes must roll back, not just the last one")
}

func TestFromContext_FallsBackOutsideTransaction(t *testing.T) {
	db := newTestDB(t)

	require.NoError(t, FromContext(context.Background(), db).Create(&widget{Name: "loose"}).Error)
	assert.Equal(t, int64(1), count(t, db))
}

// TestWithTx_NestedJoinsOuter pins that a nested call does NOT open a second
// transaction or a savepoint: an inner rollback must take the outer work with
// it, because the outer caller asked for all-or-nothing.
func TestWithTx_NestedJoinsOuter(t *testing.T) {
	db := newTestDB(t)
	sentinel := errors.New("inner failed")

	err := WithTx(context.Background(), db, func(ctx context.Context) error {
		if err := FromContext(ctx, db).Create(&widget{Name: "outer"}).Error; err != nil {
			return err
		}
		return WithTx(ctx, db, func(ctx context.Context) error {
			if err := FromContext(ctx, db).Create(&widget{Name: "inner"}).Error; err != nil {
				return err
			}
			return sentinel
		})
	})

	assert.ErrorIs(t, err, sentinel)
	assert.Equal(t, int64(0), count(t, db))
}

func TestInTx(t *testing.T) {
	db := newTestDB(t)

	assert.False(t, InTx(context.Background()))
	require.NoError(t, WithTx(context.Background(), db, func(ctx context.Context) error {
		assert.True(t, InTx(ctx))
		return nil
	}))
}
