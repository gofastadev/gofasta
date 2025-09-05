// Package core provides a high-performance regex-based decorator extraction engine.
// This implements Phase 1.3a: Design regex-based decorator extraction engine (microsecond parsing).
package core

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// DecoratorExtractor provides high-performance decorator extraction from source code
type DecoratorExtractor struct {
	config   *ExtractorConfig
	patterns map[string]*CompiledPattern
	cache    map[string]*ExtractionResult
	mu       sync.RWMutex
	
	// Performance metrics
	extractions    int64
	hits          int64
	misses        int64
	totalDuration int64
}

// ExtractorConfig contains configuration for the decorator extractor
type ExtractorConfig struct {
	// Performance settings
	EnableCache         bool
	MaxCacheEntries    int
	CacheTTL           time.Duration
	ParallelExtraction bool
	WorkerCount        int
	
	// Pattern settings
	CustomPatterns     map[string]string
	StrictMode        bool
	CaptureContext    bool
	ContextLines      int
	
	// Optimization settings
	UseByteScanning   bool
	EnableMetrics     bool
	PrecompilePatterns bool
}

// CompiledPattern represents a compiled regex pattern for extraction
type CompiledPattern struct {
	Name         string
	Pattern      *regexp.Regexp
	Priority     int
	CaptureGroups []string
	Validator    func(match []string) bool
}

// ExtractionResult contains the results of decorator extraction
type ExtractionResult struct {
	Decorators   []Decorator
	Metadata     map[string]interface{}
	SourceHash   string
	ExtractedAt  time.Time
	Duration     time.Duration
	BytesScanned int64
}

// Decorator represents an extracted decorator
type Decorator struct {
	Type       string                 `json:"type"`
	Name       string                 `json:"name"`
	Arguments  []string               `json:"arguments,omitempty"`
	Properties map[string]interface{} `json:"properties,omitempty"`
	Line       int                    `json:"line"`
	Column     int                    `json:"column"`
	Context    []string               `json:"context,omitempty"`
	Raw        string                 `json:"raw"`
}

// Standard decorator patterns
var (
	// REST decorators
	restPattern = regexp.MustCompile(`@(GET|POST|PUT|DELETE|PATCH|HEAD|OPTIONS)\s*\(\s*["']([^"']+)["']\s*(?:,\s*({[^}]*}))?\s*\)`)
	
	// Validation decorators
	validationPattern = regexp.MustCompile(`@(Required|MinLength|MaxLength|Pattern|Email|URL|UUID)\s*(?:\(\s*([^)]*)\s*\))?`)
	
	// Security decorators
	securityPattern = regexp.MustCompile(`@(Auth|Role|Permission|RateLimit|CORS)\s*(?:\(\s*([^)]*)\s*\))?`)
	
	// Database decorators
	databasePattern = regexp.MustCompile(`@(Transaction|Cache|Index|Query)\s*(?:\(\s*([^)]*)\s*\))?`)
	
	// Monitoring decorators
	monitoringPattern = regexp.MustCompile(`@(Trace|Metric|Log|Alert)\s*(?:\(\s*([^)]*)\s*\))?`)
	
	// Generic decorator pattern - supports multi-line decorators
	genericPattern = regexp.MustCompile(`(?s)@(\w+)\s*(?:\(\s*(.*?)\s*\))?`)
)

// DefaultExtractorConfig returns the default configuration
func DefaultExtractorConfig() *ExtractorConfig {
	return &ExtractorConfig{
		EnableCache:        true,
		MaxCacheEntries:    1000,
		CacheTTL:          5 * time.Minute,
		ParallelExtraction: true,
		WorkerCount:       4,
		StrictMode:        false,
		CaptureContext:    true,
		ContextLines:      2,
		UseByteScanning:   true,
		EnableMetrics:     true,
		PrecompilePatterns: true,
	}
}

// NewDecoratorExtractor creates a new decorator extractor
func NewDecoratorExtractor(config *ExtractorConfig) *DecoratorExtractor {
	if config == nil {
		config = DefaultExtractorConfig()
	}
	
	de := &DecoratorExtractor{
		config:   config,
		patterns: make(map[string]*CompiledPattern),
		cache:    make(map[string]*ExtractionResult),
	}
	
	// Initialize standard patterns
	de.initializePatterns()
	
	// Add custom patterns
	for name, pattern := range config.CustomPatterns {
		de.AddPattern(name, pattern, 100)
	}
	
	return de
}

