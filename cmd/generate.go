package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/spf13/cobra"
)

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate boilerplate code",
	Aliases: []string{"g"},
}

var scaffoldCmd = &cobra.Command{
	Use:   "scaffold [Name] [field:type ...]",
	Short: "Generate a full resource (model, repository, service, controller, routes, DTOs, GraphQL schema, migration, Wire provider)",
	Long: `Generate all files for a new resource domain. Like Rails scaffolding, this creates
everything you need so you can focus on business logic.

Examples:
  gofasta generate scaffold Product name:string price:float description:text
  gofasta g scaffold BlogPost title:string body:text published:bool
  gofasta g scaffold Order totalAmount:float status:string

Supported field types: string, text, int, float, bool, uuid, time`,
	Aliases: []string{"s"},
	Args:    cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		fields := parseFields(args[1:])
		return runScaffold(name, fields)
	},
}

var generateModelCmd = &cobra.Command{
	Use:   "model [Name] [field:type ...]",
	Short: "Generate a new model",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fields := parseFields(args[1:])
		return generateSingleFile("model", args[0], fields)
	},
}

var generateServiceCmd = &cobra.Command{
	Use:   "service [Name]",
	Short: "Generate a new service with interface",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return generateSingleFile("service", args[0], nil)
	},
}

var generateRepositoryCmd = &cobra.Command{
	Use:   "repository [Name]",
	Short: "Generate a new repository with interface",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return generateSingleFile("repository", args[0], nil)
	},
}

var generateControllerCmd = &cobra.Command{
	Use:   "controller [Name]",
	Short: "Generate a new REST controller",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return generateSingleFile("controller", args[0], nil)
	},
}

var wireCmd = &cobra.Command{
	Use:   "wire",
	Short: "Run Wire to regenerate dependency injection code",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Running Wire...")
		wireExec := exec.Command("go", "tool", "wire", "./app/di/")
		wireExec.Stdout = os.Stdout
		wireExec.Stderr = os.Stderr
		return wireExec.Run()
	},
}

func init() {
	generateCmd.AddCommand(scaffoldCmd)
	generateCmd.AddCommand(generateModelCmd)
	generateCmd.AddCommand(generateServiceCmd)
	generateCmd.AddCommand(generateRepositoryCmd)
	generateCmd.AddCommand(generateControllerCmd)
	rootCmd.AddCommand(generateCmd)
	rootCmd.AddCommand(wireCmd)
}

// --- Field parsing ---

type Field struct {
	Name       string // PascalCase: ProductName
	JSONName   string // camelCase: productName
	SnakeName  string // snake_case: product_name
	GoType     string // Go type: string
	GormType   string // GORM column type: VARCHAR(255)
	GQLType    string // GraphQL type: String
	SQLType    string // SQL type: VARCHAR(255)
	IsNullable bool
}

type scaffoldData struct {
	Name       string // PascalCase: Product
	LowerName  string // camelCase: product
	SnakeName  string // snake_case: product
	PluralName string // PascalCase plural: Products
	PluralSnake string // snake_case plural: products
	PluralLower string // camelCase plural: products
	Fields     []Field
	MigrationNum string
}

func parseFields(args []string) []Field {
	var fields []Field
	for _, arg := range args {
		parts := strings.SplitN(arg, ":", 2)
		if len(parts) != 2 {
			continue
		}
		fieldName := parts[0]
		fieldType := parts[1]

		f := Field{
			Name:      toPascalCase(fieldName),
			JSONName:  toCamelCase(fieldName),
			SnakeName: toSnakeCase(fieldName),
		}

		switch strings.ToLower(fieldType) {
		case "string":
			f.GoType = "string"
			f.GormType = `gorm:"not null"`
			f.GQLType = "String"
			f.SQLType = "VARCHAR(255) NOT NULL"
		case "text":
			f.GoType = "string"
			f.GormType = `gorm:"type:text;not null"`
			f.GQLType = "String"
			f.SQLType = "TEXT NOT NULL"
		case "int":
			f.GoType = "int"
			f.GormType = `gorm:"not null"`
			f.GQLType = "Int"
			f.SQLType = "INTEGER NOT NULL"
		case "float":
			f.GoType = "float64"
			f.GormType = `gorm:"not null"`
			f.GQLType = "Float"
			f.SQLType = "DECIMAL(10,2) NOT NULL"
		case "bool":
			f.GoType = "bool"
			f.GormType = `gorm:"not null;default:false"`
			f.GQLType = "Boolean"
			f.SQLType = "BOOLEAN NOT NULL DEFAULT false"
		case "uuid":
			f.GoType = "uuid.UUID"
			f.GormType = `gorm:"type:uuid;not null"`
			f.GQLType = "ID"
			f.SQLType = "UUID NOT NULL"
		case "time", "datetime":
			f.GoType = "time.Time"
			f.GormType = `gorm:"type:timestamp;not null"`
			f.GQLType = "DateTime"
			f.SQLType = "TIMESTAMP NOT NULL DEFAULT now()"
		default:
			f.GoType = "string"
			f.GormType = `gorm:"not null"`
			f.GQLType = "String"
			f.SQLType = "VARCHAR(255) NOT NULL"
		}

		fields = append(fields, f)
	}
	return fields
}

