package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"text/template"

	"github.com/spf13/cobra"
)

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate boilerplate code",
}

var generateModelCmd = &cobra.Command{
	Use:   "model [name]",
	Short: "Generate a new model",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return generateFile("model", args[0])
	},
}

var generateServiceCmd = &cobra.Command{
	Use:   "service [name]",
	Short: "Generate a new service with interface",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return generateFile("service", args[0])
	},
}

var generateRepositoryCmd = &cobra.Command{
	Use:   "repository [name]",
	Short: "Generate a new repository with interface",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return generateFile("repository", args[0])
	},
}

var generateControllerCmd = &cobra.Command{
	Use:   "controller [name]",
	Short: "Generate a new REST controller",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return generateFile("controller", args[0])
	},
}

var wireCmd = &cobra.Command{
	Use:   "wire",
	Short: "Run Wire to regenerate dependency injection code",
	RunE: func(cmd *cobra.Command, args []string) error {
		wireCmd := exec.Command("wire", "./app/di/")
		wireCmd.Stdout = os.Stdout
		wireCmd.Stderr = os.Stderr
		return wireCmd.Run()
	},
}

func init() {
	generateCmd.AddCommand(generateModelCmd)
	generateCmd.AddCommand(generateServiceCmd)
	generateCmd.AddCommand(generateRepositoryCmd)
	generateCmd.AddCommand(generateControllerCmd)
	rootCmd.AddCommand(generateCmd)
	rootCmd.AddCommand(wireCmd)
}

type templateData struct {
	Name      string // PascalCase: User
	LowerName string // camelCase: user
	SnakeName string // snake_case: user
}

func generateFile(kind, name string) error {
	data := templateData{
		Name:      strings.Title(strings.ToLower(name)),
		LowerName: strings.ToLower(name[:1]) + name[1:],
		SnakeName: toSnakeCase(name),
	}

	templates := map[string]struct {
		path string
		tmpl string
	}{
		"model": {
			path: fmt.Sprintf("app/models/%s.model.go", data.SnakeName),
			tmpl: modelTemplate,
		},
		"service": {
			path: fmt.Sprintf("app/services/%s.service.go", data.SnakeName),
			tmpl: serviceTemplate,
		},
		"repository": {
			path: fmt.Sprintf("app/repositories/%s.repository.go", data.SnakeName),
			tmpl: repositoryTemplate,
		},
		"controller": {
			path: fmt.Sprintf("app/rest/controllers/%s.controller.go", data.SnakeName),
			tmpl: controllerTemplate,
		},
	}

	entry, ok := templates[kind]
	if !ok {
		return fmt.Errorf("unknown generator: %s", kind)
	}

	if _, err := os.Stat(entry.path); err == nil {
		return fmt.Errorf("file already exists: %s", entry.path)
	}

	t, err := template.New(kind).Parse(entry.tmpl)
	if err != nil {
		return err
	}

	f, err := os.Create(entry.path)
	if err != nil {
		return err
	}
	defer f.Close()

	if err := t.Execute(f, data); err != nil {
		return err
	}

	fmt.Printf("Generated %s: %s\n", kind, entry.path)

	// Also generate interface file for service and repository
	if kind == "service" {
		return generateServiceInterface(data)
	}
	if kind == "repository" {
		return generateRepositoryInterface(data)
	}
	return nil
}

func generateServiceInterface(data templateData) error {
	path := fmt.Sprintf("app/services/interfaces/%s_service.go", data.SnakeName)
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("file already exists: %s", path)
	}
	t, err := template.New("svc_interface").Parse(serviceInterfaceTemplate)
	if err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := t.Execute(f, data); err != nil {
		return err
	}
	fmt.Printf("Generated service interface: %s\n", path)
	return nil
}

func generateRepositoryInterface(data templateData) error {
	path := fmt.Sprintf("app/repositories/interfaces/%s_repository.go", data.SnakeName)
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("file already exists: %s", path)
	}
	t, err := template.New("repo_interface").Parse(repositoryInterfaceTemplate)
	if err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := t.Execute(f, data); err != nil {
		return err
	}
	fmt.Printf("Generated repository interface: %s\n", path)
	return nil
}

