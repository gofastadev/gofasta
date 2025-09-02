package core

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestNewBuildCache(t *testing.T) {
	t.Run("default config", func(t *testing.T) {
		cache := NewBuildCache(nil)
		if cache == nil {
			t.Fatal("Expected non-nil cache")
		}
		if cache.config == nil {
			t.Fatal("Expected non-nil config")
		}
	})
	
	t.Run("custom config", func(t *testing.T) {
		config := &BuildCacheConfig{
			MaxCacheEntries: 100,
			TTL:            5 * time.Minute,
			CustomTags:     []string{"test", "integration"},
		}
		cache := NewBuildCache(config)
		if cache.config.MaxCacheEntries != 100 {
			t.Errorf("Expected MaxCacheEntries=100, got %d", cache.config.MaxCacheEntries)
		}
		if len(cache.config.CustomTags) != 2 {
			t.Errorf("Expected 2 custom tags, got %d", len(cache.config.CustomTags))
		}
	})
}

func TestEvaluateConstraints(t *testing.T) {
	cache := NewBuildCache(nil)
	
	tests := []struct {
		name        string
		filename    string
		source      string
		expectMatch bool
	}{
		{
			name:     "no constraints",
			filename: "main.go",
			source: `package main

func main() {}`,
			expectMatch: true,
		},
		{
			name:     "OS constraint matching",
			filename: "file_" + runtime.GOOS + ".go",
			source:   `package main`,
			expectMatch: true,
		},
		{
			name:     "OS constraint not matching",
			filename: "file_nonexistent.go",
			source:   `package main`,
			expectMatch: false,
		},
		{
			name:     "build tag",
			filename: "tagged.go",
			source: `// +build integration

package main`,
			expectMatch: false, // No 'integration' tag set
		},
		{
			name:     "go:build directive",
			filename: "modern.go",
			source: `//go:build go1.18

package main`,
			expectMatch: true, // Assuming Go 1.18+
		},
	}
	
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			match, err := cache.EvaluateConstraints(test.filename, []byte(test.source))
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if match != test.expectMatch {
				t.Errorf("Expected match=%v, got %v", test.expectMatch, match)
			}
		})
	}
}

func TestBuildTags(t *testing.T) {
	cache := NewBuildCache(&BuildCacheConfig{
		CustomTags: []string{"test"},
	})
	
	t.Run("list tags", func(t *testing.T) {
		tags := cache.ListBuildTags()
		found := false
		for _, tag := range tags {
			if tag == "test" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected 'test' tag in list")
		}
	})
	
	t.Run("add tag", func(t *testing.T) {
		cache.AddBuildTag("integration")
		tags := cache.ListBuildTags()
		found := false
		for _, tag := range tags {
			if tag == "integration" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected 'integration' tag after adding")
		}
	})
	
	t.Run("remove tag", func(t *testing.T) {
		cache.RemoveBuildTag("test")
		tags := cache.ListBuildTags()
		found := false
		for _, tag := range tags {
			if tag == "test" {
				found = true
				break
			}
		}
		if found {
			t.Error("Did not expect 'test' tag after removal")
		}
	})
	
	t.Run("duplicate add", func(t *testing.T) {
		initialLen := len(cache.ListBuildTags())
		cache.AddBuildTag("integration")
		cache.AddBuildTag("integration") // Duplicate
		afterLen := len(cache.ListBuildTags())
		if afterLen != initialLen {
			t.Error("Duplicate tag should not be added")
		}
	})
}

func TestImportPackage(t *testing.T) {
	cache := NewBuildCache(nil)
	
	t.Run("import standard package", func(t *testing.T) {
		pkg, err := cache.ImportPackage("fmt", "", 0)
		if err != nil {
			t.Fatalf("Failed to import fmt: %v", err)
		}
		if pkg.Name != "fmt" {
			t.Errorf("Expected package name 'fmt', got '%s'", pkg.Name)
		}
		
		// Second import should hit cache
		pkg2, err := cache.ImportPackage("fmt", "", 0)
		if err != nil {
			t.Fatal(err)
		}
		if pkg2.Name != pkg.Name {
			t.Error("Cached result differs")
		}
		
		stats := cache.GetStatistics()
		if stats["cache_hits"].(int64) < 1 {
			t.Error("Expected at least one cache hit")
		}
	})
	
	t.Run("import non-existent package", func(t *testing.T) {
		_, err := cache.ImportPackage("nonexistent/package", "", 0)
		if err == nil {
			t.Error("Expected error for non-existent package")
		}
	})
}

