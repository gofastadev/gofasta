package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/spf13/cobra"
)

func generateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate code components",
		Long:  "Generate controllers, services, models, and other components",
		Aliases: []string{"g"},
	}

	cmd.AddCommand(generateModuleCmd())
	cmd.AddCommand(generateControllerCmd())
	cmd.AddCommand(generateServiceCmd())
	cmd.AddCommand(generateModelCmd())
	cmd.AddCommand(generateRepositoryCmd())
	cmd.AddCommand(generateMiddlewareCmd())

	return cmd
}

func generateModuleCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "module [name]",
		Short: "Generate a new module",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return generateModule(args[0])
		},
	}
}

func generateControllerCmd() *cobra.Command {
	var modulePath string
	var withCRUD bool

	cmd := &cobra.Command{
		Use:   "controller [name]",
		Short: "Generate a new controller",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return generateController(args[0], modulePath, withCRUD)
		},
	}

	cmd.Flags().StringVarP(&modulePath, "module", "m", "internal/controllers", "Module path")
	cmd.Flags().BoolVar(&withCRUD, "crud", false, "Generate CRUD operations")

	return cmd
}

func generateServiceCmd() *cobra.Command {
	var modulePath string

	cmd := &cobra.Command{
		Use:   "service [name]",
		Short: "Generate a new service",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return generateService(args[0], modulePath)
		},
	}

	cmd.Flags().StringVarP(&modulePath, "module", "m", "internal/services", "Module path")

	return cmd
}

func generateModelCmd() *cobra.Command {
	var modulePath string
	var withValidation bool

	cmd := &cobra.Command{
		Use:   "model [name]",
		Short: "Generate a new model",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return generateModel(args[0], modulePath, withValidation)
		},
	}

	cmd.Flags().StringVarP(&modulePath, "module", "m", "internal/models", "Module path")
	cmd.Flags().BoolVar(&withValidation, "validation", true, "Include validation tags")

	return cmd
}

func generateRepositoryCmd() *cobra.Command {
	var modulePath string
	var modelName string

	cmd := &cobra.Command{
		Use:   "repository [name]",
		Short: "Generate a new repository",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return generateRepository(args[0], modulePath, modelName)
		},
	}

	cmd.Flags().StringVarP(&modulePath, "module", "m", "internal/repositories", "Module path")
	cmd.Flags().StringVar(&modelName, "model", "", "Associated model name")

	return cmd
}

func generateMiddlewareCmd() *cobra.Command {
	var modulePath string

	cmd := &cobra.Command{
		Use:   "middleware [name]",
		Short: "Generate a new middleware",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return generateMiddleware(args[0], modulePath)
		},
	}

	cmd.Flags().StringVarP(&modulePath, "module", "m", "internal/middleware", "Module path")

	return cmd
}

// Template data structures
type TemplateData struct {
	Name       string
	LowerName  string
	UpperName  string
	PluralName string
	Package    string
	ModelName  string
	WithCRUD   bool
	WithValidation bool
}

func generateModule(name string) error {
	data := &TemplateData{
		Name:      name,
		LowerName: strings.ToLower(name),
		UpperName: strings.Title(name),
		Package:   "main",
	}

	moduleTemplate := `package {{.Package}}

import (
	"github.com/healtronlabs/gofasta/packages/core"
)

type {{.UpperName}}Module struct {
	*core.BaseModule
}

func New{{.UpperName}}Module() *{{.UpperName}}Module {
	return &{{.UpperName}}Module{
		BaseModule: core.NewBaseModule("{{.LowerName}}"),
	}
}

func (m *{{.UpperName}}Module) Configure() {
	// Register providers, controllers, and other components
	// m.AddProvider(&services.{{.UpperName}}Service{})
	// m.AddController(&controllers.{{.UpperName}}Controller{})
}

func (m *{{.UpperName}}Module) GetExports() []interface{} {
	return []interface{}{
		// Export services that other modules can use
	}
}
`

	return generateFromTemplate("module", data, moduleTemplate, fmt.Sprintf("internal/modules/%s_module.go", data.LowerName))
}

