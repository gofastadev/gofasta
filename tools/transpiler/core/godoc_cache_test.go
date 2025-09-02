package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/doc"
	"strings"
	"testing"
	"time"
)

func TestNewGoDocCache(t *testing.T) {
	t.Run("default config", func(t *testing.T) {
		cache := NewGoDocCache(nil)
		if cache == nil {
			t.Fatal("Expected non-nil cache")
		}
		if cache.config == nil {
			t.Fatal("Expected non-nil config")
		}
	})
	
	t.Run("custom config", func(t *testing.T) {
		config := &GoDocCacheConfig{
			MaxCacheEntries: 500,
			TTL:            10 * time.Minute,
			AllMode:        true,
			GenerateHTML:   false,
		}
		cache := NewGoDocCache(config)
		if cache.config.MaxCacheEntries != 500 {
			t.Errorf("Expected MaxCacheEntries=500, got %d", cache.config.MaxCacheEntries)
		}
		if !cache.config.AllMode {
			t.Error("Expected AllMode=true")
		}
	})
}

func TestGoDocExtractPackageDoc(t *testing.T) {
	cache := NewGoDocCache(nil)
	
	// Create test source files
	src := map[string][]byte{
		"main.go": []byte(`// Package testpkg provides test functionality.
package testpkg

import "fmt"

// Constant represents a constant value.
const Constant = 42

// Variable represents a package variable.
var Variable = "test"

// TestStruct is a test structure.
type TestStruct struct {
	// Field is a test field.
	Field string
}

// Method is a method on TestStruct.
func (t *TestStruct) Method() string {
	return t.Field
}

// TestFunc is a test function.
// It demonstrates documentation extraction.
func TestFunc(param string) string {
	return fmt.Sprintf("test: %s", param)
}
`),
		"example_test.go": []byte(`package testpkg_test

import "fmt"

func ExampleTestFunc() {
	result := TestFunc("example")
	fmt.Println(result)
	// Output: test: example
}
`),
	}
	
	t.Run("extract package", func(t *testing.T) {
		pkg, err := cache.ExtractPackageDoc("testpkg", src)
		if err != nil {
			t.Fatalf("Failed to extract package doc: %v", err)
		}
		
		if pkg.Name != "testpkg" {
			t.Errorf("Expected package name 'testpkg', got '%s'", pkg.Name)
		}
		
		if !strings.Contains(pkg.Doc, "test functionality") {
			t.Error("Expected package doc to contain 'test functionality'")
		}
		
		// Check constants
		if len(pkg.Consts) == 0 {
			t.Error("Expected constants in package")
		}
		
		// Check variables
		if len(pkg.Vars) == 0 {
			t.Error("Expected variables in package")
		}
		
		// Check types
		if len(pkg.Types) == 0 {
			t.Error("Expected types in package")
		}
		
		// Check functions
		if len(pkg.Funcs) == 0 {
			t.Error("Expected functions in package")
		}
	})
	
	t.Run("cache hit", func(t *testing.T) {
		// Second extraction should hit cache
		pkg2, err := cache.ExtractPackageDoc("testpkg", src)
		if err != nil {
			t.Fatal(err)
		}
		
		if pkg2.Name != "testpkg" {
			t.Error("Cached result differs")
		}
		
		stats := cache.GetStatistics()
		if stats["cache_hits"].(int64) < 1 {
			t.Error("Expected at least one cache hit")
		}
	})
}

func TestGetTypeDoc(t *testing.T) {
	cache := NewGoDocCache(nil)
	
	src := map[string][]byte{
		"types.go": []byte(`package testpkg

// MyType is a custom type.
type MyType struct {
	// Name is the name field.
	Name string ` + "`json:\"name\"`" + `
	// Age is the age field.
	Age int ` + "`json:\"age\"`" + `
}

// String returns string representation.
func (m *MyType) String() string {
	return m.Name
}

// GetAge returns the age.
func (m *MyType) GetAge() int {
	return m.Age
}
`),
	}
	
	pkg, err := cache.ExtractPackageDoc("testpkg", src)
	if err != nil {
		t.Fatal(err)
	}
	
	t.Run("get type doc", func(t *testing.T) {
		typeDoc := cache.GetTypeDoc(pkg, "MyType")
		if typeDoc == nil {
			t.Fatal("Expected type documentation")
		}
		
		if typeDoc.Name != "MyType" {
			t.Errorf("Expected type name 'MyType', got '%s'", typeDoc.Name)
		}
		
		if !strings.Contains(typeDoc.Doc, "custom type") {
			t.Error("Expected type doc to contain 'custom type'")
		}
		
		// Check methods
		if len(typeDoc.Methods) != 2 {
			t.Errorf("Expected 2 methods, got %d", len(typeDoc.Methods))
		}
		
		// Check fields
		if len(typeDoc.Fields) != 2 {
			t.Errorf("Expected 2 fields, got %d", len(typeDoc.Fields))
		}
	})
	
	t.Run("non-existent type", func(t *testing.T) {
		typeDoc := cache.GetTypeDoc(pkg, "NonExistent")
		if typeDoc != nil {
			t.Error("Expected nil for non-existent type")
		}
	})
}

