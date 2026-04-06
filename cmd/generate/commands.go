package generate

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Cmd is the parent "generate" command, registered on rootCmd by the thin stub.
var Cmd = &cobra.Command{
	Use:     "generate",
	Short:   "Generate boilerplate code",
	Aliases: []string{"g"},
}

// WireCmd is a standalone command to regenerate Wire DI code.
var WireCmd = &cobra.Command{
	Use:   "wire",
	Short: "Run Wire to regenerate dependency injection code",
	RunE: func(cmd *cobra.Command, args []string) error {
		return RunWire(ScaffoldData{})
	},
}

func init() {
	Cmd.AddCommand(scaffoldCmd)
	Cmd.AddCommand(modelCmd)
	Cmd.AddCommand(repositoryCmd)
	Cmd.AddCommand(serviceCmd)
	Cmd.AddCommand(controllerCmd)
	Cmd.AddCommand(dtoCmd)
	Cmd.AddCommand(migrationCmd)
	Cmd.AddCommand(routeCmd)
	Cmd.AddCommand(resolverCmd)
	Cmd.AddCommand(providerCmd)
}

// --- Composable step chain builders ---
// Pattern: generate ALL files first, then patch, then run tools.
// writeTemplate skips existing files, so composition is idempotent.

func modelSteps() []Step {
	return []Step{
		{"model", GenModel},
		{"migration", GenMigration},
	}
}

func dtoSteps() []Step {
	return []Step{
		{"DTOs", GenDTOs},
	}
}

func migrationSteps() []Step {
	return []Step{
		{"migration", GenMigration},
	}
}

func repositorySteps() []Step {
	return []Step{
		// Files
		{"model", GenModel},
		{"migration", GenMigration},
		{"repository interface", GenRepoInterface},
		{"repository", GenRepo},
	}
}

func serviceSteps() []Step {
	return []Step{
		// Files (all generated before any patching)
		{"model", GenModel},
		{"migration", GenMigration},
		{"repository interface", GenRepoInterface},
		{"repository", GenRepo},
		{"service interface", GenSvcInterface},
		{"service", GenSvc},
		{"DTOs", GenDTOs},
		{"Wire provider", GenWireProvider},
		// Patch
		{"auto-wire: container", PatchContainer},
		{"auto-wire: wire.go", PatchWireFile},
		{"auto-wire: resolver", PatchResolver},
		// Regenerate
		{"regenerate Wire", RunWire},
	}
}

func controllerSteps() []Step {
	return []Step{
		// Files (ALL files before any patching)
		{"model", GenModel},
		{"migration", GenMigration},
		{"repository interface", GenRepoInterface},
		{"repository", GenRepo},
		{"service interface", GenSvcInterface},
		{"service", GenSvc},
		{"DTOs", GenDTOs},
		{"Wire provider", GenWireProvider},
		{"controller", GenController},
		{"routes", GenRoutes},
		// Patch
		{"auto-wire: container", PatchContainer},
		{"auto-wire: wire.go", PatchWireFile},
		{"auto-wire: resolver", PatchResolver},
		{"auto-wire: route config", PatchRouteConfig},
		{"auto-wire: serve.go", PatchServeFile},
		// Regenerate
		{"regenerate Wire", RunWire},
	}
}

func scaffoldSteps() []Step {
	return []Step{
		// Files (ALL files before any patching)
		{"model", GenModel},
		{"migration", GenMigration},
		{"repository interface", GenRepoInterface},
		{"repository", GenRepo},
		{"service interface", GenSvcInterface},
		{"service", GenSvc},
		{"DTOs", GenDTOs},
		{"Wire provider", GenWireProvider},
		{"controller", GenController},
		{"routes", GenRoutes},
		{"GraphQL schema", GenGraphQL},
		// Patch
		{"auto-wire: container", PatchContainer},
		{"auto-wire: wire.go", PatchWireFile},
		{"auto-wire: resolver", PatchResolver},
		{"auto-wire: route config", PatchRouteConfig},
		{"auto-wire: serve.go", PatchServeFile},
		// Regenerate
		{"regenerate Wire", RunWire},
		{"regenerate gqlgen", RunGqlgen},
	}
}