func generateController(name, modulePath string, withCRUD bool) error {
	data := &TemplateData{
		Name:       name,
		LowerName:  strings.ToLower(name),
		UpperName:  strings.Title(name),
		PluralName: name + "s", // Simple pluralization
		Package:    filepath.Base(modulePath),
		WithCRUD:   withCRUD,
	}

	var controllerTemplate string
	if withCRUD {
		controllerTemplate = `package {{.Package}}

import (
	"context"
	"net/http"

	"github.com/healtronlabs/gofasta/packages/core"
	"github.com/healtronlabs/gofasta/packages/auth"
	
	"yourapp/internal/services"
	"yourapp/internal/models"
)

type {{.UpperName}}Controller struct {
	{{.UpperName}}Service *services.{{.UpperName}}Service ` + "`inject:\"\"`" + `
}

// Get{{.PluralName}} retrieves all {{.LowerName}}s
func (c *{{.UpperName}}Controller) Get{{.PluralName}}() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		{{.LowerName}}s, err := c.{{.UpperName}}Service.GetAll{{.PluralName}}(r.Context())
		if err != nil {
			core.WriteErrorResponse(w, err)
			return
		}

		core.WriteJSONResponse(w, http.StatusOK, {{.LowerName}}s)
	}
}

// Get{{.UpperName}} retrieves a {{.LowerName}} by ID
func (c *{{.UpperName}}Controller) Get{{.UpperName}}() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		if id == "" {
			core.WriteErrorResponse(w, core.NewBadRequestException("{{.UpperName}} ID is required"))
			return
		}

		{{.LowerName}}, err := c.{{.UpperName}}Service.Get{{.UpperName}}ByID(r.Context(), id)
		if err != nil {
			core.WriteErrorResponse(w, err)
			return
		}

		core.WriteJSONResponse(w, http.StatusOK, {{.LowerName}})
	}
}

// Create{{.UpperName}} creates a new {{.LowerName}}
func (c *{{.UpperName}}Controller) Create{{.UpperName}}() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var {{.LowerName}} models.{{.UpperName}}
		if err := core.ParseJSONBody(r, &{{.LowerName}}); err != nil {
			core.WriteErrorResponse(w, core.NewBadRequestException("Invalid request body"))
			return
		}

		created{{.UpperName}}, err := c.{{.UpperName}}Service.Create{{.UpperName}}(r.Context(), &{{.LowerName}})
		if err != nil {
			core.WriteErrorResponse(w, err)
			return
		}

		core.WriteJSONResponse(w, http.StatusCreated, created{{.UpperName}})
	}
}

// Update{{.UpperName}} updates an existing {{.LowerName}}
func (c *{{.UpperName}}Controller) Update{{.UpperName}}() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		if id == "" {
			core.WriteErrorResponse(w, core.NewBadRequestException("{{.UpperName}} ID is required"))
			return
		}

		var {{.LowerName}} models.{{.UpperName}}
		if err := core.ParseJSONBody(r, &{{.LowerName}}); err != nil {
			core.WriteErrorResponse(w, core.NewBadRequestException("Invalid request body"))
			return
		}

		updated{{.UpperName}}, err := c.{{.UpperName}}Service.Update{{.UpperName}}(r.Context(), &{{.LowerName}})
		if err != nil {
			core.WriteErrorResponse(w, err)
			return
		}

		core.WriteJSONResponse(w, http.StatusOK, updated{{.UpperName}})
	}
}

// Delete{{.UpperName}} deletes a {{.LowerName}}
func (c *{{.UpperName}}Controller) Delete{{.UpperName}}() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		if id == "" {
			core.WriteErrorResponse(w, core.NewBadRequestException("{{.UpperName}} ID is required"))
			return
		}

		err := c.{{.UpperName}}Service.Delete{{.UpperName}}(r.Context(), id)
		if err != nil {
			core.WriteErrorResponse(w, err)
			return
		}

		core.WriteJSONResponse(w, http.StatusNoContent, nil)
	}
}

// Routes returns the controller routes
func (c *{{.UpperName}}Controller) Routes() []core.Route {
	return []core.Route{
		{
			Method:  "GET",
			Path:    "/{{.LowerName}}s",
			Handler: c.Get{{.PluralName}}(),
			Guards:  []core.Guard{&auth.JWTGuard{}},
		},
		{
			Method:  "GET",
			Path:    "/{{.LowerName}}s/:id",
			Handler: c.Get{{.UpperName}}(),
			Guards:  []core.Guard{&auth.JWTGuard{}},
		},
		{
			Method:  "POST",
			Path:    "/{{.LowerName}}s",
			Handler: c.Create{{.UpperName}}(),
			Guards:  []core.Guard{&auth.JWTGuard{}},
		},
		{
			Method:  "PUT",
			Path:    "/{{.LowerName}}s/:id",
			Handler: c.Update{{.UpperName}}(),
			Guards:  []core.Guard{&auth.JWTGuard{}},
		},
		{
			Method:  "DELETE",
			Path:    "/{{.LowerName}}s/:id",
			Handler: c.Delete{{.UpperName}}(),
			Guards:  []core.Guard{&auth.JWTGuard{}},
		},
	}
}
`
	} else {
		controllerTemplate = `package {{.Package}}

import (
	"net/http"

	"github.com/healtronlabs/gofasta/packages/core"
)

type {{.UpperName}}Controller struct {
	// Add your dependencies here
	// SomeService *services.SomeService ` + "`inject:\"\"`" + `
}

// Handle{{.UpperName}} handles {{.LowerName}} requests
func (c *{{.UpperName}}Controller) Handle{{.UpperName}}() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Implement your logic here
		response := map[string]string{
			"message": "{{.UpperName}} endpoint",
		}

		core.WriteJSONResponse(w, http.StatusOK, response)
	}
}

// Routes returns the controller routes
func (c *{{.UpperName}}Controller) Routes() []core.Route {
	return []core.Route{
		{
			Method:  "GET",
			Path:    "/{{.LowerName}}",
			Handler: c.Handle{{.UpperName}}(),
		},
	}
}
`
	}

	fileName := fmt.Sprintf("%s/%s_controller.go", modulePath, data.LowerName)
	return generateFromTemplate("controller", data, controllerTemplate, fileName)
}