func TestGetFuncDoc(t *testing.T) {
	cache := NewGoDocCache(nil)
	
	src := map[string][]byte{
		"funcs.go": []byte(`package testpkg

import "fmt"

// ProcessData processes the given data.
// It returns the processed result.
func ProcessData(data string) string {
	return fmt.Sprintf("processed: %s", data)
}

// Calculate performs a calculation.
func Calculate(a, b int) int {
	return a + b
}
`),
	}
	
	pkg, err := cache.ExtractPackageDoc("testpkg", src)
	if err != nil {
		t.Fatal(err)
	}
	
	t.Run("get func doc", func(t *testing.T) {
		funcDoc := cache.GetFuncDoc(pkg, "ProcessData")
		if funcDoc == nil {
			t.Fatal("Expected function documentation")
		}
		
		if funcDoc.Name != "ProcessData" {
			t.Errorf("Expected function name 'ProcessData', got '%s'", funcDoc.Name)
		}
		
		if !strings.Contains(funcDoc.Doc, "processes the given data") {
			t.Error("Expected function doc")
		}
	})
	
	t.Run("non-existent function", func(t *testing.T) {
		funcDoc := cache.GetFuncDoc(pkg, "NonExistent")
		if funcDoc != nil {
			t.Error("Expected nil for non-existent function")
		}
	})
}

func TestExportHTML(t *testing.T) {
	cache := NewGoDocCache(&GoDocCacheConfig{
		GenerateHTML: true,
	})
	
	src := map[string][]byte{
		"doc.go": []byte(`// Package example provides example functionality.
package example

// ExampleFunc is an example function.
func ExampleFunc() string {
	return "example"
}
`),
	}
	
	pkg, err := cache.ExtractPackageDoc("example", src)
	if err != nil {
		t.Fatal(err)
	}
	
	var buf bytes.Buffer
	err = cache.ExportHTML(pkg, &buf)
	if err != nil {
		t.Fatalf("Failed to export HTML: %v", err)
	}
	
	html := buf.String()
	if !strings.Contains(html, "Package example") {
		t.Error("Expected HTML to contain package name")
	}
	if !strings.Contains(html, "</html>") {
		t.Error("Expected valid HTML structure")
	}
}

func TestExportJSON(t *testing.T) {
	cache := NewGoDocCache(&GoDocCacheConfig{
		GenerateJSON: true,
	})
	
	src := map[string][]byte{
		"json.go": []byte(`package jsonpkg

const Version = "1.0.0"

var Config = "default"

type Data struct {
	Value string
}

func Process(d Data) string {
	return d.Value
}
`),
	}
	
	pkg, err := cache.ExtractPackageDoc("jsonpkg", src)
	if err != nil {
		t.Fatal(err)
	}
	
	var buf bytes.Buffer
	err = cache.ExportJSON(pkg, &buf)
	if err != nil {
		t.Fatalf("Failed to export JSON: %v", err)
	}
	
	// Parse JSON to verify structure
	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("Invalid JSON output: %v", err)
	}
	
	if result["name"] != "jsonpkg" {
		t.Errorf("Expected package name 'jsonpkg' in JSON")
	}
	
	if _, exists := result["constants"]; !exists {
		t.Error("Expected 'constants' in JSON")
	}
	
	if _, exists := result["variables"]; !exists {
		t.Error("Expected 'variables' in JSON")
	}
	
	if _, exists := result["types"]; !exists {
		t.Error("Expected 'types' in JSON")
	}
	
	if _, exists := result["functions"]; !exists {
		t.Error("Expected 'functions' in JSON")
	}
}

