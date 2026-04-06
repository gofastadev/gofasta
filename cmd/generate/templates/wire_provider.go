package templates

var WireProvider = `package providers

import (
	"github.com/google/wire"
	"github.com/healtronlabs/gofasta/app/repositories"
	repoInterfaces "github.com/healtronlabs/gofasta/app/repositories/interfaces"
	"github.com/healtronlabs/gofasta/app/rest/controllers"
	"github.com/healtronlabs/gofasta/app/services"
	svcInterfaces "github.com/healtronlabs/gofasta/app/services/interfaces"
)

var {{.Name}}Set = wire.NewSet(
	repositories.New{{.Name}}Repository,
	wire.Bind(new(repoInterfaces.{{.Name}}RepositoryInterface), new(*repositories.{{.Name}}Repository)),
	services.New{{.Name}}Service,
	wire.Bind(new(svcInterfaces.{{.Name}}ServiceInterface), new(*services.{{.Name}}Service)),
	controllers.New{{.Name}}ControllerInstance,
)
`