func generateService(name, modulePath string) error {
	data := &TemplateData{
		Name:       name,
		LowerName:  strings.ToLower(name),
		UpperName:  strings.Title(name),
		PluralName: name + "s",
		Package:    filepath.Base(modulePath),
	}

	serviceTemplate := `package {{.Package}}

import (
	"context"

	"github.com/healtronlabs/gofasta/packages/orm"
	
	"yourapp/internal/models"
)

type {{.UpperName}}Service struct {
	{{.UpperName}}Repo orm.Repository[models.{{.UpperName}}] ` + "`inject:\"\"`" + `
}

func (s *{{.UpperName}}Service) GetAll{{.PluralName}}(ctx context.Context) ([]*models.{{.UpperName}}, error) {
	return s.{{.UpperName}}Repo.Query().
		OrderBy("created_at", orm.DirectionDesc).
		Execute(ctx)
}

func (s *{{.UpperName}}Service) Get{{.UpperName}}ByID(ctx context.Context, id string) (*models.{{.UpperName}}, error) {
	return s.{{.UpperName}}Repo.FindByID(ctx, id)
}

func (s *{{.UpperName}}Service) Create{{.UpperName}}(ctx context.Context, {{.LowerName}} *models.{{.UpperName}}) (*models.{{.UpperName}}, error) {
	// Add business logic here
	return s.{{.UpperName}}Repo.Create(ctx, {{.LowerName}})
}

func (s *{{.UpperName}}Service) Update{{.UpperName}}(ctx context.Context, {{.LowerName}} *models.{{.UpperName}}) (*models.{{.UpperName}}, error) {
	// Add business logic here
	return s.{{.UpperName}}Repo.Update(ctx, {{.LowerName}})
}

func (s *{{.UpperName}}Service) Delete{{.UpperName}}(ctx context.Context, id string) error {
	return s.{{.UpperName}}Repo.Query().
		Where("id", orm.OpEquals, id).
		Delete(ctx)
}
`

	fileName := fmt.Sprintf("%s/%s_service.go", modulePath, data.LowerName)
	return generateFromTemplate("service", data, serviceTemplate, fileName)
}

