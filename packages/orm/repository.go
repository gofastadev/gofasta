package orm

import (
	"context"
	"fmt"
	"reflect"
	"strings"
)

// Repository is the universal repository interface that works with any database
type Repository[T any] interface {
	Create(ctx context.Context, entity *T) (*T, error)
	FindByID(ctx context.Context, id interface{}) (*T, error)
	FindOne(ctx context.Context, query Query) (*T, error)
	FindMany(ctx context.Context, query Query) ([]*T, error)
	Update(ctx context.Context, entity *T) (*T, error)
	Delete(ctx context.Context, query Query) error
	Query() QueryBuilder[T]
	Transaction(ctx context.Context, fn func(repo Repository[T]) error) error
	Count(ctx context.Context, query Query) (int64, error)
}

// QueryBuilder provides a database-agnostic query building interface
type QueryBuilder[T any] interface {
	Where(field string, op Operator, value interface{}) QueryBuilder[T]
	WhereIn(field string, values []interface{}) QueryBuilder[T]
	WhereNotIn(field string, values []interface{}) QueryBuilder[T]
	WhereBetween(field string, min, max interface{}) QueryBuilder[T]
	WhereNull(field string) QueryBuilder[T]
	WhereNotNull(field string) QueryBuilder[T]
	OrderBy(field string, direction Direction) QueryBuilder[T]
	Limit(limit int) QueryBuilder[T]
	Offset(offset int) QueryBuilder[T]
	Join(table string, condition string) QueryBuilder[T] // SQL only
	Select(fields ...string) QueryBuilder[T]
	GroupBy(fields ...string) QueryBuilder[T]
	Having(condition string, args ...interface{}) QueryBuilder[T]
	Execute() ([]*T, error)
	First() (*T, error)
	Count() (int64, error)
}

// Query represents a database query
type Query interface {
	GetConditions() []Condition
	GetOrderBy() []OrderByClause
	GetLimit() int
	GetOffset() int
	GetJoins() []JoinClause
	GetFields() []string
	GetGroupBy() []string
	GetHaving() []HavingClause
}

// Condition represents a query condition
type Condition struct {
	Field    string
	Operator Operator
	Value    interface{}
	Values   []interface{} // For IN, NOT IN operations
}

// OrderByClause represents an ORDER BY clause
type OrderByClause struct {
	Field     string
	Direction Direction
}

// JoinClause represents a JOIN clause (SQL only)
type JoinClause struct {
	Table     string
	Condition string
	Type      JoinType
}

// HavingClause represents a HAVING clause
type HavingClause struct {
	Condition string
	Args      []interface{}
}

// Operator represents query operators
type Operator string

const (
	OpEquals              Operator = "="
	OpNotEquals           Operator = "!="
	OpGreaterThan         Operator = ">"
	OpGreaterThanOrEqual  Operator = ">="
	OpLessThan            Operator = "<"
	OpLessThanOrEqual     Operator = "<="
	OpLike                Operator = "LIKE"
	OpNotLike             Operator = "NOT LIKE"
	OpIn                  Operator = "IN"
	OpNotIn               Operator = "NOT IN"
	OpBetween             Operator = "BETWEEN"
	OpNotBetween          Operator = "NOT BETWEEN"
	OpIsNull              Operator = "IS NULL"
	OpIsNotNull           Operator = "IS NOT NULL"
	OpRegex               Operator = "REGEX" // MongoDB only
	OpExists              Operator = "EXISTS" // MongoDB only
)

// Direction represents sort direction
type Direction string

const (
	DirectionAsc  Direction = "ASC"
	DirectionDesc Direction = "DESC"
)

// JoinType represents SQL join types
type JoinType string

const (
	JoinInner JoinType = "INNER"
	JoinLeft  JoinType = "LEFT"
	JoinRight JoinType = "RIGHT"
	JoinFull  JoinType = "FULL"
)

// BaseQuery provides a default implementation of Query
type BaseQuery struct {
	conditions []Condition
	orderBy    []OrderByClause
	limit      int
	offset     int
	joins      []JoinClause
	fields     []string
	groupBy    []string
	having     []HavingClause
}

