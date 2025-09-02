// Package core provides tests for the decorator extraction engine.
package core

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestNewDecoratorExtractor(t *testing.T) {
	t.Run("with default config", func(t *testing.T) {
		de := NewDecoratorExtractor(nil)
		if de == nil {
			t.Fatal("Expected non-nil decorator extractor")
		}
		if de.config == nil {
			t.Error("Expected non-nil config")
		}
		if !de.config.EnableCache {
			t.Error("Expected cache to be enabled by default")
		}
		if !de.config.UseByteScanning {
			t.Error("Expected byte scanning to be enabled by default")
		}
	})

	t.Run("with custom config", func(t *testing.T) {
		config := &ExtractorConfig{
			EnableCache:        false,
			MaxCacheEntries:    500,
			ParallelExtraction: false,
			WorkerCount:        2,
			UseByteScanning:    false,
			CaptureContext:     false,
		}
		de := NewDecoratorExtractor(config)
		if de.config.EnableCache {
			t.Error("Expected cache to be disabled")
		}
		if de.config.MaxCacheEntries != 500 {
			t.Errorf("Expected max cache entries 500, got %d", de.config.MaxCacheEntries)
		}
	})

	t.Run("with custom patterns", func(t *testing.T) {
		config := &ExtractorConfig{
			CustomPatterns: map[string]string{
				"custom": `@Custom\((.*?)\)`,
			},
		}
		de := NewDecoratorExtractor(config)
		if _, exists := de.patterns["custom"]; !exists {
			t.Error("Expected custom pattern to be added")
		}
	})
}

func TestExtractREST(t *testing.T) {
	de := NewDecoratorExtractor(nil)

	tests := []struct {
		name     string
		source   string
		expected []string
	}{
		{
			"GET decorator",
			`@GET("/users")
func GetUsers() {}`,
			[]string{"GET"},
		},
		{
			"POST with options",
			`@POST("/users", {"auth": true})
func CreateUser() {}`,
			[]string{"POST"},
		},
		{
			"multiple REST decorators",
			`@GET("/users")
@POST("/users")
@DELETE("/users/{id}")
func UserHandler() {}`,
			[]string{"GET", "POST", "DELETE"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := de.Extract([]byte(tt.source))
			if err != nil {
				t.Fatalf("Failed to extract: %v", err)
			}

			if len(result.Decorators) != len(tt.expected) {
				t.Errorf("Expected %d decorators, got %d", len(tt.expected), len(result.Decorators))
			}

			for i, dec := range result.Decorators {
				if i < len(tt.expected) && dec.Name != tt.expected[i] {
					t.Errorf("Expected decorator name %s, got %s", tt.expected[i], dec.Name)
				}
			}
		})
	}
}

func TestExtractValidation(t *testing.T) {
	de := NewDecoratorExtractor(nil)

	source := `
@Required
type User struct {
	@MinLength(3)
	@MaxLength(50)
	Name string

	@Email
	EmailAddress string

	@Pattern("^[0-9]+$")
	PhoneNumber string
}
`

	result, err := de.Extract([]byte(source))
	if err != nil {
		t.Fatalf("Failed to extract: %v", err)
	}

	expectedDecorators := []string{"Required", "MinLength", "MaxLength", "Email", "Pattern"}
	if len(result.Decorators) != len(expectedDecorators) {
		t.Errorf("Expected %d decorators, got %d", len(expectedDecorators), len(result.Decorators))
	}

	// Check decorator types
	for _, dec := range result.Decorators {
		if dec.Type != "validation" {
			t.Errorf("Expected decorator type 'validation', got %s", dec.Type)
		}
	}
}

func TestExtractSecurity(t *testing.T) {
	de := NewDecoratorExtractor(nil)

	source := `
@Auth
@Role("admin")
@Permission("write")
@RateLimit(100)
func AdminEndpoint() {}
`

	result, err := de.Extract([]byte(source))
	if err != nil {
		t.Fatalf("Failed to extract: %v", err)
	}

	expectedDecorators := []string{"Auth", "Role", "Permission", "RateLimit"}
	if len(result.Decorators) != len(expectedDecorators) {
		t.Errorf("Expected %d decorators, got %d", len(expectedDecorators), len(result.Decorators))
	}

	// Check decorator with arguments
	for _, dec := range result.Decorators {
		if dec.Name == "RateLimit" {
			if len(dec.Arguments) == 0 || dec.Arguments[0] != "100" {
				t.Error("Expected RateLimit to have argument '100'")
			}
		}
	}
}