func generateModel(name, modulePath string, withValidation bool) error {
	data := &TemplateData{
		Name:           name,
		LowerName:      strings.ToLower(name),
		UpperName:      strings.Title(name),
		Package:        filepath.Base(modulePath),
		WithValidation: withValidation,
	}

	var modelTemplate string
	if withValidation {
		modelTemplate = `package {{.Package}}

import (
	"time"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type {{.UpperName}} struct {
	ID        primitive.ObjectID ` + "`gorm:\"primaryKey\" bson:\"_id,omitempty\" json:\"id\"`" + `
	Name      string             ` + "`gorm:\"not null\" bson:\"name\" json:\"name\" validate:\"required\"`" + `
	Status    string             ` + "`gorm:\"type:varchar(20)\" bson:\"status\" json:\"status\" validate:\"oneof=active inactive\"`" + `
	CreatedAt time.Time          ` + "`gorm:\"autoCreateTime\" bson:\"createdAt\" json:\"createdAt\"`" + `
	UpdatedAt time.Time          ` + "`gorm:\"autoUpdateTime\" bson:\"updatedAt\" json:\"updatedAt\"`" + `
}

func ({{.LowerName}} *{{.UpperName}}) TableName() string {
	return "{{.LowerName}}s"
}
`
	} else {
		modelTemplate = `package {{.Package}}

import (
	"time"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type {{.UpperName}} struct {
	ID        primitive.ObjectID ` + "`gorm:\"primaryKey\" bson:\"_id,omitempty\" json:\"id\"`" + `
	Name      string             ` + "`gorm:\"not null\" bson:\"name\" json:\"name\"`" + `
	Status    string             ` + "`gorm:\"type:varchar(20)\" bson:\"status\" json:\"status\"`" + `
	CreatedAt time.Time          ` + "`gorm:\"autoCreateTime\" bson:\"createdAt\" json:\"createdAt\"`" + `
	UpdatedAt time.Time          ` + "`gorm:\"autoUpdateTime\" bson:\"updatedAt\" json:\"updatedAt\"`" + `
}

func ({{.LowerName}} *{{.UpperName}}) TableName() string {
	return "{{.LowerName}}s"
}
`
	}

	fileName := fmt.Sprintf("%s/%s.go", modulePath, data.LowerName)
	return generateFromTemplate("model", data, modelTemplate, fileName)
}

func generateRepository(name, modulePath, modelName string) error {
	if modelName == "" {
		modelName = name
	}

	data := &TemplateData{
		Name:      name,
		LowerName: strings.ToLower(name),
		UpperName: strings.Title(name),
		Package:   filepath.Base(modulePath),
		ModelName: strings.Title(modelName),
	}

	repositoryTemplate := `package {{.Package}}

import (
	"context"

	"github.com/healtronlabs/gofasta/packages/orm"
	
	"yourapp/internal/models"
)

type {{.UpperName}}Repository struct {
	orm.Repository[models.{{.ModelName}}]
}

func New{{.UpperName}}Repository(adapter orm.DatabaseAdapter) *{{.UpperName}}Repository {
	return &{{.UpperName}}Repository{
		Repository: orm.NewRepository[models.{{.ModelName}}](adapter),
	}
}

// Custom repository methods
func (r *{{.UpperName}}Repository) FindByStatus(ctx context.Context, status string) ([]*models.{{.ModelName}}, error) {
	return r.Query().
		Where("status", orm.OpEquals, status).
		OrderBy("created_at", orm.DirectionDesc).
		Execute(ctx)
}

func (r *{{.UpperName}}Repository) FindActive(ctx context.Context) ([]*models.{{.ModelName}}, error) {
	return r.FindByStatus(ctx, "active")
}
`

	fileName := fmt.Sprintf("%s/%s_repository.go", modulePath, data.LowerName)
	return generateFromTemplate("repository", data, repositoryTemplate, fileName)
}

func generateMiddleware(name, modulePath string) error {
	data := &TemplateData{
		Name:      name,
		LowerName: strings.ToLower(name),
		UpperName: strings.Title(name),
		Package:   filepath.Base(modulePath),
	}

	middlewareTemplate := `package {{.Package}}

import (
	"net/http"

	"github.com/healtronlabs/gofasta/packages/core"
)

type {{.UpperName}}Middleware struct {
	// Add dependencies here
}

func New{{.UpperName}}Middleware() *{{.UpperName}}Middleware {
	return &{{.UpperName}}Middleware{}
}

func (m *{{.UpperName}}Middleware) Use() core.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Implement middleware logic here
			
			// Call next handler
			next.ServeHTTP(w, r)
		})
	}
}
`

	fileName := fmt.Sprintf("%s/%s_middleware.go", modulePath, data.LowerName)
	return generateFromTemplate("middleware", data, middlewareTemplate, fileName)
}

func generateFromTemplate(templateName string, data *TemplateData, templateContent, fileName string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(fileName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Parse template
	tmpl, err := template.New(templateName).Parse(templateContent)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	// Create file
	file, err := os.Create(fileName)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", fileName, err)
	}
	defer file.Close()

	// Execute template
	if err := tmpl.Execute(file, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	fmt.Printf("✅ Generated %s: %s\n", templateName, fileName)
	return nil
}