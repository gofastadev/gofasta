package orm

import (
	"context"
	"reflect"
	"strings"

	"github.com/healtronlabs/gofasta/packages/core"
)

// DatabaseAdapter defines the interface that database-specific adapters must implement
type DatabaseAdapter interface {
	Connect(connectionString string) error
	Close() error
	Create(ctx context.Context, entity interface{}) (interface{}, error)
	FindByID(ctx context.Context, entityType reflect.Type, id interface{}) (interface{}, error)
	FindOne(ctx context.Context, entityType reflect.Type, query Query) (interface{}, error)
	FindMany(ctx context.Context, entityType reflect.Type, query Query) (interface{}, error)
	Update(ctx context.Context, entity interface{}) (interface{}, error)
	Delete(ctx context.Context, entityType reflect.Type, query Query) error
	Count(ctx context.Context, entityType reflect.Type, query Query) (int64, error)
	Transaction(ctx context.Context, fn func(adapter DatabaseAdapter) error) error
	Migrate(entities ...interface{}) error
	GetType() DatabaseType
}

// DatabaseType represents the type of database
type DatabaseType string

const (
	DatabaseTypePostgreSQL DatabaseType = "postgresql"
	DatabaseTypeMySQL      DatabaseType = "mysql"
	DatabaseTypeSQLite     DatabaseType = "sqlite"
	DatabaseTypeMongoDB    DatabaseType = "mongodb"
	DatabaseTypeRedis      DatabaseType = "redis"
)

// DatabaseConfig represents database configuration
type DatabaseConfig struct {
	Type           DatabaseType
	ConnectionURL  string
	Host           string
	Port           int
	Username       string
	Password       string
	Database       string
	SSLMode        string
	MaxConnections int
	AutoMigrate    bool
	LogLevel       string
}

// AdapterFactory creates database adapters based on configuration
type AdapterFactory struct{}

// NewAdapterFactory creates a new adapter factory
func NewAdapterFactory() *AdapterFactory {
	return &AdapterFactory{}
}

// CreateAdapter creates a database adapter based on the configuration
func (f *AdapterFactory) CreateAdapter(config *DatabaseConfig) (DatabaseAdapter, error) {
	switch config.Type {
	case DatabaseTypePostgreSQL, DatabaseTypeMySQL, DatabaseTypeSQLite:
		return NewGORMAdapter(config)
	case DatabaseTypeMongoDB:
		return NewMongoAdapter(config)
	case DatabaseTypeRedis:
		return NewRedisAdapter(config)
	default:
		return nil, core.NewInternalServerException("Unsupported database type: "+string(config.Type), nil)
	}
}

// DetectDatabaseType detects database type from connection URL
func DetectDatabaseType(connectionURL string) DatabaseType {
	switch {
	case strings.HasPrefix(connectionURL, "postgresql://") || strings.HasPrefix(connectionURL, "postgres://"):
		return DatabaseTypePostgreSQL
	case strings.HasPrefix(connectionURL, "mysql://"):
		return DatabaseTypeMySQL
	case strings.HasPrefix(connectionURL, "sqlite://") || strings.HasSuffix(connectionURL, ".db") || strings.HasSuffix(connectionURL, ".sqlite"):
		return DatabaseTypeSQLite
	case strings.HasPrefix(connectionURL, "mongodb://") || strings.HasPrefix(connectionURL, "mongodb+srv://"):
		return DatabaseTypeMongoDB
	case strings.HasPrefix(connectionURL, "redis://"):
		return DatabaseTypeRedis
	default:
		return DatabaseTypePostgreSQL // Default to PostgreSQL
	}
}

// UniversalQueryBuilder provides database-agnostic query building
type UniversalQueryBuilder[T any] struct {
	adapter    DatabaseAdapter
	entityType reflect.Type
	query      *BaseQuery
}

