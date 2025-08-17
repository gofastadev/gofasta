package orm

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/healtronlabs/gofasta/packages/core"
)

// GORMAdapter implements DatabaseAdapter using GORM for SQL databases
type GORMAdapter struct {
	db     *gorm.DB
	config *DatabaseConfig
	dbType DatabaseType
}

// NewGORMAdapter creates a new GORM adapter
func NewGORMAdapter(config *DatabaseConfig) (*GORMAdapter, error) {
	adapter := &GORMAdapter{
		config: config,
		dbType: config.Type,
	}

	if err := adapter.Connect(config.ConnectionURL); err != nil {
		return nil, err
	}

	return adapter, nil
}

// Connect establishes a connection to the database
func (a *GORMAdapter) Connect(connectionString string) error {
	var dialector gorm.Dialector

	switch a.dbType {
	case DatabaseTypePostgreSQL:
		dialector = postgres.Open(connectionString)
	case DatabaseTypeMySQL:
		dialector = mysql.Open(connectionString)
	case DatabaseTypeSQLite:
		dialector = sqlite.Open(connectionString)
	default:
		return core.NewInternalServerException("Unsupported database type for GORM adapter", nil)
	}

	// Configure GORM
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(a.getLogLevel()),
	}

	db, err := gorm.Open(dialector, gormConfig)
	if err != nil {
		return core.NewInternalServerException("Failed to connect to database", err)
	}

	a.db = db

	// Configure connection pool
	if sqlDB, err := db.DB(); err == nil {
		if a.config.MaxConnections > 0 {
			sqlDB.SetMaxOpenConns(a.config.MaxConnections)
			sqlDB.SetMaxIdleConns(a.config.MaxConnections / 2)
		}
	}

	return nil
}

// Close closes the database connection
func (a *GORMAdapter) Close() error {
	if a.db != nil {
		if sqlDB, err := a.db.DB(); err == nil {
			return sqlDB.Close()
		}
	}
	return nil
}

// Create creates a new entity
func (a *GORMAdapter) Create(ctx context.Context, entity interface{}) (interface{}, error) {
	result := a.db.WithContext(ctx).Create(entity)
	if result.Error != nil {
		return nil, a.translateError(result.Error)
	}
	return entity, nil
}

// FindByID finds an entity by its ID
func (a *GORMAdapter) FindByID(ctx context.Context, entityType reflect.Type, id interface{}) (interface{}, error) {
	entity := reflect.New(entityType).Interface()
	result := a.db.WithContext(ctx).First(entity, id)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, core.NewNotFoundException("Entity not found")
		}
		return nil, a.translateError(result.Error)
	}

	return entity, nil
}

// FindOne finds a single entity matching the query
func (a *GORMAdapter) FindOne(ctx context.Context, entityType reflect.Type, query Query) (interface{}, error) {
	entity := reflect.New(entityType).Interface()
	db := a.buildQuery(a.db.WithContext(ctx), query)

	result := db.First(entity)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, core.NewNotFoundException("Entity not found")
		}
		return nil, a.translateError(result.Error)
	}

	return entity, nil
}

// FindMany finds multiple entities matching the query
func (a *GORMAdapter) FindMany(ctx context.Context, entityType reflect.Type, query Query) (interface{}, error) {
	// Create a slice of pointers to the entity type
	sliceType := reflect.SliceOf(reflect.PtrTo(entityType))
	entities := reflect.New(sliceType).Interface()

	db := a.buildQuery(a.db.WithContext(ctx), query)
	result := db.Find(entities)

	if result.Error != nil {
		return nil, a.translateError(result.Error)
	}

	return entities, nil
}

// Update updates an entity
func (a *GORMAdapter) Update(ctx context.Context, entity interface{}) (interface{}, error) {
	result := a.db.WithContext(ctx).Save(entity)
	if result.Error != nil {
		return nil, a.translateError(result.Error)
	}
	return entity, nil
}

// Delete deletes entities matching the query
func (a *GORMAdapter) Delete(ctx context.Context, entityType reflect.Type, query Query) error {
	entity := reflect.New(entityType).Interface()
	db := a.buildQuery(a.db.WithContext(ctx), query)

	result := db.Delete(entity)
	if result.Error != nil {
		return a.translateError(result.Error)
	}

	return nil
}

// Count counts entities matching the query
func (a *GORMAdapter) Count(ctx context.Context, entityType reflect.Type, query Query) (int64, error) {
	var count int64
	entity := reflect.New(entityType).Interface()
	db := a.buildQuery(a.db.WithContext(ctx).Model(entity), query)

	result := db.Count(&count)
	if result.Error != nil {
		return 0, a.translateError(result.Error)
	}

	return count, nil
}

// Transaction executes a function within a database transaction
func (a *GORMAdapter) Transaction(ctx context.Context, fn func(adapter DatabaseAdapter) error) error {
	return a.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txAdapter := &GORMAdapter{
			db:     tx,
			config: a.config,
			dbType: a.dbType,
		}
		return fn(txAdapter)
	})
}

// Migrate performs database migration
func (a *GORMAdapter) Migrate(entities ...interface{}) error {
	return a.db.AutoMigrate(entities...)
}

// GetType returns the database type
func (a *GORMAdapter) GetType() DatabaseType {
	return a.dbType
}