func TestExtractDatabase(t *testing.T) {
	de := NewDecoratorExtractor(nil)

	source := `
@Transaction
@Cache("5m")
@Query("SELECT * FROM users WHERE id = ?")
func GetUser(id int) {}
`

	result, err := de.Extract([]byte(source))
	if err != nil {
		t.Fatalf("Failed to extract: %v", err)
	}

	if len(result.Decorators) != 3 {
		t.Errorf("Expected 3 decorators, got %d", len(result.Decorators))
	}

	for _, dec := range result.Decorators {
		if dec.Type != "database" {
			t.Errorf("Expected decorator type 'database', got %s", dec.Type)
		}
	}
}

func TestExtractMonitoring(t *testing.T) {
	de := NewDecoratorExtractor(nil)

	source := `
@Trace("user-service")
@Metric("request_count")
@Log("info")
@Alert("high_latency")
func ProcessRequest() {}
`

	result, err := de.Extract([]byte(source))
	if err != nil {
		t.Fatalf("Failed to extract: %v", err)
	}

	expectedDecorators := []string{"Trace", "Metric", "Log", "Alert"}
	if len(result.Decorators) != len(expectedDecorators) {
		t.Errorf("Expected %d decorators, got %d", len(expectedDecorators), len(result.Decorators))
	}
}

func TestExtractGeneric(t *testing.T) {
	de := NewDecoratorExtractor(nil)

	source := `
@CustomDecorator
@AnotherOne("arg1", "arg2")
@YetAnother
func MyFunction() {}
`

	result, err := de.Extract([]byte(source))
	if err != nil {
		t.Fatalf("Failed to extract: %v", err)
	}

	if len(result.Decorators) != 3 {
		t.Errorf("Expected 3 decorators, got %d", len(result.Decorators))
	}

	for _, dec := range result.Decorators {
		if dec.Type != "generic" {
			t.Errorf("Expected decorator type 'generic', got %s", dec.Type)
		}
	}
}

func TestCaptureContext(t *testing.T) {
	config := &ExtractorConfig{
		CaptureContext: true,
		ContextLines:   2,
	}
	de := NewDecoratorExtractor(config)

	source := `line1
line2
line3
@GET("/test")
line5
line6
line7`

	result, err := de.Extract([]byte(source))
	if err != nil {
		t.Fatalf("Failed to extract: %v", err)
	}

	if len(result.Decorators) == 0 {
		t.Fatal("Expected at least one decorator")
	}

	dec := result.Decorators[0]
	if len(dec.Context) == 0 {
		t.Error("Expected context to be captured")
	}
}

func TestByteScanning(t *testing.T) {
	config := &ExtractorConfig{
		UseByteScanning: true,
	}
	de := NewDecoratorExtractor(config)

	source := `
// This is a comment
@GET("/users")
func GetUsers() {}

// Another comment without decorator
func NormalFunction() {}

@POST("/users")
func CreateUser() {}
`

	result, err := de.Extract([]byte(source))
	if err != nil {
		t.Fatalf("Failed to extract: %v", err)
	}

	if len(result.Decorators) != 2 {
		t.Errorf("Expected 2 decorators, got %d", len(result.Decorators))
	}

	if result.Metadata["extraction_method"] != "byte_scanning" {
		t.Error("Expected byte_scanning extraction method")
	}
}

func TestRegexExtraction(t *testing.T) {
	config := &ExtractorConfig{
		UseByteScanning: false,
	}
	de := NewDecoratorExtractor(config)

	source := `@GET("/test") @POST("/test")
func Handler() {}`

	result, err := de.Extract([]byte(source))
	if err != nil {
		t.Fatalf("Failed to extract: %v", err)
	}

	if result.Metadata["extraction_method"] != "regex" {
		t.Error("Expected regex extraction method")
	}
}

func TestCaching(t *testing.T) {
	config := &ExtractorConfig{
		EnableCache: true,
	}
	de := NewDecoratorExtractor(config)

	source := []byte(`@GET("/test")
func Test() {}`)

	// First extraction (miss)
	_, err := de.Extract(source)
	if err != nil {
		t.Fatalf("Failed to extract: %v", err)
	}

	initialMisses := de.misses

	// Second extraction (should hit cache)
	_, err = de.Extract(source)
	if err != nil {
		t.Fatalf("Failed to extract: %v", err)
	}

	if de.hits == 0 {
		t.Error("Expected at least one cache hit")
	}

	if de.misses != initialMisses {
		t.Error("Expected misses to remain the same")
	}
}

