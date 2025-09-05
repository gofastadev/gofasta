// Example demonstrating go/build with build constraint cache (Phase 1.2h)
package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/healtronlabs/gofasta/transpiler/core"
)

func main() {
	// Create build cache with custom configuration
	config := &core.BuildCacheConfig{
		MaxCacheEntries: 1000,
		EnableMetrics:   true,
		CustomTags:      []string{"integration", "debug"},
		IgnoreVendor:    true,
		ConcurrentLoads: true,
		LoadWorkers:     4,
		CacheStdlib:     true,
		PreloadStdlib:   false, // We'll manually load some packages
	}
	cache := core.NewBuildCache(config)

	fmt.Println("=== Build Cache Example ===\n")

	// 1. Build constraints evaluation
	fmt.Println("1. Evaluating build constraints:")

	// Create test files with different constraints
	testFiles := map[string]string{
		"simple.go": `package main

func Simple() {}`,

		"tagged.go": `// +build integration

package main

func Integration() {}`,

		"os_specific.go": `// +build linux darwin

package main

func UnixOnly() {}`,

		"modern.go": `//go:build go1.18 && !windows

package main

func Modern() {}`,
	}

	for filename, content := range testFiles {
		shouldBuild, err := cache.EvaluateConstraints(filename, []byte(content))
		if err != nil {
			log.Printf("Error evaluating %s: %v", filename, err)
			continue
		}
		fmt.Printf("   %s: build=%v\n", filename, shouldBuild)
	}

	// 2. Build tags management
	fmt.Println("\n2. Build tags:")
	tags := cache.ListBuildTags()
	fmt.Printf("   Current tags: %v\n", tags)

	// Add a new tag
	cache.AddBuildTag("experimental")
	fmt.Printf("   After adding 'experimental': %v\n", cache.ListBuildTags())

	// Now re-evaluate with new tag
	experimentalFile := `// +build experimental

package main

func Experimental() {}`

	shouldBuild, _ := cache.EvaluateConstraints("experimental.go", []byte(experimentalFile))
	fmt.Printf("   experimental.go builds with 'experimental' tag: %v\n", shouldBuild)

	// Remove tag
	cache.RemoveBuildTag("experimental")
	shouldBuild, _ = cache.EvaluateConstraints("experimental.go", []byte(experimentalFile))
	fmt.Printf("   experimental.go builds without 'experimental' tag: %v\n", shouldBuild)

	// 3. Import standard packages
	fmt.Println("\n3. Importing standard packages:")
	stdPackages := []string{"fmt", "strings", "net/http", "encoding/json"}

	for _, pkg := range stdPackages {
		p, err := cache.ImportPackage(pkg, "", 0)
		if err != nil {
			log.Printf("Error importing %s: %v", pkg, err)
			continue
		}
		fmt.Printf("   %s: %s (files: %d)\n", p.ImportPath, p.Name, len(p.GoFiles))
	}

	// Import again to demonstrate caching
	fmt.Println("\n4. Cache performance (importing same packages):")
	for _, pkg := range stdPackages[:2] {
		_, err := cache.ImportPackage(pkg, "", 0)
		if err == nil {
			fmt.Printf("   %s: imported (should hit cache)\n", pkg)
		}
	}

	// 5. Create custom build contexts
	fmt.Println("\n5. Custom build contexts:")
	contexts := []struct {
		goos   string
		goarch string
		tags   []string
	}{
		{"linux", "amd64", []string{"cgo"}},
		{"windows", "amd64", []string{"test"}},
		{"darwin", "arm64", []string{"debug"}},
	}

	for _, ctx := range contexts {
		buildCtx := cache.CreateContext(ctx.goos, ctx.goarch, ctx.tags)
		fmt.Printf("   GOOS=%s GOARCH=%s Tags=%v\n",
			buildCtx.GOOS, buildCtx.GOARCH, buildCtx.BuildTags)
	}

	// 6. Load a local package
	fmt.Println("\n6. Loading local package:")

	// Create a temporary package for demonstration
	tmpDir, err := ioutil.TempDir("", "buildcache_example")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create Go files in the temp directory
	mainFile := filepath.Join(tmpDir, "main.go")
	mainContent := `package example

import (
	"fmt"
	"strings"
)

// Example function demonstrates the package.
func Example() {
	fmt.Println(strings.ToUpper("hello"))
}`

	testFile := filepath.Join(tmpDir, "main_test.go")
	testContent := `package example

import "testing"

func TestExample(t *testing.T) {
	Example()
}`

	ioutil.WriteFile(mainFile, []byte(mainContent), 0644)
	ioutil.WriteFile(testFile, []byte(testContent), 0644)

	pkg, err := cache.LoadPackage(tmpDir)
	if err != nil {
		log.Printf("Error loading package: %v", err)
	} else {
		fmt.Printf("   Package: %s\n", pkg.Name)
		fmt.Printf("   Go files: %v\n", pkg.GoFiles)
		fmt.Printf("   Test files: %v\n", pkg.TestGoFiles)
		fmt.Printf("   Dependencies: %v\n", pkg.Dependencies)
	}

	// 7. Check file matching
	fmt.Println("\n7. File matching:")
	files := []string{
		"main.go",
		"main_test.go",
		"main_" + runtime.GOOS + ".go",
		"main_windows.go",
	}

	for _, file := range files {
		match, _ := cache.MatchFile(tmpDir, file)
		fmt.Printf("   %s: match=%v\n", file, match)
	}

	// 8. Check standard library packages
	fmt.Println("\n8. Standard library check:")
	packages := []string{
		"fmt",
		"net/http",
		"github.com/user/package",
		"example.com/module",
	}

	for _, pkg := range packages {
		isStd := cache.IsStandardPackage(pkg)
		fmt.Printf("   %s: standard=%v\n", pkg, isStd)
	}

	// 9. Batch load multiple packages
	fmt.Println("\n9. Batch loading packages:")

	// Create multiple temp packages
	var dirs []string
	for i := 0; i < 3; i++ {
		dir := filepath.Join(tmpDir, fmt.Sprintf("pkg%d", i))
		os.Mkdir(dir, 0755)

		file := filepath.Join(dir, "main.go")
		content := fmt.Sprintf(`package pkg%d

func Func%d() string {
	return "pkg%d"
}`, i, i, i)

		ioutil.WriteFile(file, []byte(content), 0644)
		dirs = append(dirs, dir)
	}

	results := cache.BatchLoadPackages(dirs)
	fmt.Printf("   Loaded %d packages:\n", len(results))
	for dir, pkg := range results {
		fmt.Printf("     %s: %s\n", filepath.Base(dir), pkg.Name)
	}

	// 10. File information caching
	fmt.Println("\n10. File information:")

	// Trigger file info caching
	cache.MatchFile(tmpDir, "main.go")

	fileInfo := cache.GetFileInfo(mainFile)
	if fileInfo != nil {
		fmt.Printf("   File: %s\n", filepath.Base(fileInfo.Path))
		fmt.Printf("   Package: %s\n", fileInfo.Package)
		fmt.Printf("   Imports: %v\n", fileInfo.Imports)
		fmt.Printf("   Is test: %v\n", fileInfo.IsTest)
		fmt.Printf("   Is generated: %v\n", fileInfo.IsGenerated)
	}

	// 11. OS/Arch specific files
	fmt.Println("\n11. OS/Arch file checking:")
	osArchFiles := []string{
		"file.go",
		"file_linux.go",
		"file_amd64.go",
		"file_linux_amd64.go",
		"file_test.go",
	}

	allTags := map[string]bool{
		"linux":  runtime.GOOS == "linux",
		"amd64":  runtime.GOARCH == "amd64",
		"darwin": runtime.GOOS == "darwin",
		"arm64":  runtime.GOARCH == "arm64",
	}

	for _, file := range osArchFiles {
		good := cache.GoodOSArchFile(file, allTags)
		fmt.Printf("   %s: good=%v\n", file, good)
	}

	// 12. Warmup cache
	fmt.Println("\n12. Cache warmup:")
	cache.WarmupCache(dirs)
	fmt.Println("   Cache warmed up with local packages")

	// Show cache statistics
	fmt.Println("\n=== Cache Statistics ===")
	stats := cache.GetStatistics()
	fmt.Printf("Constraint cache size: %d\n", stats["constraint_cache_size"])
	fmt.Printf("Package cache size: %d\n", stats["package_cache_size"])
	fmt.Printf("Import cache size: %d\n", stats["import_cache_size"])
	fmt.Printf("Context cache size: %d\n", stats["context_cache_size"])
	fmt.Printf("File cache size: %d\n", stats["file_cache_size"])
	fmt.Printf("Cache hits: %d\n", stats["cache_hits"])
	fmt.Printf("Cache misses: %d\n", stats["cache_misses"])
	if stats["cache_hits"].(int64)+stats["cache_misses"].(int64) > 0 {
		fmt.Printf("Hit rate: %.2f%%\n", stats["hit_rate"])
	}
	fmt.Printf("Total evaluations: %d\n", stats["total_evaluations"])
	fmt.Printf("Constraint evaluations: %d\n", stats["constraint_evals"])
	fmt.Printf("Package loads: %d\n", stats["package_loads"])
	fmt.Printf("Import resolves: %d\n", stats["import_resolves"])
	fmt.Printf("Custom tags: %v\n", stats["custom_tags"])

	// Clear cache
	fmt.Println("\n13. Clearing cache:")
	cache.Clear()
	stats = cache.GetStatistics()
	fmt.Printf("   Cache size after clear: %d\n", stats["constraint_cache_size"])
}
