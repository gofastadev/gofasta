# @gofasta/orm

Unified ORM module for the Gofasta framework solving Go's database fragmentation problem with a single API for SQL and NoSQL databases.

## Features

- **Database-Agnostic API**: Same code works with PostgreSQL, MongoDB, MySQL, SQLite
- **Battle-Tested Foundation**: Uses GORM for SQL databases and mongo-driver for MongoDB
- **Type-Safe Repository Pattern**: Full Go generics support
- **Intelligent Query Builder**: Translates to database-specific queries
- **Seamless Migration**: Switch databases without code changes
- **Advanced Relationships**: Works with both SQL joins and MongoDB references
- **Transaction Support**: Unified transaction API across databases

## Supported Databases

- **SQL Databases** (via GORM):
  - PostgreSQL
  - MySQL
  - SQLite
  - SQL Server
- **NoSQL Databases**:
  - MongoDB
  - (Future: Redis, DynamoDB)

## Installation

```bash
go get github.com/healtronlabs/gofasta/packages/orm
```

## Quick Start

```go
package main

import (
    "context"
    "github.com/healtronlabs/gofasta/packages/orm"
)

// Universal model definition - works with any database
type User struct {
    ID        string    `gorm:"primaryKey" bson:"_id,omitempty" gofasta:"primary_key"`
    Email     string    `gorm:"uniqueIndex" bson:"email" gofasta:"unique,required"`
    FirstName string    `gorm:"not null" bson:"firstName" gofasta:"required"`
    CreatedAt time.Time `gorm:"autoCreateTime" bson:"createdAt" gofasta:"auto_now_add"`
}

type UserService struct {
    UserRepo orm.Repository[User] `inject:""`
}

func (s *UserService) FindActiveUsers() ([]*User, error) {
    return s.UserRepo.Query().
        Where("status", orm.OpEquals, "active").
        OrderBy("created_at", orm.DirectionDesc).
        Limit(10).
        Execute()
}

// Configuration - automatically detects database type
@Module{
    Imports: []interface{}{
        &orm.OrmModule{
            ConnectionURL: "postgresql://localhost:5432/myapp", // Uses GORM
            // ConnectionURL: "mongodb://localhost:27017/myapp", // Uses mongo-driver
        },
    },
}
type AppModule struct{}
```

## Database Configuration Examples

```go
// PostgreSQL
&orm.OrmModule{
    ConnectionURL: "postgresql://user:pass@localhost:5432/myapp",
    AutoMigrate:   true,
}

// MongoDB  
&orm.OrmModule{
    ConnectionURL: "mongodb://localhost:27017/myapp",
    Database:      "myapp",
}

// Multi-database setup
@Module{
    Imports: []interface{}{
        &orm.OrmModule{
            Name:          "primary",
            ConnectionURL: "postgresql://localhost:5432/main",
        },
        &orm.OrmModule{
            Name:          "analytics", 
            ConnectionURL: "mongodb://localhost:27017/analytics",
        },
    },
}
```