// initializePatterns initializes standard decorator patterns
func (de *DecoratorExtractor) initializePatterns() {
	de.patterns["rest"] = &CompiledPattern{
		Name:          "rest",
		Pattern:       restPattern,
		Priority:      10,
		CaptureGroups: []string{"method", "path", "options"},
	}
	
	de.patterns["validation"] = &CompiledPattern{
		Name:          "validation",
		Pattern:       validationPattern,
		Priority:      20,
		CaptureGroups: []string{"type", "args"},
	}
	
	de.patterns["security"] = &CompiledPattern{
		Name:          "security",
		Pattern:       securityPattern,
		Priority:      15,
		CaptureGroups: []string{"type", "args"},
	}
	
	de.patterns["database"] = &CompiledPattern{
		Name:          "database",
		Pattern:       databasePattern,
		Priority:      25,
		CaptureGroups: []string{"type", "args"},
	}
	
	de.patterns["monitoring"] = &CompiledPattern{
		Name:          "monitoring",
		Pattern:       monitoringPattern,
		Priority:      30,
		CaptureGroups: []string{"type", "args"},
	}
	
	de.patterns["generic"] = &CompiledPattern{
		Name:          "generic",
		Pattern:       genericPattern,
		Priority:      100,
		CaptureGroups: []string{"name", "args"},
	}
}

// Extract extracts decorators from source code
func (de *DecoratorExtractor) Extract(source []byte) (*ExtractionResult, error) {
	startTime := time.Now()
	
	// Check cache
	if de.config.EnableCache {
		if cached := de.getFromCache(source); cached != nil {
			atomic.AddInt64(&de.hits, 1)
			return cached, nil
		}
		atomic.AddInt64(&de.misses, 1)
	}
	
	// Perform extraction
	var result *ExtractionResult
	var err error
	
	if de.config.UseByteScanning {
		result, err = de.extractWithByteScanning(source)
	} else {
		result, err = de.extractWithRegex(source)
	}
	
	if err != nil {
		return nil, err
	}
	
	// Update metrics
	duration := time.Since(startTime)
	result.Duration = duration
	result.ExtractedAt = time.Now()
	result.BytesScanned = int64(len(source))
	
	atomic.AddInt64(&de.extractions, 1)
	atomic.AddInt64(&de.totalDuration, int64(duration))
	
	// Cache result
	if de.config.EnableCache {
		de.cacheResult(source, result)
	}
	
	return result, nil
}

// extractWithByteScanning uses optimized byte scanning for extraction
func (de *DecoratorExtractor) extractWithByteScanning(source []byte) (*ExtractionResult, error) {
	var decorators []Decorator
	scanner := bufio.NewScanner(bytes.NewReader(source))
	lineNum := 0
	
	for scanner.Scan() {
		lineNum++
		line := scanner.Bytes()
		
		// Quick check for @ symbol
		if bytes.IndexByte(line, '@') == -1 {
			continue
		}
		
		// Extract decorators from line
		lineDecorators := de.extractFromLine(line, lineNum)
		decorators = append(decorators, lineDecorators...)
	}
	
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning error: %w", err)
	}
	
	return &ExtractionResult{
		Decorators: decorators,
		Metadata: map[string]interface{}{
			"extraction_method": "byte_scanning",
			"lines_scanned":    lineNum,
		},
	}, nil
}