// buildQuery builds a GORM query from the universal Query interface
func (a *GORMAdapter) buildQuery(db *gorm.DB, query Query) *gorm.DB {
	// Apply WHERE conditions
	for _, condition := range query.GetConditions() {
		db = a.applyCondition(db, condition)
	}

	// Apply ORDER BY
	for _, orderBy := range query.GetOrderBy() {
		db = db.Order(fmt.Sprintf("%s %s", orderBy.Field, orderBy.Direction))
	}

	// Apply LIMIT
	if limit := query.GetLimit(); limit > 0 {
		db = db.Limit(limit)
	}

	// Apply OFFSET
	if offset := query.GetOffset(); offset > 0 {
		db = db.Offset(offset)
	}

	// Apply JOINs
	for _, join := range query.GetJoins() {
		switch join.Type {
		case JoinInner:
			db = db.Joins(fmt.Sprintf("INNER JOIN %s ON %s", join.Table, join.Condition))
		case JoinLeft:
			db = db.Joins(fmt.Sprintf("LEFT JOIN %s ON %s", join.Table, join.Condition))
		case JoinRight:
			db = db.Joins(fmt.Sprintf("RIGHT JOIN %s ON %s", join.Table, join.Condition))
		case JoinFull:
			db = db.Joins(fmt.Sprintf("FULL JOIN %s ON %s", join.Table, join.Condition))
		}
	}

	// Apply SELECT fields
	if fields := query.GetFields(); len(fields) > 0 {
		db = db.Select(fields)
	}

	// Apply GROUP BY
	if groupBy := query.GetGroupBy(); len(groupBy) > 0 {
		db = db.Group(strings.Join(groupBy, ", "))
	}

	// Apply HAVING
	for _, having := range query.GetHaving() {
		db = db.Having(having.Condition, having.Args...)
	}

	return db
}

// applyCondition applies a single condition to the GORM query
func (a *GORMAdapter) applyCondition(db *gorm.DB, condition Condition) *gorm.DB {
	switch condition.Operator {
	case OpEquals:
		return db.Where(fmt.Sprintf("%s = ?", condition.Field), condition.Value)
	case OpNotEquals:
		return db.Where(fmt.Sprintf("%s != ?", condition.Field), condition.Value)
	case OpGreaterThan:
		return db.Where(fmt.Sprintf("%s > ?", condition.Field), condition.Value)
	case OpGreaterThanOrEqual:
		return db.Where(fmt.Sprintf("%s >= ?", condition.Field), condition.Value)
	case OpLessThan:
		return db.Where(fmt.Sprintf("%s < ?", condition.Field), condition.Value)
	case OpLessThanOrEqual:
		return db.Where(fmt.Sprintf("%s <= ?", condition.Field), condition.Value)
	case OpLike:
		return db.Where(fmt.Sprintf("%s LIKE ?", condition.Field), condition.Value)
	case OpNotLike:
		return db.Where(fmt.Sprintf("%s NOT LIKE ?", condition.Field), condition.Value)
	case OpIn:
		return db.Where(fmt.Sprintf("%s IN ?", condition.Field), condition.Values)
	case OpNotIn:
		return db.Where(fmt.Sprintf("%s NOT IN ?", condition.Field), condition.Values)
	case OpBetween:
		if len(condition.Values) >= 2 {
			return db.Where(fmt.Sprintf("%s BETWEEN ? AND ?", condition.Field), condition.Values[0], condition.Values[1])
		}
		return db
	case OpNotBetween:
		if len(condition.Values) >= 2 {
			return db.Where(fmt.Sprintf("%s NOT BETWEEN ? AND ?", condition.Field), condition.Values[0], condition.Values[1])
		}
		return db
	case OpIsNull:
		return db.Where(fmt.Sprintf("%s IS NULL", condition.Field))
	case OpIsNotNull:
		return db.Where(fmt.Sprintf("%s IS NOT NULL", condition.Field))
	default:
		// Default to equality
		return db.Where(fmt.Sprintf("%s = ?", condition.Field), condition.Value)
	}
}

// getLogLevel converts string log level to GORM log level
func (a *GORMAdapter) getLogLevel() logger.LogLevel {
	switch strings.ToLower(a.config.LogLevel) {
	case "error":
		return logger.Error
	case "warn", "warning":
		return logger.Warn
	case "info":
		return logger.Info
	case "debug":
		return logger.Info // GORM doesn't have debug level
	default:
		return logger.Warn
	}
}

// translateError translates GORM errors to Gofasta exceptions
func (a *GORMAdapter) translateError(err error) error {
	if err == nil {
		return nil
	}

	switch err {
	case gorm.ErrRecordNotFound:
		return core.NewNotFoundException("Record not found")
	case gorm.ErrInvalidTransaction:
		return core.NewInternalServerException("Invalid transaction", err)
	case gorm.ErrNotImplemented:
		return core.NewInternalServerException("Feature not implemented", err)
	case gorm.ErrMissingWhereClause:
		return core.NewBadRequestException("Missing WHERE clause")
	case gorm.ErrUnsupportedRelation:
		return core.NewInternalServerException("Unsupported relation", err)
	case gorm.ErrPrimaryKeyRequired:
		return core.NewBadRequestException("Primary key required")
	case gorm.ErrModelValueRequired:
		return core.NewBadRequestException("Model value required")
	case gorm.ErrInvalidData:
		return core.NewBadRequestException("Invalid data")
	default:
		// Check for constraint violations
		errStr := err.Error()
		if strings.Contains(errStr, "duplicate") || strings.Contains(errStr, "unique") {
			return core.NewConflictException("Duplicate entry")
		}
		if strings.Contains(errStr, "foreign key") {
			return core.NewBadRequestException("Foreign key constraint violation")
		}
		if strings.Contains(errStr, "not null") {
			return core.NewBadRequestException("Required field is missing")
		}

		return core.NewInternalServerException("Database error", err)
	}
}
