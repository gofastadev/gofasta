// Example demonstrating Phase 1.2a: go/printer with template pre-compilation
package main

import (
	"fmt"
	"go/ast"
	"go/printer"
	"log"

	"github.com/healtronlabs/gofasta/tools/transpiler/core"
)

func main() {
	fmt.Println("=== Phase 1.2a: Printer Cache Example ===")
	fmt.Println()

	// Create printer cache with default config
	config := core.DefaultPrinterCacheConfig()
	config.EnableMetrics = true
	pc := core.NewPrinterCache(config)

	// Example 1: Pre-compile templates for code generation
	fmt.Println("1. Pre-compiling code generation templates...")
	
	// Template for generating a struct
	structTemplate := `type {{.Name}} struct {
{{range .Fields}}	{{.Name}} {{.Type}}
{{end}}}`
	
	err := pc.CompileTemplate("struct", structTemplate)
	if err != nil {
		log.Fatal(err)
	}

	// Template for generating a method
	methodTemplate := `func ({{.Receiver}} *{{.Type}}) {{.Name}}({{range $i, $p := .Params}}{{if $i}}, {{end}}{{$p.Name}} {{$p.Type}}{{end}}) {{.ReturnType}} {
	{{.Body}}
}`
	
	err = pc.CompileTemplate("method", methodTemplate)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("✓ Templates compiled successfully")
	fmt.Println()

	// Example 2: Execute templates to generate code
	fmt.Println("2. Generating code from templates...")
	
	// Generate a User struct
	structData := map[string]interface{}{
		"Name": "User",
		"Fields": []map[string]string{
			{"Name": "ID", "Type": "int64"},
			{"Name": "Name", "Type": "string"},
			{"Name": "Email", "Type": "string"},
			{"Name": "Active", "Type": "bool"},
		},
	}
	
	userStruct, err := pc.ExecuteTemplate("struct", structData)
	if err != nil {
		log.Fatal(err)
	}
	
	fmt.Println("Generated struct:")
	fmt.Println(userStruct)
	fmt.Println()

	// Generate a method
	methodData := map[string]interface{}{
		"Receiver":   "u",
		"Type":       "User",
		"Name":       "IsValid",
		"Params":     []map[string]string{},
		"ReturnType": "bool",
		"Body":       "return u.ID > 0 && u.Email != \"\"",
	}
	
	method, err := pc.ExecuteTemplate("method", methodData)
	if err != nil {
		log.Fatal(err)
	}
	
	fmt.Println("Generated method:")
	fmt.Println(method)
	fmt.Println()

	// Example 3: Print AST nodes
	fmt.Println("3. Printing AST nodes...")
	
	// Create some AST nodes
	ident := &ast.Ident{Name: "MyVariable"}
	literal := &ast.BasicLit{
		Kind:  1, // token.STRING
		Value: `"Hello, World!"`,
	}
	
	// Print nodes
	identStr, err := pc.PrintNode(ident)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Identifier: %s\n", identStr)
	
	literalStr, err := pc.PrintNode(literal)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Literal: %s\n", literalStr)
	fmt.Println()

	// Example 4: Batch printing
	fmt.Println("4. Batch printing multiple nodes...")
	
	nodes := []ast.Node{
		&ast.Ident{Name: "First"},
		&ast.Ident{Name: "Second"},
		&ast.Ident{Name: "Third"},
		&ast.BasicLit{Value: `"Fourth"`},
	}
	
	results, err := pc.BatchPrint(nodes)
	if err != nil {
		log.Fatal(err)
	}
	
	for i, result := range results {
		fmt.Printf("Node %d: %s\n", i+1, result)
	}
	fmt.Println()

	// Example 5: Format source code
	fmt.Println("5. Formatting source code...")
	
	unformatted := []byte(`package main
import("fmt"
"strings")
func main(){fmt.Println("hello")}`)
	
	formatted, err := pc.FormatSource(unformatted)
	if err != nil {
		log.Fatal(err)
	}
	
	fmt.Println("Formatted code:")
	fmt.Println(string(formatted))
	fmt.Println()

	// Example 6: Custom printer configuration
	fmt.Println("6. Custom printer configuration...")
	
	customConfig := &core.PrinterCacheConfig{
		Mode:     printer.TabIndent | printer.UseSpaces,
		Tabwidth: 4,
		PrecompileTemplates: true,
		MaxTemplates:        50,
		BufferPoolSize:      25,
		EnableMetrics:       true,
	}
	
	customPC := core.NewPrinterCache(customConfig)
	
	// Add template with custom functions
	tmpl := `{{comment .Description}}
func {{lower .Name}}() {
	// Implementation
}`
	
	err = customPC.CompileTemplate("custom_func", tmpl)
	if err != nil {
		log.Fatal(err)
	}
	
	funcData := map[string]string{
		"Description": "This is a custom function",
		"Name":        "MyFunction",
	}
	
	result, err := customPC.ExecuteTemplate("custom_func", funcData)
	if err != nil {
		log.Fatal(err)
	}
	
	fmt.Println("Custom formatted function:")
	fmt.Println(result)
	fmt.Println()

	// Show statistics
	fmt.Println("=== Cache Statistics ===")
	stats := pc.GetStatistics()
	fmt.Printf("Templates compiled: %d\n", stats["template_count"])
	fmt.Printf("Template hits: %d\n", stats["template_hits"])
	fmt.Printf("Template misses: %d\n", stats["template_misses"])
	fmt.Printf("Hit rate: %.2f%%\n", stats["hit_rate"])
	fmt.Printf("Print operations: %d\n", stats["print_operations"])
	fmt.Printf("Total bytes processed: %d\n", stats["total_bytes"])
	fmt.Printf("Average bytes per operation: %d\n", stats["avg_bytes_per_op"])

	fmt.Println()
	fmt.Println("✅ Phase 1.2a demonstration completed successfully!")
}