// extractWithRegex uses regex patterns for extraction
func (de *DecoratorExtractor) extractWithRegex(source []byte) (*ExtractionResult, error) {
	var decorators []Decorator
	text := string(source)
	lines := strings.Split(text, "\n")
	
	// Track already matched decorators to avoid duplicates
	matched := make(map[string]bool)
	
	// Try patterns in priority order (specific patterns first, generic last)
	patterns := de.getSortedPatterns()
	for _, pattern := range patterns {
		matches := pattern.Pattern.FindAllStringSubmatchIndex(text, -1)
		
		for _, match := range matches {
			// Create a unique key for this match position
			key := fmt.Sprintf("%d-%d", match[0], match[1])
			if matched[key] {
				continue // Already matched by a higher priority pattern
			}
			
			decorator := de.createDecorator(text, match, pattern, lines)
			if decorator != nil {
				decorators = append(decorators, *decorator)
				matched[key] = true
			}
		}
	}
	
	return &ExtractionResult{
		Decorators: decorators,
		Metadata: map[string]interface{}{
			"extraction_method": "regex",
			"patterns_used":    len(de.patterns),
		},
	}, nil
}

// extractFromLine extracts decorators from a single line
func (de *DecoratorExtractor) extractFromLine(line []byte, lineNum int) []Decorator {
	var decorators []Decorator
	lineStr := string(line)
	
	// Track already matched decorators to avoid duplicates
	matched := make(map[int]bool)
	
	// Try patterns in priority order
	patterns := de.getSortedPatterns()
	for _, pattern := range patterns {
		matches := pattern.Pattern.FindAllStringSubmatchIndex(lineStr, -1)
		
		for _, match := range matches {
			// Check if this position was already matched
			if matched[match[0]] {
				continue
			}
			
			decorator := Decorator{
				Type:   pattern.Name,
				Line:   lineNum,
				Column: match[0] + 1,
				Raw:    lineStr[match[0]:match[1]],
			}
			
			// Extract name and arguments
			if len(match) > 2 && match[2] >= 0 {
				decorator.Name = lineStr[match[2]:match[3]]
			}
			if len(match) > 4 && match[4] >= 0 {
				decorator.Arguments = de.parseArguments(lineStr[match[4]:match[5]])
			}
			
			decorators = append(decorators, decorator)
			matched[match[0]] = true
		}
	}
	
	return decorators
}

// createDecorator creates a decorator from regex match
func (de *DecoratorExtractor) createDecorator(text string, match []int, pattern *CompiledPattern, lines []string) *Decorator {
	if len(match) < 2 {
		return nil
	}
	
	// Calculate line and column
	beforeMatch := text[:match[0]]
	lineNum := strings.Count(beforeMatch, "\n") + 1
	lastNewline := strings.LastIndex(beforeMatch, "\n")
	column := match[0] - lastNewline
	
	decorator := &Decorator{
		Type:   pattern.Name,
		Line:   lineNum,
		Column: column,
		Raw:    text[match[0]:match[1]],
	}
	
	// Extract captured groups
	for i := 2; i < len(match); i += 2 {
		if match[i] >= 0 && match[i+1] >= 0 {
			groupIndex := i/2 - 1
			if groupIndex < len(pattern.CaptureGroups) {
				value := text[match[i]:match[i+1]]
				captureGroup := pattern.CaptureGroups[groupIndex]
				
				// Multi-line decorator support - no debug output needed
				
				switch captureGroup {
				case "name", "method", "type":
					decorator.Name = value
				case "args", "path":
					decorator.Arguments = de.parseArguments(value)
					decorator.Properties = de.parseProperties(value) // Also try properties parsing
				case "options":
					decorator.Properties = de.parseProperties(value)
				}
			}
		}
	}
	
	// Capture context if enabled
	if de.config.CaptureContext && lineNum > 0 && lineNum <= len(lines) {
		decorator.Context = de.captureContext(lines, lineNum-1)
	}
	
	// Validate if validator exists
	if pattern.Validator != nil {
		groups := make([]string, 0)
		for i := 0; i < len(match); i += 2 {
			if match[i] >= 0 && match[i+1] >= 0 {
				groups = append(groups, text[match[i]:match[i+1]])
			}
		}
		if !pattern.Validator(groups) {
			return nil
		}
	}
	
	return decorator
}

// parseArguments parses decorator arguments
func (de *DecoratorExtractor) parseArguments(args string) []string {
	args = strings.TrimSpace(args)
	if args == "" {
		return nil
	}
	
	// Simple argument parsing
	// TODO: Implement more sophisticated parsing for complex arguments
	parts := strings.Split(args, ",")
	result := make([]string, 0, len(parts))
	
	for _, part := range parts {
		part = strings.TrimSpace(part)
		part = strings.Trim(part, `"'`)
		if part != "" {
			result = append(result, part)
		}
	}
	
	return result
}