// NewUniversalQueryBuilder creates a new universal query builder
func NewUniversalQueryBuilder[T any](adapter DatabaseAdapter, entityType reflect.Type) *UniversalQueryBuilder[T] {
	return &UniversalQueryBuilder[T]{
		adapter:    adapter,
		entityType: entityType,
		query:      NewBaseQuery(),
	}
}

// Where adds a WHERE condition
func (qb *UniversalQueryBuilder[T]) Where(field string, op Operator, value interface{}) QueryBuilder[T] {
	qb.query.conditions = append(qb.query.conditions, Condition{
		Field:    field,
		Operator: op,
		Value:    value,
	})
	return qb
}

// WhereIn adds a WHERE IN condition
func (qb *UniversalQueryBuilder[T]) WhereIn(field string, values []interface{}) QueryBuilder[T] {
	qb.query.conditions = append(qb.query.conditions, Condition{
		Field:    field,
		Operator: OpIn,
		Values:   values,
	})
	return qb
}

// WhereNotIn adds a WHERE NOT IN condition
func (qb *UniversalQueryBuilder[T]) WhereNotIn(field string, values []interface{}) QueryBuilder[T] {
	qb.query.conditions = append(qb.query.conditions, Condition{
		Field:    field,
		Operator: OpNotIn,
		Values:   values,
	})
	return qb
}

// WhereBetween adds a WHERE BETWEEN condition
func (qb *UniversalQueryBuilder[T]) WhereBetween(field string, min, max interface{}) QueryBuilder[T] {
	qb.query.conditions = append(qb.query.conditions, Condition{
		Field:    field,
		Operator: OpBetween,
		Values:   []interface{}{min, max},
	})
	return qb
}

// WhereNull adds a WHERE IS NULL condition
func (qb *UniversalQueryBuilder[T]) WhereNull(field string) QueryBuilder[T] {
	qb.query.conditions = append(qb.query.conditions, Condition{
		Field:    field,
		Operator: OpIsNull,
	})
	return qb
}

// WhereNotNull adds a WHERE IS NOT NULL condition
func (qb *UniversalQueryBuilder[T]) WhereNotNull(field string) QueryBuilder[T] {
	qb.query.conditions = append(qb.query.conditions, Condition{
		Field:    field,
		Operator: OpIsNotNull,
	})
	return qb
}

// OrderBy adds an ORDER BY clause
func (qb *UniversalQueryBuilder[T]) OrderBy(field string, direction Direction) QueryBuilder[T] {
	qb.query.orderBy = append(qb.query.orderBy, OrderByClause{
		Field:     field,
		Direction: direction,
	})
	return qb
}

// Limit sets the LIMIT
func (qb *UniversalQueryBuilder[T]) Limit(limit int) QueryBuilder[T] {
	qb.query.limit = limit
	return qb
}

// Offset sets the OFFSET
func (qb *UniversalQueryBuilder[T]) Offset(offset int) QueryBuilder[T] {
	qb.query.offset = offset
	return qb
}

// Join adds a JOIN clause (SQL databases only)
func (qb *UniversalQueryBuilder[T]) Join(table string, condition string) QueryBuilder[T] {
	// Only add joins for SQL databases
	if qb.adapter.GetType() != DatabaseTypeMongoDB {
		qb.query.joins = append(qb.query.joins, JoinClause{
			Table:     table,
			Condition: condition,
			Type:      JoinInner,
		})
	}
	return qb
}

// Select specifies which fields to select
func (qb *UniversalQueryBuilder[T]) Select(fields ...string) QueryBuilder[T] {
	qb.query.fields = append(qb.query.fields, fields...)
	return qb
}

// GroupBy adds GROUP BY fields
func (qb *UniversalQueryBuilder[T]) GroupBy(fields ...string) QueryBuilder[T] {
	qb.query.groupBy = append(qb.query.groupBy, fields...)
	return qb
}

