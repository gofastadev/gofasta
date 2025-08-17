package orm

import (
	"fmt"

	"github.com/healtronlabs/gofasta/packages/core"
)

// GofastaOrmModule provides unified database abstraction
type GofastaOrmModule struct {
	*core.BaseModule
	config   *DatabaseConfig
	adapter  DatabaseAdapter
	factory  *RepositoryFactory
	entities []interface{}
}

// NewGofastaOrmModule creates a new Gofasta ORM module
func NewGofastaOrmModule(config *DatabaseConfig) *GofastaOrmModule {
	return &GofastaOrmModule{
		BaseModule: core.NewBaseModule(),
		config:     config,
		entities:   make([]interface{}, 0),
	}
}

// NewGofastaOrmModuleFromURL creates a new Gofasta ORM module from connection URL
func NewGofastaOrmModuleFromURL(connectionURL string) *GofastaOrmModule {
	config := &DatabaseConfig{
		Type:          DetectDatabaseType(connectionURL),
		ConnectionURL: connectionURL,
		AutoMigrate:   true,
		LogLevel:      "warn",
	}

	return NewGofastaOrmModule(config)
}

// Configure configures the ORM module
func (m *GofastaOrmModule) Configure(container *core.DIContainer) error {
	// Create the appropriate database adapter
	adapterFactory := NewAdapterFactory()
	adapter, err := adapterFactory.CreateAdapter(m.config)
	if err != nil {
		return fmt.Errorf("failed to create database adapter: %w", err)
	}

	m.adapter = adapter

	// Create repository factory
	m.factory = NewRepositoryFactory(adapter)

	// Register the adapter and factory as providers
	if err := container.RegisterProvider(adapter); err != nil {
		return fmt.Errorf("failed to register database adapter: %w", err)
	}

	if err := container.RegisterProvider(m.factory); err != nil {
		return fmt.Errorf("failed to register repository factory: %w", err)
	}

	// Perform auto-migration if enabled
	if m.config.AutoMigrate && len(m.entities) > 0 {
		if err := adapter.Migrate(m.entities...); err != nil {
			return fmt.Errorf("failed to auto-migrate entities: %w", err)
		}
	}

	return nil
}

// AddEntity adds an entity to be managed by the ORM
func (m *GofastaOrmModule) AddEntity(entity interface{}) *GofastaOrmModule {
	m.entities = append(m.entities, entity)
	return m
}

// AddEntities adds multiple entities to be managed by the ORM
func (m *GofastaOrmModule) AddEntities(entities ...interface{}) *GofastaOrmModule {
	m.entities = append(m.entities, entities...)
	return m
}

// GetAdapter returns the database adapter
func (m *GofastaOrmModule) GetAdapter() DatabaseAdapter {
	return m.adapter
}

// GetFactory returns the repository factory
func (m *GofastaOrmModule) GetFactory() *RepositoryFactory {
	return m.factory
}

// WithConfig sets the database configuration
func (m *GofastaOrmModule) WithConfig(config *DatabaseConfig) *GofastaOrmModule {
	m.config = config
	return m
}

// WithAutoMigrate enables or disables auto-migration
func (m *GofastaOrmModule) WithAutoMigrate(enable bool) *GofastaOrmModule {
	m.config.AutoMigrate = enable
	return m
}

// WithLogLevel sets the log level
func (m *GofastaOrmModule) WithLogLevel(level string) *GofastaOrmModule {
	m.config.LogLevel = level
	return m
}

// WithMaxConnections sets the maximum number of connections
func (m *GofastaOrmModule) WithMaxConnections(max int) *GofastaOrmModule {
	m.config.MaxConnections = max
	return m
}

// RepositoryProvider provides repositories through dependency injection
type RepositoryProvider[T any] struct {
	repository Repository[T]
}

// NewRepositoryProvider creates a new repository provider
func NewRepositoryProvider[T any](factory *RepositoryFactory) *RepositoryProvider[T] {
	repo := NewBaseRepository[T](factory.dbAdapter)
	return &RepositoryProvider[T]{
		repository: repo,
	}
}

// GetRepository returns the repository
func (p *RepositoryProvider[T]) GetRepository() Repository[T] {
	return p.repository
}

// RegisterRepository registers a repository for dependency injection
func RegisterRepository[T any](container *core.DIContainer, factory *RepositoryFactory) error {
	provider := NewRepositoryProvider[T](factory)
	return container.RegisterProvider(provider)
}

// Utility functions for common database operations

// WithDatabase creates a module configuration for a specific database
func WithDatabase(dbType DatabaseType, connectionURL string) *DatabaseConfig {
	return &DatabaseConfig{
		Type:          dbType,
		ConnectionURL: connectionURL,
		AutoMigrate:   true,
		LogLevel:      "warn",
	}
}

// WithPostgreSQL creates a PostgreSQL configuration
func WithPostgreSQL(connectionURL string) *DatabaseConfig {
	return WithDatabase(DatabaseTypePostgreSQL, connectionURL)
}

// WithMySQL creates a MySQL configuration
func WithMySQL(connectionURL string) *DatabaseConfig {
	return WithDatabase(DatabaseTypeMySQL, connectionURL)
}

// WithSQLite creates a SQLite configuration
func WithSQLite(connectionURL string) *DatabaseConfig {
	return WithDatabase(DatabaseTypeSQLite, connectionURL)
}

// WithMongoDB creates a MongoDB configuration
func WithMongoDB(connectionURL string) *DatabaseConfig {
	return WithDatabase(DatabaseTypeMongoDB, connectionURL)
}

// Example usage patterns for different databases:

/*
// PostgreSQL example
module := NewGofastaOrmModule(WithPostgreSQL("postgresql://user:pass@localhost:5432/myapp"))
    .AddEntities(&User{}, &Order{}, &Product{})
    .WithAutoMigrate(true)
    .WithLogLevel("info")

// MongoDB example
module := NewGofastaOrmModule(WithMongoDB("mongodb://localhost:27017/myapp"))
    .AddEntities(&User{}, &Order{}, &Product{})

// Multi-database example
app := core.CreateApp(&AppModule{})
app.RegisterModule(NewGofastaOrmModule(WithPostgreSQL("postgresql://localhost:5432/primary")))
app.RegisterModule(NewGofastaOrmModule(WithMongoDB("mongodb://localhost:27017/analytics")))

// Same service code works with any database
type UserService struct {
    UserRepo Repository[User] `inject:""`
}

func (s *UserService) FindActiveUsers() ([]*User, error) {
    return s.UserRepo.Query().
        Where("status", OpEquals, "active").
        Where("age", OpGreaterThan, 18).
        OrderBy("created_at", DirectionDesc).
        Limit(10).
        Execute()
}
*/