// NewBaseQuery creates a new BaseQuery
func NewBaseQuery() *BaseQuery {
	return &BaseQuery{
		conditions: make([]Condition, 0),
		orderBy:    make([]OrderByClause, 0),
		joins:      make([]JoinClause, 0),
		fields:     make([]string, 0),
		groupBy:    make([]string, 0),
		having:     make([]HavingClause, 0),
		limit:      -1,
		offset:     -1,
	}
}

// GetConditions implements Query interface
func (q *BaseQuery) GetConditions() []Condition {
	return q.conditions
}

// GetOrderBy implements Query interface
func (q *BaseQuery) GetOrderBy() []OrderByClause {
	return q.orderBy
}

// GetLimit implements Query interface
func (q *BaseQuery) GetLimit() int {
	return q.limit
}

// GetOffset implements Query interface
func (q *BaseQuery) GetOffset() int {
	return q.offset
}

// GetJoins implements Query interface
func (q *BaseQuery) GetJoins() []JoinClause {
	return q.joins
}

// GetFields implements Query interface
func (q *BaseQuery) GetFields() []string {
	return q.fields
}

// GetGroupBy implements Query interface
func (q *BaseQuery) GetGroupBy() []string {
	return q.groupBy
}

// GetHaving implements Query interface
func (q *BaseQuery) GetHaving() []HavingClause {
	return q.having
}

// BaseRepository provides common repository functionality
type BaseRepository[T any] struct {
	dbAdapter DatabaseAdapter
	entityType reflect.Type
}

// NewBaseRepository creates a new BaseRepository
func NewBaseRepository[T any](dbAdapter DatabaseAdapter) *BaseRepository[T] {
	var entity T
	entityType := reflect.TypeOf(entity)
	if entityType.Kind() == reflect.Ptr {
		entityType = entityType.Elem()
	}

	return &BaseRepository[T]{
		dbAdapter:  dbAdapter,
		entityType: entityType,
	}
}

// Create implements Repository interface
func (r *BaseRepository[T]) Create(ctx context.Context, entity *T) (*T, error) {
	result, err := r.dbAdapter.Create(ctx, entity)
	if err != nil {
		return nil, err
	}
	if typedResult, ok := result.(*T); ok {
		return typedResult, nil
	}
	return entity, nil // Return original entity if type assertion fails
}

// FindByID implements Repository interface
func (r *BaseRepository[T]) FindByID(ctx context.Context, id interface{}) (*T, error) {
	result, err := r.dbAdapter.FindByID(ctx, r.entityType, id)
	if err != nil {
		return nil, err
	}
	if typedResult, ok := result.(*T); ok {
		return typedResult, nil
	}
	return nil, fmt.Errorf("type assertion failed")
}

// FindOne implements Repository interface
func (r *BaseRepository[T]) FindOne(ctx context.Context, query Query) (*T, error) {
	result, err := r.dbAdapter.FindOne(ctx, r.entityType, query)
	if err != nil {
		return nil, err
	}
	if typedResult, ok := result.(*T); ok {
		return typedResult, nil
	}
	return nil, fmt.Errorf("type assertion failed")
}

// FindMany implements Repository interface
func (r *BaseRepository[T]) FindMany(ctx context.Context, query Query) ([]*T, error) {
	result, err := r.dbAdapter.FindMany(ctx, r.entityType, query)
	if err != nil {
		return nil, err
	}
	if typedResult, ok := result.([]*T); ok {
		return typedResult, nil
	}
	return nil, fmt.Errorf("type assertion failed")
}

// Update implements Repository interface
func (r *BaseRepository[T]) Update(ctx context.Context, entity *T) (*T, error) {
	result, err := r.dbAdapter.Update(ctx, entity)
	if err != nil {
		return nil, err
	}
	if typedResult, ok := result.(*T); ok {
		return typedResult, nil
	}
	return entity, nil // Return original entity if type assertion fails
}