// Having adds a HAVING clause
func (qb *UniversalQueryBuilder[T]) Having(condition string, args ...interface{}) QueryBuilder[T] {
	qb.query.having = append(qb.query.having, HavingClause{
		Condition: condition,
		Args:      args,
	})
	return qb
}

// Execute executes the query and returns results
func (qb *UniversalQueryBuilder[T]) Execute() ([]*T, error) {
	results, err := qb.adapter.FindMany(context.Background(), qb.entityType, qb.query)
	if err != nil {
		return nil, err
	}
	
	// Convert results to []*T
	return convertToTypedSlice[T](results)
}

// First executes the query and returns the first result
func (qb *UniversalQueryBuilder[T]) First() (*T, error) {
	// Set limit to 1
	qb.query.limit = 1
	
	result, err := qb.adapter.FindOne(context.Background(), qb.entityType, qb.query)
	if err != nil {
		return nil, err
	}
	
	// Convert result to *T
	return convertToTypedEntity[T](result)
}

// Count executes a count query
func (qb *UniversalQueryBuilder[T]) Count() (int64, error) {
	return qb.adapter.Count(context.Background(), qb.entityType, qb.query)
}

// Helper functions for type conversion
func convertToTypedSlice[T any](results interface{}) ([]*T, error) {
	// This would implement proper type conversion
	// For now, returning a placeholder
	return make([]*T, 0), nil
}

func convertToTypedEntity[T any](result interface{}) (*T, error) {
	// This would implement proper type conversion
	// For now, returning a placeholder
	var entity T
	return &entity, nil
}

// QueryTranslator translates universal queries to database-specific queries
type QueryTranslator interface {
	TranslateQuery(query Query) (string, []interface{}, error)
	TranslateCondition(condition Condition) (string, []interface{}, error)
}

// BaseQueryTranslator provides common query translation functionality
type BaseQueryTranslator struct {
	dbType DatabaseType
}

// NewBaseQueryTranslator creates a new base query translator
func NewBaseQueryTranslator(dbType DatabaseType) *BaseQueryTranslator {
	return &BaseQueryTranslator{
		dbType: dbType,
	}
}

// TranslateOperator translates a universal operator to database-specific operator
func (t *BaseQueryTranslator) TranslateOperator(op Operator) string {
	switch t.dbType {
	case DatabaseTypeMongoDB:
		return t.translateMongoOperator(op)
	default:
		return t.translateSQLOperator(op)
	}
}

// translateSQLOperator translates operators for SQL databases
func (t *BaseQueryTranslator) translateSQLOperator(op Operator) string {
	switch op {
	case OpEquals:
		return "="
	case OpNotEquals:
		return "!="
	case OpGreaterThan:
		return ">"
	case OpGreaterThanOrEqual:
		return ">="
	case OpLessThan:
		return "<"
	case OpLessThanOrEqual:
		return "<="
	case OpLike:
		return "LIKE"
	case OpNotLike:
		return "NOT LIKE"
	case OpIn:
		return "IN"
	case OpNotIn:
		return "NOT IN"
	case OpBetween:
		return "BETWEEN"
	case OpNotBetween:
		return "NOT BETWEEN"
	case OpIsNull:
		return "IS NULL"
	case OpIsNotNull:
		return "IS NOT NULL"
	default:
		return string(op)
	}
}

// translateMongoOperator translates operators for MongoDB
func (t *BaseQueryTranslator) translateMongoOperator(op Operator) string {
	switch op {
	case OpEquals:
		return "$eq"
	case OpNotEquals:
		return "$ne"
	case OpGreaterThan:
		return "$gt"
	case OpGreaterThanOrEqual:
		return "$gte"
	case OpLessThan:
		return "$lt"
	case OpLessThanOrEqual:
		return "$lte"
	case OpIn:
		return "$in"
	case OpNotIn:
		return "$nin"
	case OpRegex:
		return "$regex"
	case OpExists:
		return "$exists"
	default:
		return "$eq" // Default to equality
	}
}