package orm

import (
	"context"
	"reflect"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/healtronlabs/gofasta/packages/core"
)

// MongoAdapter implements DatabaseAdapter using MongoDB driver
type MongoAdapter struct {
	client     *mongo.Client
	database   *mongo.Database
	config     *DatabaseConfig
	dbName     string
}

// NewMongoAdapter creates a new MongoDB adapter
func NewMongoAdapter(config *DatabaseConfig) (*MongoAdapter, error) {
	adapter := &MongoAdapter{
		config: config,
	}

	if err := adapter.Connect(config.ConnectionURL); err != nil {
		return nil, err
	}

	return adapter, nil
}

// Connect establishes a connection to MongoDB
func (a *MongoAdapter) Connect(connectionString string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOptions := options.Client().ApplyURI(connectionString)
	
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return core.NewInternalServerException("Failed to connect to MongoDB", err)
	}

	// Test the connection
	if err := client.Ping(ctx, nil); err != nil {
		return core.NewInternalServerException("Failed to ping MongoDB", err)
	}

	a.client = client
	
	// Extract database name from connection string
	a.dbName = a.extractDatabaseName(connectionString)
	if a.dbName == "" {
		a.dbName = a.config.Database
	}
	if a.dbName == "" {
		a.dbName = "gofasta" // Default database name
	}
	
	a.database = client.Database(a.dbName)

	return nil
}

// Close closes the MongoDB connection
func (a *MongoAdapter) Close() error {
	if a.client != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return a.client.Disconnect(ctx)
	}
	return nil
}

// Create creates a new document
func (a *MongoAdapter) Create(ctx context.Context, entity interface{}) (interface{}, error) {
	collection := a.getCollection(entity)
	
	result, err := collection.InsertOne(ctx, entity)
	if err != nil {
		return nil, a.translateError(err)
	}

	// Set the ID if it's an ObjectID
	if oid, ok := result.InsertedID.(primitive.ObjectID); ok {
		a.setEntityID(entity, oid)
	}

	return entity, nil
}

// FindByID finds a document by its ID
func (a *MongoAdapter) FindByID(ctx context.Context, entityType reflect.Type, id interface{}) (interface{}, error) {
	entity := reflect.New(entityType).Interface()
	collection := a.getCollectionByType(entityType)

	// Convert ID to ObjectID if it's a string
	mongoID := a.convertToMongoID(id)
	
	filter := bson.M{"_id": mongoID}
	err := collection.FindOne(ctx, filter).Decode(entity)
	
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, core.NewNotFoundException("Document", "not found")
		}
		return nil, a.translateError(err)
	}

	return entity, nil
}

// FindOne finds a single document matching the query
func (a *MongoAdapter) FindOne(ctx context.Context, entityType reflect.Type, query Query) (interface{}, error) {
	entity := reflect.New(entityType).Interface()
	collection := a.getCollectionByType(entityType)

	filter := a.buildMongoFilter(query)
	opts := a.buildMongoOptions(query)

	err := collection.FindOne(ctx, filter, opts).Decode(entity)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, core.NewNotFoundException("Document", "not found")
		}
		return nil, a.translateError(err)
	}

	return entity, nil
}

// FindMany finds multiple documents matching the query
func (a *MongoAdapter) FindMany(ctx context.Context, entityType reflect.Type, query Query) (interface{}, error) {
	collection := a.getCollectionByType(entityType)

	filter := a.buildMongoFilter(query)
	opts := a.buildMongoFindOptions(query)

	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, a.translateError(err)
	}
	defer cursor.Close(ctx)

	// Create a slice to hold results
	sliceType := reflect.SliceOf(reflect.PtrTo(entityType))
	results := reflect.New(sliceType).Elem()

	for cursor.Next(ctx) {
		entity := reflect.New(entityType).Interface()
		if err := cursor.Decode(entity); err != nil {
			return nil, a.translateError(err)
		}
		results = reflect.Append(results, reflect.ValueOf(entity))
	}

	if err := cursor.Err(); err != nil {
		return nil, a.translateError(err)
	}

	return results.Interface(), nil
}

// Update updates a document
func (a *MongoAdapter) Update(ctx context.Context, entity interface{}) (interface{}, error) {
	collection := a.getCollection(entity)
	
	// Get the ID from the entity
	id := a.getEntityID(entity)
	if id == nil {
		return nil, core.NewBadRequestException("Entity ID is required for update")
	}

	filter := bson.M{"_id": id}

	result, err := collection.ReplaceOne(ctx, filter, entity)
	if err != nil {
		return nil, a.translateError(err)
	}

	if result.MatchedCount == 0 {
		return nil, core.NewNotFoundException("Document", "not found")
	}

	return entity, nil
}