func TestExportMarkdown(t *testing.T) {
	cache := NewGoDocCache(nil)
	
	src := map[string][]byte{
		"markdown.go": []byte(`// Package mdpkg provides markdown test.
package mdpkg

import "fmt"

// Constant is a constant.
const Constant = 100

// Variable is a variable.
var Variable = "test"

// Type is a custom type.
type Type struct {
	// Field is a field.
	Field string
}

// Method is a method on Type.
func (t *Type) Method() string {
	return t.Field
}

// Function is a function.
func Function(param string) string {
	return fmt.Sprintf("result: %s", param)
}
`),
	}
	
	pkg, err := cache.ExtractPackageDoc("mdpkg", src)
	if err != nil {
		t.Fatal(err)
	}
	
	var buf bytes.Buffer
	err = cache.ExportMarkdown(pkg, &buf)
	if err != nil {
		t.Fatalf("Failed to export Markdown: %v", err)
	}
	
	markdown := buf.String()
	
	// Check markdown structure
	if !strings.Contains(markdown, "# Package mdpkg") {
		t.Error("Expected package header in Markdown")
	}
	
	if !strings.Contains(markdown, "## Constants") {
		t.Error("Expected Constants section")
	}
	
	if !strings.Contains(markdown, "## Variables") {
		t.Error("Expected Variables section")
	}
	
	if !strings.Contains(markdown, "## Functions") {
		t.Error("Expected Functions section")
	}
	
	if !strings.Contains(markdown, "## Types") {
		t.Error("Expected Types section")
	}
	
	if !strings.Contains(markdown, "```go") {
		t.Error("Expected code blocks in Markdown")
	}
}

func TestSearchDocumentation(t *testing.T) {
	cache := NewGoDocCache(nil)
	
	// Create multiple packages for searching
	packages := map[string]map[string][]byte{
		"pkg1": {
			"main.go": []byte(`package pkg1

// SearchableType is searchable.
type SearchableType struct{}

// SearchFunc is a searchable function.
func SearchFunc() {}`),
		},
		"pkg2": {
			"main.go": []byte(`package pkg2

// AnotherType with search keyword in doc.
type AnotherType struct{}

// NormalFunc is normal.
func NormalFunc() {}`),
		},
	}
	
	for path, src := range packages {
		if _, err := cache.ExtractPackageDoc(path, src); err != nil {
			t.Fatal(err)
		}
	}
	
	t.Run("search by keyword", func(t *testing.T) {
		results := cache.SearchDocumentation("search")
		if len(results) == 0 {
			t.Error("Expected search results")
		}
		
		// Should find SearchableType and SearchFunc
		foundType := false
		foundFunc := false
		for _, r := range results {
			if r.Name == "SearchableType" {
				foundType = true
			}
			if r.Name == "SearchFunc" {
				foundFunc = true
			}
		}
		
		if !foundType {
			t.Error("Expected to find SearchableType")
		}
		if !foundFunc {
			t.Error("Expected to find SearchFunc")
		}
	})
	
	t.Run("search non-existent", func(t *testing.T) {
		results := cache.SearchDocumentation("nonexistent")
		if len(results) != 0 {
			t.Error("Expected no results for non-existent keyword")
		}
	})
}

func TestGetExamples(t *testing.T) {
	cache := NewGoDocCache(nil)
	
	src := map[string][]byte{
		"example.go": []byte(`package examplepkg

func MyFunc() string {
	return "result"
}`),
		"example_test.go": []byte(`package examplepkg_test

import "fmt"

func ExampleMyFunc() {
	result := MyFunc()
	fmt.Println(result)
	// Output: result
}

func Example() {
	fmt.Println("package example")
	// Output: package example
}`),
	}
	
	pkg, err := cache.ExtractPackageDoc("examplepkg", src)
	if err != nil {
		t.Fatal(err)
	}
	
	t.Run("get function examples", func(t *testing.T) {
		examples := cache.GetExamples(pkg, "MyFunc")
		if len(examples) == 0 {
			t.Error("Expected examples for MyFunc")
		}
	})
	
	t.Run("get package examples", func(t *testing.T) {
		examples := cache.GetExamples(pkg, "")
		if len(examples) == 0 {
			t.Error("Expected package examples")
		}
	})
}

func TestBatchExtract(t *testing.T) {
	cache := NewGoDocCache(&GoDocCacheConfig{
		ConcurrentExtraction: true,
		ExtractionWorkers:   2,
	})
	
	packages := map[string]map[string][]byte{
		"pkg1": {
			"main.go": []byte(`package pkg1
func Func1() {}`),
		},
		"pkg2": {
			"main.go": []byte(`package pkg2
func Func2() {}`),
		},
		"pkg3": {
			"main.go": []byte(`package pkg3
func Func3() {}`),
		},
	}
	
	results := cache.BatchExtract(packages)
	if len(results) != len(packages) {
		t.Errorf("Expected %d results, got %d", len(packages), len(results))
	}
	
	for name, pkg := range results {
		if pkg.Name != name {
			t.Errorf("Expected package name %s, got %s", name, pkg.Name)
		}
	}
}