// parseProperties parses JSON-like properties
func (de *DecoratorExtractor) parseProperties(props string) map[string]interface{} {
	// Simple property parsing
	// TODO: Implement proper JSON parsing
	result := make(map[string]interface{})
	
	props = strings.TrimSpace(props)
	original := props
	props = strings.Trim(props, "{}")
	
	if props == "" {
		return result
	}
	
	// Handle simple key-value pairs
	// Split by comma but be careful about nested structures
	var pairs []string
	var current strings.Builder
	quoteChar := rune(0)
	depth := 0
	
	for _, r := range props {
		switch r {
		case '"', '\'':
			if quoteChar == 0 {
				quoteChar = r
			} else if quoteChar == r {
				quoteChar = 0
			}
			current.WriteRune(r)
		case '{':
			if quoteChar == 0 {
				depth++
			}
			current.WriteRune(r)
		case '}':
			if quoteChar == 0 {
				depth--
			}
			current.WriteRune(r)
		case ',':
			if quoteChar == 0 && depth == 0 {
				pairs = append(pairs, current.String())
				current.Reset()
			} else {
				current.WriteRune(r)
			}
		default:
			current.WriteRune(r)
		}
	}
	
	if current.Len() > 0 {
		pairs = append(pairs, current.String())
	}
	
	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		
		// Find the colon separator
		colonIdx := strings.Index(pair, ":")
		if colonIdx < 0 {
			// If the original had braces, this is a parsing error
			if strings.HasPrefix(original, "{") {
				continue
			}
			// Otherwise treat the whole thing as a single value
			key := strings.Trim(pair, `"'`)
			result[key] = true
			continue
		}
		
		keyPart := strings.TrimSpace(pair[:colonIdx])
		valuePart := strings.TrimSpace(pair[colonIdx+1:])
		
		key := strings.Trim(keyPart, `"'`)
		value := strings.Trim(valuePart, `"'`)
		
		result[key] = value
	}
	
	return result
}

// captureContext captures surrounding context lines
func (de *DecoratorExtractor) captureContext(lines []string, lineIndex int) []string {
	var context []string
	
	start := lineIndex - de.config.ContextLines
	if start < 0 {
		start = 0
	}
	
	end := lineIndex + de.config.ContextLines + 1
	if end > len(lines) {
		end = len(lines)
	}
	
	for i := start; i < end; i++ {
		if i != lineIndex {
			context = append(context, lines[i])
		}
	}
	
	return context
}

// AddPattern adds a custom pattern
func (de *DecoratorExtractor) AddPattern(name, pattern string, priority int) error {
	compiled, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("failed to compile pattern %s: %w", name, err)
	}
	
	de.mu.Lock()
	defer de.mu.Unlock()
	
	de.patterns[name] = &CompiledPattern{
		Name:     name,
		Pattern:  compiled,
		Priority: priority,
	}
	
	return nil
}

// ExtractParallel extracts decorators from multiple sources in parallel
func (de *DecoratorExtractor) ExtractParallel(sources map[string][]byte) (map[string]*ExtractionResult, error) {
	if !de.config.ParallelExtraction || len(sources) <= 1 {
		// Extract sequentially
		results := make(map[string]*ExtractionResult)
		for name, source := range sources {
			result, err := de.Extract(source)
			if err != nil {
				return results, fmt.Errorf("failed to extract from %s: %w", name, err)
			}
			results[name] = result
		}
		return results, nil
	}
	
	// Extract in parallel
	type workResult struct {
		name   string
		result *ExtractionResult
		err    error
	}
	
	workChan := make(chan workResult, len(sources))
	semaphore := make(chan struct{}, de.config.WorkerCount)
	
	var wg sync.WaitGroup
	for name, source := range sources {
		wg.Add(1)
		go func(n string, s []byte) {
			defer wg.Done()
			
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			
			result, err := de.Extract(s)
			workChan <- workResult{name: n, result: result, err: err}
		}(name, source)
	}
	
	go func() {
		wg.Wait()
		close(workChan)
	}()
	
	results := make(map[string]*ExtractionResult)
	for work := range workChan {
		if work.err != nil {
			return results, fmt.Errorf("failed to extract from %s: %w", work.name, work.err)
		}
		results[work.name] = work.result
	}
	
	return results, nil
}