func toSnakeCase(s string) string {
	var result []byte
	for i, c := range s {
		if c >= 'A' && c <= 'Z' {
			if i > 0 {
				result = append(result, '_')
			}
			result = append(result, byte(c+32))
		} else {
			result = append(result, byte(c))
		}
	}
	return string(result)
}

var modelTemplate = `package models

// {{.Name}} represents the {{.LowerName}} domain entity.
type {{.Name}} struct {
	BaseModelImpl
	// Add fields here
}
`

var serviceTemplate = `package services

import (
	"context"

	repoInterfaces "github.com/healtronlabs/gofasta/app/repositories/interfaces"
	"github.com/healtronlabs/gofasta/app/validators"
	"gorm.io/gorm"
)

type {{.Name}}Service struct {
	{{.Name}}Repo repoInterfaces.{{.Name}}RepositoryInterface
	Validator     *validators.AppValidator
	DB            *gorm.DB
}

func New{{.Name}}Service(db *gorm.DB, repo repoInterfaces.{{.Name}}RepositoryInterface, validator *validators.AppValidator) *{{.Name}}Service {
	return &{{.Name}}Service{
		{{.Name}}Repo: repo,
		Validator:     validator,
		DB:            db,
	}
}

// Add service methods here. Example:
// func (s *{{.Name}}Service) FindByID(ctx context.Context, id uuid.UUID) (*dtos.{{.Name}}ResponseDto, error) { ... }
`

var repositoryTemplate = `package repositories

import (
	"context"

	"github.com/google/uuid"
	"github.com/healtronlabs/gofasta/app/models"
	repoInterfaces "github.com/healtronlabs/gofasta/app/repositories/interfaces"
	"gorm.io/gorm"
)

var _ repoInterfaces.{{.Name}}RepositoryInterface = (*{{.Name}}Repository)(nil)

type {{.Name}}Repository struct {
	DB *gorm.DB
}

func New{{.Name}}Repository(db *gorm.DB) *{{.Name}}Repository {
	return &{{.Name}}Repository{DB: db}
}

func (r *{{.Name}}Repository) FindByID(ctx context.Context, id uuid.UUID) (*models.{{.Name}}, error) {
	var entity models.{{.Name}}
	if err := r.DB.WithContext(ctx).Where("id = ?", id).First(&entity).Error; err != nil {
		return nil, err
	}
	return &entity, nil
}

func (r *{{.Name}}Repository) Create(ctx context.Context, entity *models.{{.Name}}) error {
	return r.DB.WithContext(ctx).Create(entity).Error
}
`

var controllerTemplate = `package controllers

import (
	"net/http"

	svcInterfaces "github.com/healtronlabs/gofasta/app/services/interfaces"
)

type {{.Name}}Controller struct {
	{{.Name}}Service svcInterfaces.{{.Name}}ServiceInterface
}

func New{{.Name}}ControllerInstance(svc svcInterfaces.{{.Name}}ServiceInterface) *{{.Name}}Controller {
	return &{{.Name}}Controller{ {{.Name}}Service: svc}
}

// Add handler methods here. Example:
// func (c *{{.Name}}Controller) GetByID(w http.ResponseWriter, r *http.Request) error { ... }
`

var serviceInterfaceTemplate = `package interfaces

import (
	"context"
)

// {{.Name}}ServiceInterface defines the contract for {{.LowerName}} business logic.
type {{.Name}}ServiceInterface interface {
	// Add methods here
}
`

var repositoryInterfaceTemplate = `package interfaces

import (
	"context"

	"github.com/google/uuid"
	"github.com/healtronlabs/gofasta/app/models"
)

// {{.Name}}RepositoryInterface defines the contract for {{.LowerName}} data access.
type {{.Name}}RepositoryInterface interface {
	FindByID(ctx context.Context, id uuid.UUID) (*models.{{.Name}}, error)
	Create(ctx context.Context, entity *models.{{.Name}}) error
}
`
