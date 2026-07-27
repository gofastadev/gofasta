// Package database provides transaction propagation for the repository layer.
//
// Generated repositories take a context and hold their own *gorm.DB:
//
//	func (r *UserRepository) Create(ctx context.Context, u *models.User) error {
//	    return r.db.WithContext(ctx).Create(u).Error
//	}
//
// That shape has no way to express "these three repository calls must commit
// or roll back together" — each call reaches for its own handle, so a service
// coordinating several repositories gets one implicit transaction per
// statement. The usual workarounds are to widen every repository method with a
// *gorm.DB parameter, or to give each repository a WithTx(tx) clone; both
// change the signature the generator produces and leak persistence mechanics
// into the service's call sites.
//
// This package keeps the signatures untouched by carrying the transaction on
// the context instead. A repository resolves its handle with FromContext, and
// a service groups work with WithTx:
//
//	func (r *UserRepository) Create(ctx context.Context, u *models.User) error {
//	    return database.FromContext(ctx, r.db).Create(u).Error
//	}
//
//	err := database.WithTx(ctx, s.db, func(ctx context.Context) error {
//	    if err := s.units.Create(ctx, unit); err != nil {
//	        return err
//	    }
//	    return s.quizzes.Create(ctx, quiz)
//	})
//
// Every repository call made with the derived ctx joins the transaction;
// returning an error rolls the whole group back.
package database

import (
	"context"

	"gorm.io/gorm"
)

// txKeyType is unexported so no other package can collide with this context
// key or overwrite an in-flight transaction.
type txKeyType struct{}

var txKey txKeyType

// WithTx runs fn inside a database transaction, passing it a context carrying
// that transaction. Repository calls made with the supplied context — directly
// or further down the call tree — join it automatically via FromContext.
//
// The transaction commits when fn returns nil and rolls back when it returns
// an error or panics.
//
// Nested calls join the outer transaction rather than opening a second one.
// GORM supports savepoints, but silently converting a nested WithTx into a
// savepoint would mean an inner rollback left the outer transaction alive with
// partial work — surprising for a caller who wrote "all or nothing". Joining
// keeps the guarantee the outer call asked for.
func WithTx(ctx context.Context, db *gorm.DB, fn func(ctx context.Context) error) error {
	if tx, ok := ctx.Value(txKey).(*gorm.DB); ok && tx != nil {
		return fn(ctx)
	}
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(context.WithValue(ctx, txKey, tx))
	})
}

// FromContext returns the transaction carried by ctx, or fallback when there
// is none. Repositories call it instead of using their own handle directly, so
// the same method works inside and outside a transaction.
//
// The returned handle already has the context applied, so callers should not
// call WithContext again.
func FromContext(ctx context.Context, fallback *gorm.DB) *gorm.DB {
	if tx, ok := ctx.Value(txKey).(*gorm.DB); ok && tx != nil {
		return tx.WithContext(ctx)
	}
	return fallback.WithContext(ctx)
}

// InTx reports whether ctx is already inside a transaction. Useful for
// assertions and for logging that wants to note transactional work; business
// logic should not branch on it — code that behaves differently inside a
// transaction is code whose behavior depends on its caller.
func InTx(ctx context.Context) bool {
	tx, ok := ctx.Value(txKey).(*gorm.DB)
	return ok && tx != nil
}