// ExtractFromReader extracts decorators from an io.Reader
func (de *DecoratorExtractor) ExtractFromReader(reader io.Reader) (*ExtractionResult, error) {
	source, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read source: %w", err)
	}
	
	return de.Extract(source)
}

// getFromCache retrieves cached extraction result
func (de *DecoratorExtractor) getFromCache(source []byte) *ExtractionResult {
	hash := de.calculateHash(source)
	
	de.mu.RLock()
	defer de.mu.RUnlock()
	
	if cached, exists := de.cache[hash]; exists {
		if de.config.CacheTTL == 0 || time.Since(cached.ExtractedAt) < de.config.CacheTTL {
			return cached
		}
	}
	
	return nil
}

// cacheResult caches extraction result
func (de *DecoratorExtractor) cacheResult(source []byte, result *ExtractionResult) {
	hash := de.calculateHash(source)
	result.SourceHash = hash
	
	de.mu.Lock()
	defer de.mu.Unlock()
	
	// Check cache size limit
	if de.config.MaxCacheEntries > 0 && len(de.cache) >= de.config.MaxCacheEntries {
		// Evict oldest entry
		var oldestKey string
		var oldestTime time.Time
		
		for key, cached := range de.cache {
			if oldestKey == "" || cached.ExtractedAt.Before(oldestTime) {
				oldestKey = key
				oldestTime = cached.ExtractedAt
			}
		}
		
		if oldestKey != "" {
			delete(de.cache, oldestKey)
		}
	}
	
	de.cache[hash] = result
}

// calculateHash calculates hash of source code
func (de *DecoratorExtractor) calculateHash(source []byte) string {
	// Simple hash for demo - in production use crypto hash
	return fmt.Sprintf("%x", source[:min(64, len(source))])
}

// GetStatistics returns extraction statistics
func (de *DecoratorExtractor) GetStatistics() map[string]interface{} {
	extractions := atomic.LoadInt64(&de.extractions)
	hits := atomic.LoadInt64(&de.hits)
	misses := atomic.LoadInt64(&de.misses)
	totalDuration := atomic.LoadInt64(&de.totalDuration)
	
	hitRate := float64(0)
	if total := hits + misses; total > 0 {
		hitRate = float64(hits) * 100.0 / float64(total)
	}
	
	avgDuration := time.Duration(0)
	if extractions > 0 {
		avgDuration = time.Duration(totalDuration / extractions)
	}
	
	de.mu.RLock()
	cacheSize := len(de.cache)
	patternCount := len(de.patterns)
	de.mu.RUnlock()
	
	return map[string]interface{}{
		"extractions":       extractions,
		"cache_hits":       hits,
		"cache_misses":     misses,
		"cache_hit_rate":   hitRate,
		"cache_size":       cacheSize,
		"pattern_count":    patternCount,
		"avg_duration_ns":  avgDuration.Nanoseconds(),
		"avg_duration":     avgDuration.String(),
	}
}

// Clear clears the cache
func (de *DecoratorExtractor) Clear() {
	de.mu.Lock()
	defer de.mu.Unlock()
	
	de.cache = make(map[string]*ExtractionResult)
	atomic.StoreInt64(&de.extractions, 0)
	atomic.StoreInt64(&de.hits, 0)
	atomic.StoreInt64(&de.misses, 0)
	atomic.StoreInt64(&de.totalDuration, 0)
}

// getSortedPatterns returns patterns sorted by priority (lower priority first)
func (de *DecoratorExtractor) getSortedPatterns() []*CompiledPattern {
	de.mu.RLock()
	defer de.mu.RUnlock()
	
	patterns := make([]*CompiledPattern, 0, len(de.patterns))
	for _, p := range de.patterns {
		patterns = append(patterns, p)
	}
	
	// Sort by priority (lower priority value = higher priority)
	sort.Slice(patterns, func(i, j int) bool {
		return patterns[i].Priority < patterns[j].Priority
	})
	
	return patterns
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}