// Delete deletes documents matching the query
func (a *MongoAdapter) Delete(ctx context.Context, entityType reflect.Type, query Query) error {
	collection := a.getCollectionByType(entityType)

	filter := a.buildMongoFilter(query)
	
	result, err := collection.DeleteMany(ctx, filter)
	if err != nil {
		return a.translateError(err)
	}

	if result.DeletedCount == 0 {
		return core.NewNotFoundException("Documents", "no documents found to delete")
	}

	return nil
}

// Count counts documents matching the query
func (a *MongoAdapter) Count(ctx context.Context, entityType reflect.Type, query Query) (int64, error) {
	collection := a.getCollectionByType(entityType)

	filter := a.buildMongoFilter(query)
	
	count, err := collection.CountDocuments(ctx, filter)
	if err != nil {
		return 0, a.translateError(err)
	}

	return count, nil
}

// Transaction executes a function within a MongoDB transaction
func (a *MongoAdapter) Transaction(ctx context.Context, fn func(adapter DatabaseAdapter) error) error {
	session, err := a.client.StartSession()
	if err != nil {
		return a.translateError(err)
	}
	defer session.EndSession(ctx)

	return mongo.WithSession(ctx, session, func(sc mongo.SessionContext) error {
		if err := session.StartTransaction(); err != nil {
			return a.translateError(err)
		}

		// Create a new adapter with the same client but in transaction context
		txAdapter := &MongoAdapter{
			client:   a.client,
			database: a.database,
			config:   a.config,
			dbName:   a.dbName,
		}

		if err := fn(txAdapter); err != nil {
			session.AbortTransaction(sc)
			return err
		}

		return session.CommitTransaction(sc)
	})
}

// Migrate performs collection setup (MongoDB doesn't need migrations like SQL)
func (a *MongoAdapter) Migrate(entities ...interface{}) error {
	// MongoDB is schemaless, but we can create indexes here
	for _, entity := range entities {
		if err := a.createIndexes(entity); err != nil {
			return err
		}
	}
	return nil
}

// GetType returns the database type
func (a *MongoAdapter) GetType() DatabaseType {
	return DatabaseTypeMongoDB
}

// buildMongoFilter builds a MongoDB filter from the universal Query
func (a *MongoAdapter) buildMongoFilter(query Query) bson.M {
	filter := bson.M{}

	for _, condition := range query.GetConditions() {
		fieldFilter := a.buildConditionFilter(condition)
		
		// Merge filters for the same field
		if existing, exists := filter[condition.Field]; exists {
			if existingMap, ok := existing.(bson.M); ok {
				if conditionMap, ok := fieldFilter.(bson.M); ok {
					for k, v := range conditionMap {
						existingMap[k] = v
					}
				}
			}
		} else {
			filter[condition.Field] = fieldFilter
		}
	}

	return filter
}

// buildConditionFilter builds a MongoDB filter for a single condition
func (a *MongoAdapter) buildConditionFilter(condition Condition) interface{} {
	switch condition.Operator {
	case OpEquals:
		return condition.Value
	case OpNotEquals:
		return bson.M{"$ne": condition.Value}
	case OpGreaterThan:
		return bson.M{"$gt": condition.Value}
	case OpGreaterThanOrEqual:
		return bson.M{"$gte": condition.Value}
	case OpLessThan:
		return bson.M{"$lt": condition.Value}
	case OpLessThanOrEqual:
		return bson.M{"$lte": condition.Value}
	case OpIn:
		return bson.M{"$in": condition.Values}
	case OpNotIn:
		return bson.M{"$nin": condition.Values}
	case OpBetween:
		if len(condition.Values) >= 2 {
			return bson.M{"$gte": condition.Values[0], "$lte": condition.Values[1]}
		}
		return condition.Value
	case OpRegex:
		return bson.M{"$regex": condition.Value}
	case OpExists:
		return bson.M{"$exists": condition.Value}
	case OpIsNull:
		return nil
	case OpIsNotNull:
		return bson.M{"$ne": nil}
	default:
		return condition.Value
	}
}