func pluralize(s string) string {
	if strings.HasSuffix(s, "s") || strings.HasSuffix(s, "x") || strings.HasSuffix(s, "z") ||
		strings.HasSuffix(s, "ch") || strings.HasSuffix(s, "sh") {
		return s + "es"
	}
	if strings.HasSuffix(s, "y") && len(s) > 1 {
		c := s[len(s)-2]
		if c != 'a' && c != 'e' && c != 'i' && c != 'o' && c != 'u' {
			return s[:len(s)-1] + "ies"
		}
	}
	return s + "s"
}

func nextMigrationNumber() string {
	entries, err := os.ReadDir("db/migrations")
	if err != nil {
		return "000006"
	}
	max := 0
	for _, e := range entries {
		name := e.Name()
		if len(name) >= 6 {
			var num int
			fmt.Sscanf(name[:6], "%d", &num)
			if num > max {
				max = num
			}
		}
	}
	return fmt.Sprintf("%06d", max+1)
}

// --- Scaffold orchestrator ---

func runScaffold(name string, fields []Field) error {
	data := buildScaffoldData(name, fields)

	steps := []struct {
		label string
		fn    func(scaffoldData) error
	}{
		{"model", genModel},
		{"repository interface", genRepoInterface},
		{"repository", genRepo},
		{"service interface", genSvcInterface},
		{"service", genSvc},
		{"controller", genController},
		{"routes", genRoutes},
		{"DTOs", genDTOs},
		{"GraphQL schema", genGraphQL},
		{"Wire provider set", genWireProvider},
		{"migration", genMigration},
	}

	for _, step := range steps {
		if err := step.fn(data); err != nil {
			return fmt.Errorf("failed to generate %s: %w", step.label, err)
		}
	}

	fmt.Println("\n--- Scaffold complete ---")
	fmt.Println("\nNext steps:")
	fmt.Printf("  1. Add %sController to RouteConfig in app/rest/routes/index.routes.go\n", data.Name)
	fmt.Printf("  2. Add %sSet to wire.Build in app/di/wire.go\n", data.Name)
	fmt.Printf("  3. Add %s fields to ServiceContainer in app/di/container.go\n", data.Name)
	fmt.Printf("  4. Run: gofasta wire\n")
	fmt.Printf("  5. Run: gofasta generate wire  (or go tool wire ./app/di/)\n")
	fmt.Printf("  6. Run: go tool gqlgen generate\n")
	fmt.Printf("  7. Run: gofasta migrate up\n")
	fmt.Printf("  8. Implement your business logic in app/services/%s.service.go\n", data.SnakeName)
	return nil
}

func buildScaffoldData(name string, fields []Field) scaffoldData {
	pascal := toPascalCase(name)
	snake := toSnakeCase(name)
	lower := toCamelCase(name)
	plural := pluralize(pascal)
	return scaffoldData{
		Name:         pascal,
		LowerName:    lower,
		SnakeName:    snake,
		PluralName:   plural,
		PluralSnake:  toSnakeCase(plural),
		PluralLower:  toCamelCase(plural),
		Fields:       fields,
		MigrationNum: nextMigrationNumber(),
	}
}