func routeSteps() []Step {
	return []Step{
		{"routes", GenRoutes},
	}
}

func resolverSteps() []Step {
	return []Step{
		{"auto-wire: resolver", GenResolver},
	}
}

func providerSteps() []Step {
	return []Step{
		{"Wire provider", GenWireProvider},
		{"auto-wire: container", PatchContainer},
		{"auto-wire: wire.go", PatchWireFile},
		// Wire not run here — run `gofasta wire` after all dependent files exist
	}
}

// --- Helper to build data from CLI args ---

func buildFromArgs(args []string) ScaffoldData {
	return BuildScaffoldData(args[0], ParseFields(args[1:]))
}

// --- Cobra command definitions ---

var scaffoldCmd = &cobra.Command{
	Use:   "scaffold [Name] [field:type ...]",
	Short: "Generate a full resource with auto-wiring (model, repo, service, controller, routes, DTOs, GraphQL, migration, DI)",
	Long: `Generate all files for a new resource domain and auto-wire them into the framework.
No manual wiring needed — the developer only writes business logic.

Examples:
  gofasta generate scaffold Product name:string price:float description:text
  gofasta g s BlogPost title:string body:text published:bool

Supported field types: string, text, int, float, bool, uuid, time`,
	Aliases: []string{"s"},
	Args:    cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		d := buildFromArgs(args)
		d.IncludeController = true
		if err := RunSteps(d, scaffoldSteps()); err != nil {
			return err
		}
		fmt.Printf("\nScaffold complete for %s. All files generated and wired.\n", d.Name)
		fmt.Printf("Run migrations: gofasta migrate up\n")
		fmt.Printf("Write business logic: app/services/%s.service.go\n", d.SnakeName)
		return nil
	},
}

var modelCmd = &cobra.Command{
	Use:   "model [Name] [field:type ...]",
	Short: "Generate model + migration",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return RunSteps(buildFromArgs(args), modelSteps())
	},
}

var repositoryCmd = &cobra.Command{
	Use:     "repository [Name] [field:type ...]",
	Short:   "Generate model + migration + repository interface + repository",
	Aliases: []string{"repo"},
	Args:    cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return RunSteps(buildFromArgs(args), repositorySteps())
	},
}

var serviceCmd = &cobra.Command{
	Use:     "service [Name] [field:type ...]",
	Short:   "Generate model + repo + service + DTOs + Wire provider, fully auto-wired",
	Aliases: []string{"svc"},
	Args:    cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return RunSteps(buildFromArgs(args), serviceSteps())
	},
}

var controllerCmd = &cobra.Command{
	Use:     "controller [Name] [field:type ...]",
	Short:   "Generate everything up to controller + routes, fully auto-wired",
	Aliases: []string{"ctrl"},
	Args:    cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		d := buildFromArgs(args)
		d.IncludeController = true
		return RunSteps(d, controllerSteps())
	},
}

var dtoCmd = &cobra.Command{
	Use:   "dto [Name] [field:type ...]",
	Short: "Generate DTOs only (standalone)",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return RunSteps(buildFromArgs(args), dtoSteps())
	},
}

var migrationCmd = &cobra.Command{
	Use:   "migration [Name] [field:type ...]",
	Short: "Generate SQL migration files only",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return RunSteps(buildFromArgs(args), migrationSteps())
	},
}

var routeCmd = &cobra.Command{
	Use:   "route [Name]",
	Short: "Generate route file only (assumes controller exists)",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return RunSteps(buildFromArgs(args), routeSteps())
	},
}

var resolverCmd = &cobra.Command{
	Use:   "resolver [Name]",
	Short: "Patch GraphQL resolver to add service dependency (assumes service interface exists)",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return RunSteps(buildFromArgs(args), resolverSteps())
	},
}

var providerCmd = &cobra.Command{
	Use:   "provider [Name]",
	Short: "Generate Wire provider + auto-wire container and wire.go + regenerate Wire",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return RunSteps(buildFromArgs(args), providerSteps())
	},
}