func TestLoadPackage(t *testing.T) {
	// Create a temporary directory with Go files
	tmpDir, err := ioutil.TempDir("", "buildcache_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	
	// Create a simple Go file
	goFile := filepath.Join(tmpDir, "test.go")
	content := `package testpkg

import "fmt"

func Hello() {
	fmt.Println("Hello")
}`
	
	if err := ioutil.WriteFile(goFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	
	cache := NewBuildCache(nil)
	
	t.Run("load package", func(t *testing.T) {
		pkg, err := cache.LoadPackage(tmpDir)
		if err != nil {
			t.Fatalf("Failed to load package: %v", err)
		}
		if pkg.Name != "testpkg" {
			t.Errorf("Expected package name 'testpkg', got '%s'", pkg.Name)
		}
		if len(pkg.Dependencies) == 0 {
			t.Error("Expected dependencies")
		}
		
		// Check cache hit
		pkg2, err := cache.LoadPackage(tmpDir)
		if err != nil {
			t.Fatal(err)
		}
		if pkg2.Name != pkg.Name {
			t.Error("Cached result differs")
		}
	})
}

func TestCreateContext(t *testing.T) {
	cache := NewBuildCache(nil)
	
	t.Run("create custom context", func(t *testing.T) {
		ctx := cache.CreateContext("linux", "amd64", []string{"test"})
		if ctx.GOOS != "linux" {
			t.Errorf("Expected GOOS=linux, got %s", ctx.GOOS)
		}
		if ctx.GOARCH != "amd64" {
			t.Errorf("Expected GOARCH=amd64, got %s", ctx.GOARCH)
		}
		if len(ctx.BuildTags) != 1 || ctx.BuildTags[0] != "test" {
			t.Error("Expected build tag 'test'")
		}
		
		// Should be cached
		ctx2 := cache.CreateContext("linux", "amd64", []string{"test"})
		if ctx2 != ctx {
			t.Error("Expected cached context")
		}
	})
}

func TestMatchFile(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "matchfile_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	
	// Create test files
	files := map[string]string{
		"main.go":         `package main`,
		"main_test.go":    `package main`,
		"main_linux.go":   `package main`,
		"main_windows.go": `package main`,
	}
	
	for name, content := range files {
		path := filepath.Join(tmpDir, name)
		if err := ioutil.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	
	cache := NewBuildCache(nil)
	
	tests := []struct {
		filename    string
		shouldMatch bool
	}{
		{"main.go", true},
		{"main_test.go", false}, // Tests excluded by default
		{"main_" + runtime.GOOS + ".go", true},
		{"main_nonexistent.go", false},
	}
	
	for _, test := range tests {
		t.Run(test.filename, func(t *testing.T) {
			match, err := cache.MatchFile(tmpDir, test.filename)
			if err != nil && test.shouldMatch {
				t.Fatalf("Unexpected error: %v", err)
			}
			if match != test.shouldMatch {
				t.Errorf("Expected match=%v for %s, got %v", 
					test.shouldMatch, test.filename, match)
			}
		})
	}
}

func TestShouldBuild(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "shouldbuild_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	
	// Create a file with build constraints
	filename := filepath.Join(tmpDir, "constrained.go")
	content := `// +build !test

package main`
	
	if err := ioutil.WriteFile(filename, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	
	cache := NewBuildCache(nil)
	
	shouldBuild, err := cache.ShouldBuild(filename)
	if err != nil {
		t.Fatal(err)
	}
	if !shouldBuild {
		t.Error("Expected file to be built")
	}
}

func TestIsStandardPackage(t *testing.T) {
	cache := NewBuildCache(nil)
	
	tests := []struct {
		path     string
		expected bool
	}{
		{"fmt", true},
		{"encoding/json", true},
		{"net/http", true},
		{"C", true},
		{"github.com/user/package", false},
		{"example.com/package", false},
		{"mypackage", false},
	}
	
	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			result := cache.IsStandardPackage(test.path)
			if result != test.expected {
				t.Errorf("IsStandardPackage(%s) = %v, expected %v",
					test.path, result, test.expected)
			}
		})
	}
}

func TestBatchLoadPackages(t *testing.T) {
	// Create temporary directories with packages
	tmpBase, err := ioutil.TempDir("", "batch_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpBase)
	
	dirs := []string{}
	for i := 0; i < 3; i++ {
		dir := filepath.Join(tmpBase, fmt.Sprintf("pkg%d", i))
		if err := os.Mkdir(dir, 0755); err != nil {
			t.Fatal(err)
		}
		
		goFile := filepath.Join(dir, "main.go")
		content := fmt.Sprintf(`package pkg%d

func Func%d() {}`, i, i)
		
		if err := ioutil.WriteFile(goFile, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
		
		dirs = append(dirs, dir)
	}
	
	cache := NewBuildCache(&BuildCacheConfig{
		ConcurrentLoads: true,
		LoadWorkers:    2,
	})
	
	results := cache.BatchLoadPackages(dirs)
	if len(results) != len(dirs) {
		t.Errorf("Expected %d results, got %d", len(dirs), len(results))
	}
	
	for i, dir := range dirs {
		if pkg, exists := results[dir]; exists {
			expectedName := fmt.Sprintf("pkg%d", i)
			if pkg.Name != expectedName {
				t.Errorf("Expected package name %s, got %s", expectedName, pkg.Name)
			}
		} else {
			t.Errorf("Missing result for %s", dir)
		}
	}
}

func TestGoodOSArchFile(t *testing.T) {
	cache := NewBuildCache(nil)
	
	tests := []struct {
		filename string
		tags     map[string]bool
		expected bool
	}{
		{
			filename: "file.go",
			tags:     map[string]bool{},
			expected: true,
		},
		{
			filename: "file_" + runtime.GOOS + ".go",
			tags:     map[string]bool{runtime.GOOS: true},
			expected: true,
		},
		{
			filename: "file_nonexistent.go",
			tags:     map[string]bool{},
			expected: false,
		},
		{
			filename: "file_test.go",
			tags:     map[string]bool{},
			expected: false,
		},
	}
	
	for _, test := range tests {
		t.Run(test.filename, func(t *testing.T) {
			result := cache.GoodOSArchFile(test.filename, test.tags)
			if result != test.expected {
				t.Errorf("GoodOSArchFile(%s) = %v, expected %v",
					test.filename, result, test.expected)
			}
		})
	}
}

func TestFindImportPath(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "importpath_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	
	// Create a Go file
	goFile := filepath.Join(tmpDir, "test.go")
	content := `package testpkg

func Test() {}`
	
	if err := ioutil.WriteFile(goFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	
	cache := NewBuildCache(nil)
	
	importPath, err := cache.FindImportPath(tmpDir)
	if err != nil {
		t.Fatalf("Failed to find import path: %v", err)
	}
	
	// Import path might be empty for local packages
	t.Logf("Import path: %s", importPath)
}

func TestBuildCacheEviction(t *testing.T) {
	config := &BuildCacheConfig{
		MaxCacheEntries: 2,
	}
	cache := NewBuildCache(config)
	
	// Add multiple constraint evaluations to trigger eviction
	for i := 0; i < 3; i++ {
		filename := fmt.Sprintf("file%d.go", i)
		source := fmt.Sprintf("package pkg%d", i)
		cache.EvaluateConstraints(filename, []byte(source))
	}
	
	stats := cache.GetStatistics()
	constraintCacheSize := stats["constraint_cache_size"].(int)
	if constraintCacheSize > 2 {
		t.Errorf("Expected constraint cache size <= 2, got %d", constraintCacheSize)
	}
}

func TestGetPackageDependencies(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "deps_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	
	// Create a Go file with imports
	goFile := filepath.Join(tmpDir, "main.go")
	content := `package main

import (
	"fmt"
	"strings"
)

func main() {
	fmt.Println(strings.ToUpper("hello"))
}`
	
	if err := ioutil.WriteFile(goFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	
	cache := NewBuildCache(nil)
	
	deps, err := cache.GetPackageDependencies(tmpDir)
	if err != nil {
		t.Fatalf("Failed to get dependencies: %v", err)
	}
	
	// Should have fmt and strings as dependencies
	hasFmt := false
	hasStrings := false
	for _, dep := range deps {
		if dep == "fmt" {
			hasFmt = true
		}
		if dep == "strings" {
			hasStrings = true
		}
	}
	
	if !hasFmt {
		t.Error("Expected 'fmt' in dependencies")
	}
	if !hasStrings {
		t.Error("Expected 'strings' in dependencies")
	}
}

func TestBuildWarmupCache(t *testing.T) {
	cache := NewBuildCache(&BuildCacheConfig{
		CacheStdlib: true,
	})
	
	// Create some test directories
	tmpDir, err := ioutil.TempDir("", "warmup_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	
	goFile := filepath.Join(tmpDir, "test.go")
	if err := ioutil.WriteFile(goFile, []byte("package test"), 0644); err != nil {
		t.Fatal(err)
	}
	
	cache.WarmupCache([]string{tmpDir})
	
	// Check that package is cached
	stats := cache.GetStatistics()
	if stats["package_cache_size"].(int) == 0 {
		t.Error("Expected packages in cache after warmup")
	}
}

func TestBuildCacheClear(t *testing.T) {
	cache := NewBuildCache(nil)
	
	// Add some data
	cache.EvaluateConstraints("test.go", []byte("package test"))
	cache.ImportPackage("fmt", "", 0)
	cache.AddBuildTag("test")
	
	// Clear cache
	cache.Clear()
	
	stats := cache.GetStatistics()
	if stats["constraint_cache_size"].(int) != 0 {
		t.Error("Expected empty constraint cache after clear")
	}
	if stats["import_cache_size"].(int) != 0 {
		t.Error("Expected empty import cache after clear")
	}
	if stats["cache_hits"].(int64) != 0 {
		t.Error("Expected cache hits to be 0 after clear")
	}
}

func TestGetFileInfo(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "fileinfo_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	
	filename := filepath.Join(tmpDir, "test.go")
	content := `package test

import "fmt"

func Test() {
	fmt.Println("test")
}`
	
	if err := ioutil.WriteFile(filename, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	
	cache := NewBuildCache(nil)
	
	// Trigger file caching
	cache.MatchFile(tmpDir, "test.go")
	
	info := cache.GetFileInfo(filename)
	if info == nil {
		t.Fatal("Expected file info to be cached")
	}
	
	if info.Package != "test" {
		t.Errorf("Expected package 'test', got '%s'", info.Package)
	}
	
	if len(info.Imports) == 0 {
		t.Error("Expected imports in file info")
	}
}