func TestAddPattern(t *testing.T) {
	de := NewDecoratorExtractor(nil)

	err := de.AddPattern("custom", `@Custom\((.*?)\)`, 50)
	if err != nil {
		t.Fatalf("Failed to add pattern: %v", err)
	}

	de.mu.RLock()
	pattern, exists := de.patterns["custom"]
	de.mu.RUnlock()

	if !exists {
		t.Error("Expected custom pattern to exist")
	}

	if pattern.Priority != 50 {
		t.Errorf("Expected priority 50, got %d", pattern.Priority)
	}

	// Test with invalid regex
	err = de.AddPattern("invalid", `[`, 10)
	if err == nil {
		t.Error("Expected error for invalid regex")
	}
}

func TestExtractParallel(t *testing.T) {
	config := &ExtractorConfig{
		ParallelExtraction: true,
		WorkerCount:        2,
	}
	de := NewDecoratorExtractor(config)

	sources := map[string][]byte{
		"file1": []byte(`@GET("/test1")
func Test1() {}`),
		"file2": []byte(`@POST("/test2")
func Test2() {}`),
		"file3": []byte(`@DELETE("/test3")
func Test3() {}`),
	}

	results, err := de.ExtractParallel(sources)
	if err != nil {
		t.Fatalf("Failed to extract in parallel: %v", err)
	}

	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}

	for name, result := range results {
		if len(result.Decorators) != 1 {
			t.Errorf("Expected 1 decorator in %s, got %d", name, len(result.Decorators))
		}
	}
}

func TestExtractFromReader(t *testing.T) {
	de := NewDecoratorExtractor(nil)

	source := `@GET("/test")
func Test() {}`
	reader := strings.NewReader(source)

	result, err := de.ExtractFromReader(reader)
	if err != nil {
		t.Fatalf("Failed to extract from reader: %v", err)
	}

	if len(result.Decorators) != 1 {
		t.Errorf("Expected 1 decorator, got %d", len(result.Decorators))
	}
}

func TestParseArguments(t *testing.T) {
	de := NewDecoratorExtractor(nil)

	tests := []struct {
		name     string
		args     string
		expected []string
	}{
		{"single argument", `"value"`, []string{"value"}},
		{"multiple arguments", `"arg1", "arg2", "arg3"`, []string{"arg1", "arg2", "arg3"}},
		{"mixed quotes", `'single', "double"`, []string{"single", "double"}},
		{"with spaces", ` "arg1" , "arg2" `, []string{"arg1", "arg2"}},
		{"empty", "", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := de.parseArguments(tt.args)
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d arguments, got %d", len(tt.expected), len(result))
				return
			}
			for i, arg := range result {
				if arg != tt.expected[i] {
					t.Errorf("Expected argument %s, got %s", tt.expected[i], arg)
				}
			}
		})
	}
}

func TestParseProperties(t *testing.T) {
	de := NewDecoratorExtractor(nil)

	tests := []struct {
		name     string
		props    string
		expected map[string]interface{}
	}{
		{
			"simple properties",
			`{"key1": "value1", "key2": "value2"}`,
			map[string]interface{}{"key1": "value1", "key2": "value2"},
		},
		{
			"without braces",
			`"key": "value"`,
			map[string]interface{}{"key": "value"},
		},
		{
			"empty",
			`{}`,
			map[string]interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := de.parseProperties(tt.props)
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d properties, got %d", len(tt.expected), len(result))
			}
			for key, expectedVal := range tt.expected {
				if result[key] != expectedVal {
					t.Errorf("Expected property %s=%v, got %v", key, expectedVal, result[key])
				}
			}
		})
	}
}

func TestExtractorGetStatistics(t *testing.T) {
	de := NewDecoratorExtractor(nil)

	// Perform some operations
	source := []byte(`@GET("/test")
func Test() {}`)
	_, _ = de.Extract(source)
	_, _ = de.Extract(source) // Should hit cache

	stats := de.GetStatistics()

	if stats["extractions"].(int64) < 1 {
		t.Error("Expected at least 1 extraction")
	}

	if stats["cache_hits"].(int64) < 1 {
		t.Error("Expected at least 1 cache hit")
	}

	if stats["pattern_count"].(int) == 0 {
		t.Error("Expected patterns to be loaded")
	}
}