func generateSingleFile(kind, name string, fields []Field) error {
	data := buildScaffoldData(name, fields)
	switch kind {
	case "model":
		return genModel(data)
	case "service":
		if err := genSvcInterface(data); err != nil {
			return err
		}
		return genSvc(data)
	case "repository":
		if err := genRepoInterface(data); err != nil {
			return err
		}
		return genRepo(data)
	case "controller":
		return genController(data)
	default:
		return fmt.Errorf("unknown generator: %s", kind)
	}
}

// --- File writer helper ---

func writeTemplate(path, name, tmpl string, data scaffoldData) error {
	if _, err := os.Stat(path); err == nil {
		fmt.Printf("  skip (exists): %s\n", path)
		return nil
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	funcMap := template.FuncMap{
		"timestamp": func() string { return time.Now().Format(time.RFC3339) },
	}
	t, err := template.New(name).Funcs(funcMap).Parse(tmpl)
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
	fmt.Printf("  create: %s\n", path)
	return nil
}

// --- String utilities ---

func toPascalCase(s string) string {
	parts := strings.FieldsFunc(s, func(r rune) bool { return r == '_' || r == '-' })
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, "")
}

func toCamelCase(s string) string {
	p := toPascalCase(s)
	if len(p) == 0 {
		return p
	}
	return strings.ToLower(p[:1]) + p[1:]
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

// ============================================================================
// TEMPLATES
// ============================================================================

func genModel(d scaffoldData) error {
	return writeTemplate(
		fmt.Sprintf("app/models/%s.model.go", d.SnakeName),
		"model", tplModel, d,
	)
}

var tplModel = `package models

// {{.Name}} represents the {{.LowerName}} domain entity.
type {{.Name}} struct {
	BaseModelImpl
{{- range .Fields}}
	{{.Name}} {{.GoType}} ` + "`" + `{{.GormType}} json:"{{.JSONName}}"` + "`" + `
{{- end}}
}
`

func genRepoInterface(d scaffoldData) error {
	return writeTemplate(
		fmt.Sprintf("app/repositories/interfaces/%s_repository.go", d.SnakeName),
		"repo_iface", tplRepoInterface, d,
	)
}

var tplRepoInterface = `package interfaces

import (
	"context"

	"github.com/google/uuid"
	"github.com/healtronlabs/gofasta/app/models"
)

// {{.Name}}RepositoryInterface defines the contract for {{.LowerName}} data access.
type {{.Name}}RepositoryInterface interface {
	FindAll(ctx context.Context, page, limit int, sort string) ([]*models.{{.Name}}, int64, error)
	FindByID(ctx context.Context, id uuid.UUID) (*models.{{.Name}}, error)
	FindByIDAndRecordVersion(ctx context.Context, id uuid.UUID, version int) (*models.{{.Name}}, error)
	Create(ctx context.Context, entity *models.{{.Name}}) error
	Update(ctx context.Context, id uuid.UUID, fields map[string]interface{}) error
	SoftDelete(ctx context.Context, id uuid.UUID) error
}
`

func genRepo(d scaffoldData) error {
	return writeTemplate(
		fmt.Sprintf("app/repositories/%s.repository.go", d.SnakeName),
		"repo", tplRepo, d,
	)
}

var tplRepo = `package repositories

import (
	"context"
	"time"

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

func (r *{{.Name}}Repository) FindAll(ctx context.Context, page, limit int, sort string) ([]*models.{{.Name}}, int64, error) {
	var total int64
	query := r.DB.WithContext(ctx).Model(&models.{{.Name}}{}).Where("deleted_at IS NULL")
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var entities []*models.{{.Name}}
	offset := (page - 1) * limit
	if err := query.Limit(limit).Offset(offset).Order(sort).Find(&entities).Error; err != nil {
		return nil, 0, err
	}
	return entities, total, nil
}

func (r *{{.Name}}Repository) FindByID(ctx context.Context, id uuid.UUID) (*models.{{.Name}}, error) {
	var entity models.{{.Name}}
	if err := r.DB.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&entity).Error; err != nil {
		return nil, err
	}
	return &entity, nil
}

func (r *{{.Name}}Repository) FindByIDAndRecordVersion(ctx context.Context, id uuid.UUID, version int) (*models.{{.Name}}, error) {
	var entity models.{{.Name}}
	if err := r.DB.WithContext(ctx).Where("id = ? AND deleted_at IS NULL AND record_version = ?", id, version).First(&entity).Error; err != nil {
		return nil, err
	}
	return &entity, nil
}

func (r *{{.Name}}Repository) Create(ctx context.Context, entity *models.{{.Name}}) error {
	return r.DB.WithContext(ctx).Create(entity).Error
}

func (r *{{.Name}}Repository) Update(ctx context.Context, id uuid.UUID, fields map[string]interface{}) error {
	return r.DB.WithContext(ctx).Model(&models.{{.Name}}{}).Where("id = ?", id).Updates(fields).Error
}

func (r *{{.Name}}Repository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	return r.DB.WithContext(ctx).Model(&models.{{.Name}}{}).
		Where("id = ? AND is_deletable = ?", id, true).
		Updates(map[string]interface{}{"deleted_at": time.Now(), "is_active": false}).Error
}
`

func genSvcInterface(d scaffoldData) error {
	return writeTemplate(
		fmt.Sprintf("app/services/interfaces/%s_service.go", d.SnakeName),
		"svc_iface", tplSvcInterface, d,
	)
}

var tplSvcInterface = `package interfaces

import (
	"context"

	"github.com/healtronlabs/gofasta/app/dtos"
)

// {{.Name}}ServiceInterface defines the contract for {{.LowerName}} business logic.
type {{.Name}}ServiceInterface interface {
	FindAll(ctx context.Context, filters dtos.{{.Name}}FiltersDto) (*dtos.T{{.PluralName}}ResponseDto, error)
	FindByID(ctx context.Context, input dtos.TFind{{.Name}}ByIDDto) (*dtos.T{{.Name}}ResponseDto, error)
	Create(ctx context.Context, input dtos.TCreate{{.Name}}Dto) (*dtos.T{{.Name}}ResponseDto, error)
	Update(ctx context.Context, input dtos.TUpdate{{.Name}}Dto) (*dtos.T{{.Name}}ResponseDto, error)
	Archive(ctx context.Context, input dtos.TArchive{{.Name}}Dto) (*dtos.TCommonResponseDto, error)
}
`

func genSvc(d scaffoldData) error {
	return writeTemplate(
		fmt.Sprintf("app/services/%s.service.go", d.SnakeName),
		"svc", tplSvc, d,
	)
}

var tplSvc = `package services

import (
	"context"
	"math"

	"github.com/healtronlabs/gofasta/app/dtos"
	"github.com/healtronlabs/gofasta/app/models"
	repoInterfaces "github.com/healtronlabs/gofasta/app/repositories/interfaces"
	svcInterfaces "github.com/healtronlabs/gofasta/app/services/interfaces"
	"github.com/healtronlabs/gofasta/app/utils"
	"github.com/healtronlabs/gofasta/app/validators"
)

var _ svcInterfaces.{{.Name}}ServiceInterface = (*{{.Name}}Service)(nil)

type {{.Name}}Service struct {
	{{.Name}}Repo repoInterfaces.{{.Name}}RepositoryInterface
	Validator     *validators.AppValidator
}

func New{{.Name}}Service(repo repoInterfaces.{{.Name}}RepositoryInterface, validator *validators.AppValidator) *{{.Name}}Service {
	return &{{.Name}}Service{
		{{.Name}}Repo: repo,
		Validator:     validator,
	}
}

func (s *{{.Name}}Service) FindAll(ctx context.Context, filters dtos.{{.Name}}FiltersDto) (*dtos.T{{.PluralName}}ResponseDto, error) {
	paginator := utils.PreparePaginating{PageFilters: filters.Pagination, Sorting: filters.Sorting}
	page := paginator.GetPage()
	limit := paginator.GetLimit()
	sort := paginator.GetSort()

	entities, totalCount, err := s.{{.Name}}Repo.FindAll(ctx, page, limit, sort)
	if err != nil {
		return nil, err
	}

	var items []*dtos.{{.Name}}
	for _, e := range entities {
		items = append(items, cast{{.Name}}ModelToDto(e))
	}

	totalRecords := int(totalCount)
	totalPages := int(math.Ceil(float64(totalCount) / float64(limit)))
	return &dtos.T{{.PluralName}}ResponseDto{
		Data: items,
		Pagination: &dtos.TPaginationObjectDto{
			TotalRecords:   &totalRecords,
			CurrentPage:    &page,
			RecordsPerPage: &limit,
			TotalPages:     &totalPages,
		},
	}, nil
}

func (s *{{.Name}}Service) FindByID(ctx context.Context, input dtos.TFind{{.Name}}ByIDDto) (*dtos.T{{.Name}}ResponseDto, error) {
	if errs := s.Validator.ValidateStruct(input); len(errs) > 0 {
		return &dtos.T{{.Name}}ResponseDto{Errors: errs}, nil
	}
	entity, err := s.{{.Name}}Repo.FindByID(ctx, input.ID)
	if err != nil {
		return nil, err
	}
	return &dtos.T{{.Name}}ResponseDto{Data: cast{{.Name}}ModelToDto(entity)}, nil
}

func (s *{{.Name}}Service) Create(ctx context.Context, input dtos.TCreate{{.Name}}Dto) (*dtos.T{{.Name}}ResponseDto, error) {
	if errs := s.Validator.ValidateStruct(input); len(errs) > 0 {
		return &dtos.T{{.Name}}ResponseDto{Errors: errs}, nil
	}
	entity := &models.{{.Name}}{
		// TODO: Map input fields to model
	}
	if err := s.{{.Name}}Repo.Create(ctx, entity); err != nil {
		return nil, err
	}
	return &dtos.T{{.Name}}ResponseDto{Data: cast{{.Name}}ModelToDto(entity)}, nil
}

func (s *{{.Name}}Service) Update(ctx context.Context, input dtos.TUpdate{{.Name}}Dto) (*dtos.T{{.Name}}ResponseDto, error) {
	if errs := s.Validator.ValidateStruct(input); len(errs) > 0 {
		return &dtos.T{{.Name}}ResponseDto{Errors: errs}, nil
	}
	if found, _ := s.{{.Name}}Repo.FindByIDAndRecordVersion(ctx, input.ID, input.RecordVersion); found == nil {
		fieldName := "recordVersion"
		return &dtos.T{{.Name}}ResponseDto{Errors: []*dtos.TCommonAPIErrorDto{{"{"}}FieldName: &fieldName, Message: "The record version you passed is not matching"{{"}"}}}, nil
	}
	fields := utils.ConvertStructToMap(input)
	if err := s.{{.Name}}Repo.Update(ctx, input.ID, fields); err != nil {
		return nil, err
	}
	updated, err := s.{{.Name}}Repo.FindByID(ctx, input.ID)
	if err != nil {
		return nil, err
	}
	return &dtos.T{{.Name}}ResponseDto{Data: cast{{.Name}}ModelToDto(updated)}, nil
}

func (s *{{.Name}}Service) Archive(ctx context.Context, input dtos.TArchive{{.Name}}Dto) (*dtos.TCommonResponseDto, error) {
	if errs := s.Validator.ValidateStruct(input); len(errs) > 0 {
		return &dtos.TCommonResponseDto{Errors: errs}, nil
	}
	if err := s.{{.Name}}Repo.SoftDelete(ctx, input.ID); err != nil {
		return nil, err
	}
	status := 200
	message := "Success"
	return &dtos.TCommonResponseDto{Status: status, Message: &message}, nil
}

func cast{{.Name}}ModelToDto(e *models.{{.Name}}) *dtos.{{.Name}} {
	return &dtos.{{.Name}}{
		ID:            e.ID,
		RecordVersion: e.RecordVersion,
		CreatedAt:     e.CreatedAt,
		UpdatedAt:     e.UpdatedAt,
		IsActive:      e.IsActive,
		IsDeletable:   e.IsDeletable,
		DeletedAt:     &e.DeletedAt,
		// TODO: Map model fields to DTO
	}
}
`

func genController(d scaffoldData) error {
	return writeTemplate(
		fmt.Sprintf("app/rest/controllers/%s.controller.go", d.SnakeName),
		"controller", tplController, d,
	)
}

var tplController = `package controllers

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/healtronlabs/gofasta/app/dtos"
	svcInterfaces "github.com/healtronlabs/gofasta/app/services/interfaces"
	"github.com/healtronlabs/gofasta/app/utils"
	apperrors "github.com/healtronlabs/gofasta/pkg/errors"
	"github.com/healtronlabs/gofasta/pkg/httputil"
)

type {{.Name}}Controller struct {
	{{.Name}}Service svcInterfaces.{{.Name}}ServiceInterface
}

func New{{.Name}}ControllerInstance(svc svcInterfaces.{{.Name}}ServiceInterface) *{{.Name}}Controller {
	return &{{.Name}}Controller{ {{.Name}}Service: svc}
}

func (c *{{.Name}}Controller) List(w http.ResponseWriter, r *http.Request) error {
	filters := dtos.{{.Name}}FiltersDto{
		Pagination: &dtos.TPaginationInputDto{},
		Sorting:    &dtos.TSortingInputDto{SortByField: "created_at"},
	}
	res, err := c.{{.Name}}Service.FindAll(r.Context(), filters)
	if err != nil {
		return apperrors.NewInternal("failed to fetch {{.PluralLower}}", err)
	}
	return httputil.OK(w, res)
}

func (c *{{.Name}}Controller) GetByID(w http.ResponseWriter, r *http.Request) error {
	id, err := utils.ParseIdStringIsValidUUID(mux.Vars(r)["id"])
	if err != nil {
		return apperrors.NewBadRequest("id should be a valid UUID", nil)
	}
	res, err := c.{{.Name}}Service.FindByID(r.Context(), dtos.TFind{{.Name}}ByIDDto{ID: id})
	if err != nil {
		return apperrors.NewInternal("failed to find {{.LowerName}}", err)
	}
	return httputil.OK(w, res)
}

func (c *{{.Name}}Controller) Create(w http.ResponseWriter, r *http.Request) error {
	var input dtos.TCreate{{.Name}}Dto
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		return apperrors.NewBadRequest("invalid request payload", nil)
	}
	res, err := c.{{.Name}}Service.Create(r.Context(), input)
	if err != nil {
		return apperrors.NewInternal("failed to create {{.LowerName}}", err)
	}
	return httputil.Created(w, res)
}

func (c *{{.Name}}Controller) Update(w http.ResponseWriter, r *http.Request) error {
	id, err := utils.ParseIdStringIsValidUUID(mux.Vars(r)["id"])
	if err != nil {
		return apperrors.NewBadRequest("id should be a valid UUID", nil)
	}
	var input dtos.TUpdate{{.Name}}Dto
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		return apperrors.NewBadRequest("invalid request payload", nil)
	}
	input.ID = id
	res, err := c.{{.Name}}Service.Update(r.Context(), input)
	if err != nil {
		return apperrors.NewInternal("failed to update {{.LowerName}}", err)
	}
	return httputil.OK(w, res)
}

func (c *{{.Name}}Controller) Archive(w http.ResponseWriter, r *http.Request) error {
	id, err := utils.ParseIdStringIsValidUUID(mux.Vars(r)["id"])
	if err != nil {
		return apperrors.NewBadRequest("id should be a valid UUID", nil)
	}
	res, err := c.{{.Name}}Service.Archive(r.Context(), dtos.TArchive{{.Name}}Dto{ID: id})
	if err != nil {
		return apperrors.NewInternal("failed to archive {{.LowerName}}", err)
	}
	return httputil.OK(w, res)
}
`

func genRoutes(d scaffoldData) error {
	return writeTemplate(
		fmt.Sprintf("app/rest/routes/%s.routes.go", d.SnakeName),
		"routes", tplRoutes, d,
	)
}

var tplRoutes = `package routes

import (
	"github.com/gorilla/mux"
	"github.com/healtronlabs/gofasta/app/rest/controllers"
	"github.com/healtronlabs/gofasta/pkg/httputil"
)

func {{.Name}}Routes(r *mux.Router, c *controllers.{{.Name}}Controller) {
	r.HandleFunc("/{{.PluralSnake}}", httputil.Handle(c.List)).Methods("GET")
	r.HandleFunc("/{{.PluralSnake}}", httputil.Handle(c.Create)).Methods("POST")
	r.HandleFunc("/{{.PluralSnake}}/{id}", httputil.Handle(c.GetByID)).Methods("GET")
	r.HandleFunc("/{{.PluralSnake}}/{id}", httputil.Handle(c.Update)).Methods("PUT")
	r.HandleFunc("/{{.PluralSnake}}/{id}", httputil.Handle(c.Archive)).Methods("DELETE")
}
`

func genDTOs(d scaffoldData) error {
	return writeTemplate(
		fmt.Sprintf("app/dtos/%s.dtos.go", d.SnakeName),
		"dtos", tplDTOs, d,
	)
}

var tplDTOs = `package dtos

import (
	"time"

	"github.com/google/uuid"
)

// --- Response DTO (output) ---

type {{.Name}} struct {
	ID            uuid.UUID  ` + "`" + `json:"id"` + "`" + `
	RecordVersion int        ` + "`" + `json:"recordVersion"` + "`" + `
	CreatedAt     time.Time  ` + "`" + `json:"createdAt"` + "`" + `
	UpdatedAt     time.Time  ` + "`" + `json:"updatedAt"` + "`" + `
	IsActive      bool       ` + "`" + `json:"isActive"` + "`" + `
	IsDeletable   bool       ` + "`" + `json:"isDeletable"` + "`" + `
	DeletedAt     *time.Time ` + "`" + `json:"deletedAt,omitempty"` + "`" + `
{{- range .Fields}}
	{{.Name}} {{.GoType}} ` + "`" + `json:"{{.JSONName}}"` + "`" + `
{{- end}}
}

type T{{.Name}}ResponseDto struct {
	Data   *{{.Name}}            ` + "`" + `json:"data,omitempty"` + "`" + `
	Errors []*TCommonAPIErrorDto ` + "`" + `json:"errors,omitempty"` + "`" + `
}

type T{{.PluralName}}ResponseDto struct {
	Data       []*{{.Name}}            ` + "`" + `json:"data"` + "`" + `
	Pagination *TPaginationObjectDto   ` + "`" + `json:"pagination"` + "`" + `
}

// --- Input DTOs ---

type TCreate{{.Name}}Dto struct {
{{- range .Fields}}
	{{.Name}} {{.GoType}} ` + "`" + `json:"{{.JSONName}}" validate:"required"` + "`" + `
{{- end}}
}

type TUpdate{{.Name}}Dto struct {
	ID            uuid.UUID ` + "`" + `json:"id" validate:"required,uuid4_valid,does_record_exist_by_id_for_verification={{.PluralSnake}}"` + "`" + `
	RecordVersion int       ` + "`" + `json:"recordVersion" validate:"required,min=1"` + "`" + `
{{- range .Fields}}
	{{.Name}} *{{.GoType}} ` + "`" + `json:"{{.JSONName}},omitempty"` + "`" + `
{{- end}}
}

type TFind{{.Name}}ByIDDto struct {
	ID uuid.UUID ` + "`" + `json:"id" validate:"uuid4_valid,does_record_exist_by_id_for_verification={{.PluralSnake}}"` + "`" + `
}

type TArchive{{.Name}}Dto struct {
	ID uuid.UUID ` + "`" + `json:"id" validate:"uuid4_valid,does_record_exist_by_id_for_verification={{.PluralSnake}},is_record_deletable={{.PluralSnake}}"` + "`" + `
}

type {{.Name}}FiltersDto struct {
	Pagination *TPaginationInputDto ` + "`" + `json:"pagination,omitempty"` + "`" + `
	Sorting    *TSortingInputDto    ` + "`" + `json:"sorting,omitempty"` + "`" + `
}
`

func genGraphQL(d scaffoldData) error {
	return writeTemplate(
		fmt.Sprintf("app/graphql/schema/%s.gql", d.SnakeName),
		"graphql", tplGraphQL, d,
	)
}

var tplGraphQL = `# {{.Name}} GraphQL schema

type {{.Name}} {
  id: ID!
  recordVersion: Int!
  createdAt: DateTime!
  updatedAt: DateTime!
  isActive: Boolean!
  isDeletable: Boolean!
  deletedAt: DateTime
{{- range .Fields}}
  {{.JSONName}}: {{.GQLType}}!
{{- end}}
}

type T{{.PluralName}}ResponseDto {
  data: [{{.Name}}!]!
  pagination: TPaginationObjectDto!
}

type T{{.Name}}ResponseDto {
  data: {{.Name}}
  errors: [TCommonApiErrorDto]
}

extend type Query {
  findAll{{.PluralName}}(filters: {{.Name}}FiltersInput!): T{{.PluralName}}ResponseDto!
  find{{.Name}}ById(input: TFind{{.Name}}ByIdInput!): T{{.Name}}ResponseDto!
}

extend type Mutation {
  create{{.Name}}(input: TCreate{{.Name}}Input!): T{{.Name}}ResponseDto!
  update{{.Name}}(input: TUpdate{{.Name}}Input!): T{{.Name}}ResponseDto!
  archive{{.Name}}(input: TArchive{{.Name}}Input!): TCommonResponseDto!
}

input TFind{{.Name}}ByIdInput {
  id: ID!
}

input TArchive{{.Name}}Input {
  id: ID!
}

input TCreate{{.Name}}Input {
{{- range .Fields}}
  {{.JSONName}}: {{.GQLType}}!
{{- end}}
}

input TUpdate{{.Name}}Input {
  id: ID!
  recordVersion: Int!
{{- range .Fields}}
  {{.JSONName}}: {{.GQLType}}
{{- end}}
}

input {{.Name}}FiltersInput {
  pagination: TPaginationInputDto
  sorting: TSortingInputDto
}
`

func genWireProvider(d scaffoldData) error {
	return writeTemplate(
		fmt.Sprintf("app/di/providers/%s.go", d.SnakeName),
		"wire_provider", tplWireProvider, d,
	)
}

var tplWireProvider = `package providers

import (
	"github.com/google/wire"
	"github.com/healtronlabs/gofasta/app/repositories"
	repoInterfaces "github.com/healtronlabs/gofasta/app/repositories/interfaces"
	"github.com/healtronlabs/gofasta/app/rest/controllers"
	"github.com/healtronlabs/gofasta/app/services"
	svcInterfaces "github.com/healtronlabs/gofasta/app/services/interfaces"
)

// {{.Name}}Set provides {{.LowerName}} domain: repository, service, controller.
var {{.Name}}Set = wire.NewSet(
	repositories.New{{.Name}}Repository,
	wire.Bind(new(repoInterfaces.{{.Name}}RepositoryInterface), new(*repositories.{{.Name}}Repository)),
	services.New{{.Name}}Service,
	wire.Bind(new(svcInterfaces.{{.Name}}ServiceInterface), new(*services.{{.Name}}Service)),
	controllers.New{{.Name}}ControllerInstance,
)
`

func genMigration(d scaffoldData) error {
	upPath := fmt.Sprintf("db/migrations/%s_create_%s.up.sql", d.MigrationNum, d.PluralSnake)
	downPath := fmt.Sprintf("db/migrations/%s_create_%s.down.sql", d.MigrationNum, d.PluralSnake)

	if err := writeTemplate(upPath, "migration_up", tplMigrationUp, d); err != nil {
		return err
	}
	return writeTemplate(downPath, "migration_down", tplMigrationDown, d)
}

var tplMigrationUp = `CREATE TABLE IF NOT EXISTS {{.PluralSnake}} (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
{{- range .Fields}}
    {{.SnakeName}} {{.SQLType}},
{{- end}}
    is_active BOOLEAN NOT NULL DEFAULT true,
    is_deletable BOOLEAN NOT NULL DEFAULT true,
    record_version INT NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now(),
    deleted_at TIMESTAMP
);

CREATE TRIGGER update_{{.PluralSnake}}_updated_at
    BEFORE UPDATE ON {{.PluralSnake}}
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER increment_{{.PluralSnake}}_record_version
    BEFORE UPDATE ON {{.PluralSnake}}
    FOR EACH ROW EXECUTE FUNCTION increment_record_version();
`

var tplMigrationDown = `DROP TRIGGER IF EXISTS increment_{{.PluralSnake}}_record_version ON {{.PluralSnake}};
DROP TRIGGER IF EXISTS update_{{.PluralSnake}}_updated_at ON {{.PluralSnake}};
DROP TABLE IF EXISTS {{.PluralSnake}};
`
