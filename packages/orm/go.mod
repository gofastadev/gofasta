module github.com/healtronlabs/gofasta/packages/orm

go 1.22.5

require (
	github.com/healtronlabs/gofasta/packages/core v0.0.0
	gorm.io/gorm v1.25.11
	gorm.io/driver/postgres v1.5.9
	gorm.io/driver/mysql v1.5.7
	gorm.io/driver/sqlite v1.5.6
	go.mongodb.org/mongo-driver v1.12.1
)

replace github.com/healtronlabs/gofasta/packages/core => ../core