// Delete implements Repository interface
func (r *BaseRepository[T]) Delete(ctx context.Context, query Query) error {
	return r.dbAdapter.Delete(ctx, r.entityType, query)
}

// Query implements Repository interface
func (r *BaseRepository[T]) Query() QueryBuilder[T] {
	return NewUniversalQueryBuilder[T](r.dbAdapter, r.entityType)
}

// Transaction implements Repository interface
func (r *BaseRepository[T]) Transaction(ctx context.Context, fn func(repo Repository[T]) error) error {
	return r.dbAdapter.Transaction(ctx, func(adapter DatabaseAdapter) error {
		txRepo := &BaseRepository[T]{
			dbAdapter:  adapter,
			entityType: r.entityType,
		}
		return fn(txRepo)
	})
}

// Count implements Repository interface
func (r *BaseRepository[T]) Count(ctx context.Context, query Query) (int64, error) {
	return r.dbAdapter.Count(ctx, r.entityType, query)
}

// RepositoryFactory creates repositories for different entity types
type RepositoryFactory struct {
	dbAdapter DatabaseAdapter
}

// NewRepositoryFactory creates a new repository factory
func NewRepositoryFactory(dbAdapter DatabaseAdapter) *RepositoryFactory {
	return &RepositoryFactory{
		dbAdapter: dbAdapter,
	}
}

// GetRepository returns a repository for the specified entity type
func (f *RepositoryFactory) GetRepository(entityType reflect.Type) interface{} {
	// This would return a properly typed repository
	// Implementation would use reflection to create the correct generic type
	return &BaseRepository[interface{}]{
		dbAdapter:  f.dbAdapter,
		entityType: entityType,
	}
}

// EntityMetadata represents metadata about an entity
type EntityMetadata struct {
	TableName    string
	PrimaryKey   string
	Fields       map[string]FieldMetadata
	Relationships map[string]RelationshipMetadata
}

// FieldMetadata represents metadata about an entity field
type FieldMetadata struct {
	DBName     string
	Type       reflect.Type
	Required   bool
	Unique     bool
	Index      bool
	Default    interface{}
	Validators []string
}

// RelationshipMetadata represents metadata about entity relationships
type RelationshipMetadata struct {
	Type         RelationshipType
	TargetEntity reflect.Type
	ForeignKey   string
	References   string
}

// RelationshipType represents the type of relationship
type RelationshipType string

const (
	HasOne     RelationshipType = "has_one"
	HasMany    RelationshipType = "has_many"
	BelongsTo  RelationshipType = "belongs_to"
	ManyToMany RelationshipType = "many_to_many"
)

// ExtractEntityMetadata extracts metadata from an entity struct
func ExtractEntityMetadata(entity interface{}) *EntityMetadata {
	entityType := reflect.TypeOf(entity)
	if entityType.Kind() == reflect.Ptr {
		entityType = entityType.Elem()
	}

	metadata := &EntityMetadata{
		TableName:     getTableName(entityType),
		Fields:        make(map[string]FieldMetadata),
		Relationships: make(map[string]RelationshipMetadata),
	}

	for i := 0; i < entityType.NumField(); i++ {
		field := entityType.Field(i)
		
		// Extract field metadata
		if fieldMeta := extractFieldMetadata(field); fieldMeta != nil {
			metadata.Fields[field.Name] = *fieldMeta
			
			// Check for primary key
			if fieldMeta.DBName == "id" || containsTag(field, "gofasta", "primary_key") {
				metadata.PrimaryKey = field.Name
			}
		}
		
		// Extract relationship metadata
		if relMeta := extractRelationshipMetadata(field); relMeta != nil {
			metadata.Relationships[field.Name] = *relMeta
		}
	}

	return metadata
}

// Helper functions for metadata extraction
func getTableName(entityType reflect.Type) string {
	// Check for custom table name in struct tag (structs don't have tags, only fields do)
	// So we'll just use the default naming convention
	
	// Default to lowercase struct name with 's' suffix
	return strings.ToLower(entityType.Name()) + "s"
}

