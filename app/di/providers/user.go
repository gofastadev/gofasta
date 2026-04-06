package providers

import (
	"github.com/google/wire"
	"github.com/healtronlabs/gofasta/app/repositories"
	repoInterfaces "github.com/healtronlabs/gofasta/app/repositories/interfaces"
	"github.com/healtronlabs/gofasta/app/rest/controllers"
	"github.com/healtronlabs/gofasta/app/services"
	svcInterfaces "github.com/healtronlabs/gofasta/app/services/interfaces"
)

// UserSet provides user domain: repository, service, controller.
var UserSet = wire.NewSet(
	repositories.NewUserRepository,
	wire.Bind(new(repoInterfaces.UserRepositoryInterface), new(*repositories.UserRepository)),
	services.NewUserService,
	wire.Bind(new(svcInterfaces.UserServiceInterface), new(*services.UserService)),
	controllers.NewUserControllerInstance,
)
