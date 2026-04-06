package di

import (
	"log/slog"

	"github.com/healtronlabs/gofasta/app/graphql/resolvers"
	repoInterfaces "github.com/healtronlabs/gofasta/app/repositories/interfaces"
	"github.com/healtronlabs/gofasta/app/rest/controllers"
	svcInterfaces "github.com/healtronlabs/gofasta/app/services/interfaces"
	"github.com/healtronlabs/gofasta/app/validators"
	"github.com/healtronlabs/gofasta/configs"
	"gorm.io/gorm"
)

// ServiceContainer holds all application dependencies, assembled by Wire.
type ServiceContainer struct {
	Config         *configs.AppConfig
	DB             *gorm.DB
	Logger         *slog.Logger
	Validator      *validators.AppValidator
	UserRepo       repoInterfaces.UserRepositoryInterface
	UserService    svcInterfaces.UserServiceInterface
	UserController *controllers.UserController
	Resolver       *resolvers.Resolver
}