func extractFieldMetadata(field reflect.StructField) *FieldMetadata {
	// Skip unexported fields
	if !field.IsExported() {
		return nil
	}

	dbName := getDBFieldName(field)
	if dbName == "-" {
		return nil
	}

	metadata := &FieldMetadata{
		DBName:     dbName,
		Type:       field.Type,
		Required:   containsTag(field, "gofasta", "required"),
		Unique:     containsTag(field, "gofasta", "unique"),
		Index:      containsTag(field, "gofasta", "index"),
		Validators: extractValidators(field),
	}

	return metadata
}

func extractRelationshipMetadata(field reflect.StructField) *RelationshipMetadata {
	gofastaTag := field.Tag.Get("gofasta")
	if gofastaTag == "" {
		return nil
	}

	var relType RelationshipType
	if strings.Contains(gofastaTag, "has_one") {
		relType = HasOne
	} else if strings.Contains(gofastaTag, "has_many") {
		relType = HasMany
	} else if strings.Contains(gofastaTag, "belongs_to") {
		relType = BelongsTo
	} else if strings.Contains(gofastaTag, "many_to_many") {
		relType = ManyToMany
	} else {
		return nil
	}

	targetType := field.Type
	if targetType.Kind() == reflect.Slice {
		targetType = targetType.Elem()
	}
	if targetType.Kind() == reflect.Ptr {
		targetType = targetType.Elem()
	}

	return &RelationshipMetadata{
		Type:         relType,
		TargetEntity: targetType,
		ForeignKey:   extractForeignKey(field),
		References:   extractReferences(field),
	}
}

func getDBFieldName(field reflect.StructField) string {
	// Check gorm tag first
	if gormTag := field.Tag.Get("gorm"); gormTag != "" {
		if strings.Contains(gormTag, "column:") {
			parts := strings.Split(gormTag, ";")
			for _, part := range parts {
				if strings.HasPrefix(part, "column:") {
					return strings.TrimPrefix(part, "column:")
				}
			}
		}
	}

	// Check bson tag for MongoDB
	if bsonTag := field.Tag.Get("bson"); bsonTag != "" {
		parts := strings.Split(bsonTag, ",")
		if len(parts) > 0 && parts[0] != "" {
			return parts[0]
		}
	}

	// Check json tag as fallback
	if jsonTag := field.Tag.Get("json"); jsonTag != "" {
		parts := strings.Split(jsonTag, ",")
		if len(parts) > 0 && parts[0] != "" && parts[0] != "-" {
			return parts[0]
		}
	}

	// Default to field name in snake_case
	return toSnakeCase(field.Name)
}

func containsTag(field reflect.StructField, tagName, value string) bool {
	tag := field.Tag.Get(tagName)
	return strings.Contains(tag, value)
}

func extractValidators(field reflect.StructField) []string {
	validators := make([]string, 0)
	
	if validateTag := field.Tag.Get("validate"); validateTag != "" {
		parts := strings.Split(validateTag, ",")
		for _, part := range parts {
			validators = append(validators, strings.TrimSpace(part))
		}
	}
	
	return validators
}

func extractForeignKey(field reflect.StructField) string {
	gormTag := field.Tag.Get("gorm")
	if strings.Contains(gormTag, "foreignKey:") {
		parts := strings.Split(gormTag, ";")
		for _, part := range parts {
			if strings.HasPrefix(part, "foreignKey:") {
				return strings.TrimPrefix(part, "foreignKey:")
			}
		}
	}
	return ""
}

func extractReferences(field reflect.StructField) string {
	gormTag := field.Tag.Get("gorm")
	if strings.Contains(gormTag, "references:") {
		parts := strings.Split(gormTag, ";")
		for _, part := range parts {
			if strings.HasPrefix(part, "references:") {
				return strings.TrimPrefix(part, "references:")
			}
		}
	}
	return "ID"
}

func toSnakeCase(str string) string {
	var result []rune
	for i, r := range str {
		if i > 0 && 'A' <= r && r <= 'Z' {
			result = append(result, '_')
		}
		result = append(result, rune(strings.ToLower(string(r))[0]))
	}
	return string(result)
}