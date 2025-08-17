package repository

import (
	"context"
)

// Operator defines query operators
type Operator string

const (
	OpEquals      Operator = "eq"
	OpNotEquals   Operator = "ne"
	OpGreaterThan Operator = "gt"
	OpLessThan    Operator = "lt"
	OpGreaterEq   Operator = "gte"
	OpLessEq      Operator = "lte"
	OpIn          Operator = "in"
	OpNotIn       Operator = "nin"
	OpLike        Operator = "like"
	OpRegex       Operator = "regex"
)

// Direction defines sort direction
type Direction string

const (
	DirectionAsc  Direction = "asc"
	DirectionDesc Direction = "desc"
)

// Repository defines the universal repository interface
type Repository[T any] interface {
	Create(ctx context.Context, entity *T) (*T, error)
	FindByID(ctx context.Context, id interface{}) (*T, error)
	FindOne(ctx context.Context, query Query) (*T, error)
	FindMany(ctx context.Context, query Query) ([]*T, error)
	Update(ctx context.Context, entity *T) (*T, error)
	UpdateWhere(ctx context.Context, query Query, updates map[string]interface{}) error
	Delete(ctx context.Context, query Query) error
	Count(ctx context.Context, query Query) (int64, error)
	Query() QueryBuilder[T]
	Transaction(ctx context.Context, fn func(repo Repository[T]) error) error
}

// QueryBuilder defines the universal query builder interface
type QueryBuilder[T any] interface {
	Where(field string, op Operator, value interface{}) QueryBuilder[T]
	WhereIn(field string, values []interface{}) QueryBuilder[T]
	OrderBy(field string, direction Direction) QueryBuilder[T]
	Limit(limit int) QueryBuilder[T]
	Offset(offset int) QueryBuilder[T]
	Select(fields ...string) QueryBuilder[T]
	Join(table string, condition string) QueryBuilder[T] // SQL only
	GroupBy(fields ...string) QueryBuilder[T]
	Having(condition string) QueryBuilder[T]
	Execute() ([]*T, error)
	First() (*T, error)
	Count() (int64, error)
}

// Query represents a database query
type Query interface {
	GetConditions() []QueryCondition
	GetSorts() []SortCondition
	GetLimit() int
	GetOffset() int
	GetFields() []string
}

// QueryCondition represents a query condition
type QueryCondition struct {
	Field    string
	Operator Operator
	Value    interface{}
}

// SortCondition represents a sort condition
type SortCondition struct {
	Field     string
	Direction Direction
}

// BaseQuery provides a default implementation of Query
type BaseQuery struct {
	conditions []QueryCondition
	sorts      []SortCondition
	limit      int
	offset     int
	fields     []string
}

// NewQuery creates a new query
func NewQuery() *BaseQuery {
	return &BaseQuery{
		conditions: make([]QueryCondition, 0),
		sorts:      make([]SortCondition, 0),
		fields:     make([]string, 0),
		limit:      -1,
		offset:     0,
	}
}

// GetConditions returns query conditions
func (q *BaseQuery) GetConditions() []QueryCondition {
	return q.conditions
}

// GetSorts returns sort conditions
func (q *BaseQuery) GetSorts() []SortCondition {
	return q.sorts
}

// GetLimit returns the limit
func (q *BaseQuery) GetLimit() int {
	return q.limit
}

// GetOffset returns the offset
func (q *BaseQuery) GetOffset() int {
	return q.offset
}

// GetFields returns selected fields
func (q *BaseQuery) GetFields() []string {
	return q.fields
}

// Where adds a condition to the query
func (q *BaseQuery) Where(field string, op Operator, value interface{}) *BaseQuery {
	q.conditions = append(q.conditions, QueryCondition{
		Field:    field,
		Operator: op,
		Value:    value,
	})
	return q
}