func TestClear(t *testing.T) {
	de := NewDecoratorExtractor(nil)

	// Add some data
	source := []byte(`@GET("/test")
func Test() {}`)
	_, _ = de.Extract(source)

	// Clear
	de.Clear()

	// Check statistics are reset
	if de.extractions != 0 {
		t.Error("Expected extractions to be reset")
	}
	if de.hits != 0 {
		t.Error("Expected hits to be reset")
	}
	if de.misses != 0 {
		t.Error("Expected misses to be reset")
	}

	de.mu.RLock()
	cacheSize := len(de.cache)
	de.mu.RUnlock()

	if cacheSize != 0 {
		t.Error("Expected cache to be cleared")
	}
}

func TestCacheTTL(t *testing.T) {
	config := &ExtractorConfig{
		EnableCache: true,
		CacheTTL:    100 * time.Millisecond,
	}
	de := NewDecoratorExtractor(config)

	source := []byte(`@GET("/test")
func Test() {}`)

	// First extraction
	_, _ = de.Extract(source)

	// Should hit cache
	_, _ = de.Extract(source)
	initialHits := de.hits

	// Wait for TTL to expire
	time.Sleep(150 * time.Millisecond)

	// Should miss cache
	_, _ = de.Extract(source)
	if de.hits != initialHits {
		t.Error("Expected cache entry to expire")
	}
}

func TestCacheEviction(t *testing.T) {
	config := &ExtractorConfig{
		EnableCache:     true,
		MaxCacheEntries: 2,
	}
	de := NewDecoratorExtractor(config)

	// Add entries to fill cache
	for i := 0; i < 3; i++ {
		source := []byte(strings.Repeat("a", i+1) + `@GET("/test")`)
		_, _ = de.Extract(source)
	}

	de.mu.RLock()
	cacheSize := len(de.cache)
	de.mu.RUnlock()

	if cacheSize > 2 {
		t.Errorf("Expected cache size <= 2, got %d", cacheSize)
	}
}

func TestLineColumnPosition(t *testing.T) {
	de := NewDecoratorExtractor(nil)

	source := `line1
line2
  @GET("/test")
line4`

	result, err := de.Extract([]byte(source))
	if err != nil {
		t.Fatalf("Failed to extract: %v", err)
	}

	if len(result.Decorators) == 0 {
		t.Fatal("Expected at least one decorator")
	}

	dec := result.Decorators[0]
	if dec.Line != 3 {
		t.Errorf("Expected line 3, got %d", dec.Line)
	}
	if dec.Column != 3 {
		t.Errorf("Expected column 3, got %d", dec.Column)
	}
}

func BenchmarkExtractByteScanning(b *testing.B) {
	config := &ExtractorConfig{
		UseByteScanning: true,
		EnableCache:     false,
	}
	de := NewDecoratorExtractor(config)

	source := []byte(`
@GET("/users")
@Auth
@RateLimit(100)
func GetUsers() {}

@POST("/users")
@Auth
@Validate
func CreateUser() {}
`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = de.Extract(source)
	}
}

func BenchmarkExtractRegex(b *testing.B) {
	config := &ExtractorConfig{
		UseByteScanning: false,
		EnableCache:     false,
	}
	de := NewDecoratorExtractor(config)

	source := []byte(`
@GET("/users")
@Auth
@RateLimit(100)
func GetUsers() {}

@POST("/users")
@Auth
@Validate
func CreateUser() {}
`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = de.Extract(source)
	}
}

func BenchmarkExtractWithCache(b *testing.B) {
	config := &ExtractorConfig{
		EnableCache: true,
	}
	de := NewDecoratorExtractor(config)

	source := []byte(`@GET("/test")
func Test() {}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = de.Extract(source)
	}
}

func BenchmarkExtractParallelLarge(b *testing.B) {
	config := &ExtractorConfig{
		ParallelExtraction: true,
		WorkerCount:        4,
	}
	de := NewDecoratorExtractor(config)

	// Create multiple sources
	sources := make(map[string][]byte)
	for i := 0; i < 10; i++ {
		var buf bytes.Buffer
		for j := 0; j < 100; j++ {
			buf.WriteString(`@GET("/test")
func Test() {}\n`)
		}
		sources[string(rune(i))] = buf.Bytes()
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = de.ExtractParallel(sources)
	}
}