// buildMongoOptions builds MongoDB find options
func (a *MongoAdapter) buildMongoOptions(query Query) *options.FindOneOptions {
	opts := options.FindOne()

	// Apply field selection
	if fields := query.GetFields(); len(fields) > 0 {
		projection := bson.M{}
		for _, field := range fields {
			projection[field] = 1
		}
		opts.SetProjection(projection)
	}

	// Apply sorting
	if orderBy := query.GetOrderBy(); len(orderBy) > 0 {
		sort := bson.M{}
		for _, order := range orderBy {
			direction := 1 // ASC
			if order.Direction == DirectionDesc {
				direction = -1
			}
			sort[order.Field] = direction
		}
		opts.SetSort(sort)
	}

	return opts
}

// buildMongoFindOptions builds MongoDB find options for multiple documents
func (a *MongoAdapter) buildMongoFindOptions(query Query) *options.FindOptions {
	opts := options.Find()

	// Apply field selection
	if fields := query.GetFields(); len(fields) > 0 {
		projection := bson.M{}
		for _, field := range fields {
			projection[field] = 1
		}
		opts.SetProjection(projection)
	}

	// Apply sorting
	if orderBy := query.GetOrderBy(); len(orderBy) > 0 {
		sort := bson.M{}
		for _, order := range orderBy {
			direction := 1 // ASC
			if order.Direction == DirectionDesc {
				direction = -1
			}
			sort[order.Field] = direction
		}
		opts.SetSort(sort)
	}

	// Apply limit
	if limit := query.GetLimit(); limit > 0 {
		opts.SetLimit(int64(limit))
	}

	// Apply offset
	if offset := query.GetOffset(); offset > 0 {
		opts.SetSkip(int64(offset))
	}

	return opts
}

// Helper functions
func (a *MongoAdapter) getCollection(entity interface{}) *mongo.Collection {
	entityType := reflect.TypeOf(entity)
	if entityType.Kind() == reflect.Ptr {
		entityType = entityType.Elem()
	}
	return a.getCollectionByType(entityType)
}

func (a *MongoAdapter) getCollectionByType(entityType reflect.Type) *mongo.Collection {
	collectionName := a.getCollectionName(entityType)
	return a.database.Collection(collectionName)
}

func (a *MongoAdapter) getCollectionName(entityType reflect.Type) string {
	// For now, just use lowercase struct name as collection name
	// TODO: Implement proper struct tag parsing for collection names if needed
	return strings.ToLower(entityType.Name())
}

func (a *MongoAdapter) convertToMongoID(id interface{}) interface{} {
	switch v := id.(type) {
	case string:
		if oid, err := primitive.ObjectIDFromHex(v); err == nil {
			return oid
		}
		return v
	case primitive.ObjectID:
		return v
	default:
		return id
	}
}

func (a *MongoAdapter) getEntityID(entity interface{}) interface{} {
	entityValue := reflect.ValueOf(entity)
	if entityValue.Kind() == reflect.Ptr {
		entityValue = entityValue.Elem()
	}

	// Look for ID field
	idField := entityValue.FieldByName("ID")
	if idField.IsValid() && !idField.IsZero() {
		return idField.Interface()
	}

	return nil
}

func (a *MongoAdapter) setEntityID(entity interface{}, id primitive.ObjectID) {
	entityValue := reflect.ValueOf(entity)
	if entityValue.Kind() == reflect.Ptr {
		entityValue = entityValue.Elem()
	}

	idField := entityValue.FieldByName("ID")
	if idField.IsValid() && idField.CanSet() {
		if idField.Type() == reflect.TypeOf(id) {
			idField.Set(reflect.ValueOf(id))
		} else if idField.Type() == reflect.TypeOf("") {
			idField.SetString(id.Hex())
		}
	}
}

func (a *MongoAdapter) extractDatabaseName(connectionString string) string {
	// Extract database name from MongoDB connection string
	// This is a simplified implementation
	parts := strings.Split(connectionString, "/")
	if len(parts) > 3 {
		dbPart := parts[3]
		if idx := strings.Index(dbPart, "?"); idx != -1 {
			return dbPart[:idx]
		}
		return dbPart
	}
	return ""
}

func (a *MongoAdapter) createIndexes(entity interface{}) error {
	// This would analyze the entity struct and create appropriate indexes
	// For now, just return nil
	return nil
}

func (a *MongoAdapter) translateError(err error) error {
	if err == nil {
		return nil
	}

	errStr := err.Error()
	
	if err == mongo.ErrNoDocuments {
		return core.NewNotFoundException("Document", "not found")
	}
	
	if strings.Contains(errStr, "duplicate") || strings.Contains(errStr, "E11000") {
		return core.NewConflictException("Duplicate entry")
	}
	
	if strings.Contains(errStr, "timeout") {
		return core.NewInternalServerException("Database timeout", err)
	}
	
	return core.NewInternalServerException("Database error", err)
}