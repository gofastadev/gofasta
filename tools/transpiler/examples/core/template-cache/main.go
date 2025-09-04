// Example demonstrating Phase 1.2b: text/template with compiled template cache
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/healtronlabs/gofasta/tools/transpiler/core"
)

func main() {
	fmt.Println("=== Phase 1.2b: Template Cache Example ===")
	fmt.Println()

	// Create template cache with custom configuration
	config := &core.TemplateCacheConfig{
		MaxTemplates:      100,
		MaxCacheSizeMB:    50,
		EnableDedup:       true,
		EnableMetrics:     true,
		PrecompileOnAdd:   true,
		ConcurrentCompile: true,
		CompileWorkers:    4,
	}
	tc := core.NewTemplateCache(config)

	// Example 1: Basic template compilation and execution
	fmt.Println("1. Basic template compilation and execution...")

	err := tc.AddTemplate("greeting", "Hello, {{.Name}}! Welcome to {{.Place}}.")
	if err != nil {
		log.Fatal(err)
	}

	result, err := tc.Execute("greeting", map[string]string{
		"Name":  "Developer",
		"Place": "Gofasta",
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Result: %s\n", result)
	fmt.Println()

	// Example 2: Template with functions
	fmt.Println("2. Templates with built-in functions...")

	templates := map[string]string{
		"upper_case":  "{{upper .text}}",
		"formatted":   "{{sprintf \"User: %s, ID: %d\" .name .id}}",
		"conditional": "Status: {{if .active}}Active{{else}}Inactive{{end}}",
		"list":        "Items:\n{{range .items}}- {{.}}\n{{end}}",
		"quoted":      "{{quote .message}}",
	}

	for name, tmpl := range templates {
		err := tc.AddTemplate(name, tmpl)
		if err != nil {
			log.Fatal(err)
		}
	}

	// Execute templates with functions
	upperResult, _ := tc.Execute("upper_case", map[string]string{"text": "hello world"})
	fmt.Printf("Upper case: %s\n", upperResult)

	formattedResult, _ := tc.Execute("formatted", map[string]interface{}{
		"name": "Alice",
		"id":   12345,
	})
	fmt.Printf("Formatted: %s\n", formattedResult)

	activeResult, _ := tc.Execute("conditional", map[string]bool{"active": true})
	inactiveResult, _ := tc.Execute("conditional", map[string]bool{"active": false})
	fmt.Printf("Active: %s\n", activeResult)
	fmt.Printf("Inactive: %s\n", inactiveResult)

	listResult, _ := tc.Execute("list", map[string][]string{
		"items": {"First", "Second", "Third"},
	})
	fmt.Printf("List:\n%s", listResult)

	quotedResult, _ := tc.Execute("quoted", map[string]string{
		"message": "This is a \"quoted\" message",
	})
	fmt.Printf("Quoted: %s\n", quotedResult)
	fmt.Println()

	// Example 3: Template sets for related templates
	fmt.Println("3. Template sets for HTML generation...")

	htmlTemplates := map[string]string{
		"layout": `<!DOCTYPE html>
<html>
<head><title>{{.Title}}</title></head>
<body>
{{template "header" .}}
{{template "content" .}}
{{template "footer" .}}
</body>
</html>`,
		"header":  `<header><h1>{{.Title}}</h1></header>`,
		"content": `<main>{{.Content}}</main>`,
		"footer":  `<footer>© {{.Year}} {{.Company}}</footer>`,
	}

	err = tc.CreateSet("webpage", htmlTemplates)
	if err != nil {
		log.Fatal(err)
	}

	pageData := map[string]interface{}{
		"Title":   "Gofasta Template Demo",
		"Content": "This is a demonstration of template sets.",
		"Year":    2024,
		"Company": "HealtronLabs",
	}

	// Execute individual templates from the set
	headerHTML, _ := tc.ExecuteSet("webpage", "header", pageData)
	fmt.Println("Header HTML:")
	fmt.Println(headerHTML)
	fmt.Println()

	// Example 4: Template deduplication
	fmt.Println("4. Template deduplication demonstration...")

	// Add identical templates with different names
	identical := "This is {{.value}}"
	tc.AddTemplate("template1", identical)
	tc.AddTemplate("template2", identical)
	tc.AddTemplate("template3", identical)

	// Due to deduplication, they should share the same underlying template
	stats := tc.GetStatistics()
	fmt.Printf("Total templates: %d\n", stats["template_count"])
	fmt.Printf("Unique templates: %d (deduplication working)\n", stats["unique_templates"])
	fmt.Println()

	// Example 5: Batch compilation
	fmt.Println("5. Batch template compilation...")

	batchTemplates := make(map[string]string)
	for i := 1; i <= 10; i++ {
		name := fmt.Sprintf("batch_%d", i)
		tmpl := fmt.Sprintf("Batch template #%d: {{.value}}", i)
		batchTemplates[name] = tmpl
	}

	start := time.Now()
	err = tc.BatchCompile(batchTemplates)
	if err != nil {
		log.Fatal(err)
	}
	duration := time.Since(start)

	fmt.Printf("✓ Compiled %d templates in %v\n", len(batchTemplates), duration)
	fmt.Println()

	// Example 6: Template with metadata
	fmt.Println("6. Templates with metadata...")

	metadata := map[string]interface{}{
		"version":     "2.0",
		"author":      "Gofasta Team",
		"description": "API response template",
		"created":     time.Now().Format(time.RFC3339),
	}

	apiTemplate := `{
	"status": "{{.status}}",
	"message": "{{.message}}",
	"data": {{.data}},
	"timestamp": "{{.timestamp}}"
}`

	err = tc.AddTemplateWithMetadata("api_response", apiTemplate, metadata)
	if err != nil {
		log.Fatal(err)
	}

	apiData := map[string]interface{}{
		"status":    "success",
		"message":   "Operation completed",
		"data":      "null",
		"timestamp": time.Now().Format(time.RFC3339),
	}

	apiResult, _ := tc.Execute("api_response", apiData)
	fmt.Println("API Response template:")
	fmt.Println(apiResult)
	fmt.Println()

	// Example 7: Custom delimiters
	fmt.Println("7. Templates with custom delimiters...")

	customTC := core.NewTemplateCache(&core.TemplateCacheConfig{
		Delims: [2]string{"[[", "]]"},
	})

	customTC.AddTemplate("custom", "Hello [[.name]], using custom delimiters!")
	customResult, _ := customTC.Execute("custom", map[string]string{"name": "Developer"})
	fmt.Printf("Custom delimiters: %s\n", customResult)
	fmt.Println()

	// Example 8: Helper functions demonstration
	fmt.Println("8. Template helper functions...")

	helperTemplates := map[string]string{
		"indent":    "{{indent 4 .text}}",
		"wrap":      "{{wrap 20 .longText}}",
		"default":   "{{default \"N/A\" .value}}",
		"ternary":   "Result: {{ternary .condition \"Yes\" \"No\"}}",
		"join":      "{{join \", \" .items}}",
		"basename":  "{{basename .path}}",
		"dict_demo": "{{$d := dict \"key1\" \"value1\" \"key2\" \"value2\"}}Dict: {{$d.key1}}, {{$d.key2}}",
	}

	for name, tmpl := range helperTemplates {
		tc.AddTemplate(name, tmpl)
	}

	indentResult, _ := tc.Execute("indent", map[string]string{
		"text": "Line 1\nLine 2\nLine 3",
	})
	fmt.Println("Indented text:")
	fmt.Println(indentResult)

	wrapResult, _ := tc.Execute("wrap", map[string]string{
		"longText": "This is a very long text that needs to be wrapped at specific width boundaries",
	})
	fmt.Println("Wrapped text:")
	fmt.Println(wrapResult)

	defaultResult, _ := tc.Execute("default", map[string]string{})
	fmt.Printf("Default value: %s\n", defaultResult)

	ternaryResult, _ := tc.Execute("ternary", map[string]bool{"condition": true})
	fmt.Printf("Ternary: %s\n", ternaryResult)

	joinResult, _ := tc.Execute("join", map[string][]string{
		"items": {"apple", "banana", "orange"},
	})
	fmt.Printf("Joined: %s\n", joinResult)

	basenameResult, _ := tc.Execute("basename", map[string]string{
		"path": "/usr/local/bin/gofasta",
	})
	fmt.Printf("Basename: %s\n", basenameResult)

	dictResult, _ := tc.Execute("dict_demo", nil)
	fmt.Printf("Dict demo: %s\n", dictResult)
	fmt.Println()

	// Show final statistics
	fmt.Println("=== Cache Statistics ===")
	finalStats := tc.GetStatistics()
	fmt.Printf("Total templates: %d\n", finalStats["template_count"])
	fmt.Printf("Template sets: %d\n", finalStats["set_count"])
	fmt.Printf("Cache hits: %d\n", finalStats["cache_hits"])
	fmt.Printf("Cache misses: %d\n", finalStats["cache_misses"])
	fmt.Printf("Hit rate: %.2f%%\n", finalStats["hit_rate"])
	fmt.Printf("Compilations: %d\n", finalStats["compilations"])
	fmt.Printf("Executions: %d\n", finalStats["executions"])
	fmt.Printf("Cache size: %.2f MB\n", finalStats["cache_size_mb"])
	fmt.Printf("Deduplication enabled: %v\n", finalStats["dedup_enabled"])

	fmt.Println()
	fmt.Println("✅ Phase 1.2b demonstration completed successfully!")
}