func TestGoDocCacheEviction(t *testing.T) {
	config := &GoDocCacheConfig{
		MaxCacheEntries: 2,
	}
	cache := NewGoDocCache(config)
	
	// Add multiple packages to trigger eviction
	for i := 0; i < 3; i++ {
		src := map[string][]byte{
			"main.go": []byte(fmt.Sprintf(`package pkg%d
func Func%d() {}`, i, i)),
		}
		cache.ExtractPackageDoc(fmt.Sprintf("pkg%d", i), src)
	}
	
	stats := cache.GetStatistics()
	packageCacheSize := stats["package_cache_size"].(int)
	if packageCacheSize > 2 {
		t.Errorf("Expected package cache size <= 2, got %d", packageCacheSize)
	}
}

func TestGetMode(t *testing.T) {
	t.Run("default mode", func(t *testing.T) {
		cache := NewGoDocCache(&GoDocCacheConfig{
			AllMode:    false,
			AllMethods: false,
		})
		mode := cache.getMode()
		if mode != 0 {
			t.Error("Expected default mode to be 0")
		}
	})
	
	t.Run("all mode", func(t *testing.T) {
		cache := NewGoDocCache(&GoDocCacheConfig{
			AllMode:    true,
			AllMethods: true,
		})
		mode := cache.getMode()
		if mode&doc.AllDecls == 0 {
			t.Error("Expected AllDecls flag to be set")
		}
		if mode&doc.AllMethods == 0 {
			t.Error("Expected AllMethods flag to be set")
		}
	})
}

func TestGoDocWarmupCache(t *testing.T) {
	cache := NewGoDocCache(nil)
	
	packages := map[string]map[string][]byte{
		"warmup1": {
			"main.go": []byte(`package warmup1`),
		},
		"warmup2": {
			"main.go": []byte(`package warmup2`),
		},
	}
	
	cache.WarmupCache(packages)
	
	stats := cache.GetStatistics()
	if stats["package_cache_size"].(int) != 2 {
		t.Errorf("Expected 2 packages in cache after warmup")
	}
}

func TestGoDocCacheClear(t *testing.T) {
	cache := NewGoDocCache(nil)
	
	// Add some data
	src := map[string][]byte{
		"main.go": []byte(`package test
func Test() {}`),
	}
	cache.ExtractPackageDoc("test", src)
	
	// Clear cache
	cache.Clear()
	
	stats := cache.GetStatistics()
	if stats["package_cache_size"].(int) != 0 {
		t.Error("Expected empty package cache after clear")
	}
	if stats["cache_hits"].(int64) != 0 {
		t.Error("Expected cache hits to be 0 after clear")
	}
	if stats["cache_misses"].(int64) != 0 {
		t.Error("Expected cache misses to be 0 after clear")
	}
}

func TestSynopsis(t *testing.T) {
	cache := NewGoDocCache(nil)
	
	src := map[string][]byte{
		"synopsis.go": []byte(`// Package synopsis provides functionality for testing.
// This is a longer description that should be truncated.
// It contains multiple lines of documentation.
package synopsis`),
	}
	
	_, err := cache.ExtractPackageDoc("synopsis", src)
	if err != nil {
		t.Fatal(err)
	}
	
	// Check cached synopsis
	cache.mu.RLock()
	cached, exists := cache.packageDocs[cache.generatePackageKey("synopsis", src)]
	cache.mu.RUnlock()
	
	if !exists {
		t.Fatal("Expected package to be cached")
	}
	
	if cached.Synopsis == "" {
		t.Error("Expected synopsis to be generated")
	}
	
	if !strings.Contains(cached.Synopsis, "functionality for testing") {
		t.Error("Synopsis should contain first sentence")
	}
}

func TestPrecomputeFormats(t *testing.T) {
	cache := NewGoDocCache(&GoDocCacheConfig{
		PrecomputeFormats: true,
	})
	
	src := map[string][]byte{
		"format.go": []byte(`// Package format tests precomputed formats.
package format

func FormatFunc() {}`),
	}
	
	_, err := cache.ExtractPackageDoc("format", src)
	if err != nil {
		t.Fatal(err)
	}
	
	// Check that formatted doc is precomputed
	cache.mu.RLock()
	cached, exists := cache.packageDocs[cache.generatePackageKey("format", src)]
	cache.mu.RUnlock()
	
	if !exists {
		t.Fatal("Expected package to be cached")
	}
	
	if cached.FormattedDoc == "" {
		t.Error("Expected formatted doc to be precomputed")
	}
}