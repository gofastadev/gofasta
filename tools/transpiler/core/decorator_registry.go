// Package core provides a plugin-based decorator registry with parallel loading.
// This implements Phase 1.3b: Create plugin-based decorator registry with parallel loading.
package core

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"plugin"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// DecoratorRegistry manages decorator plugins with parallel loading
type DecoratorRegistry struct {
	config     *RegistryConfig
	decorators map[string]*RegisteredDecorator
	plugins    map[string]*LoadedPlugin
	handlers   map[string]DecoratorHandler
	mu         sync.RWMutex
	
	// Loading state
	loading    sync.Map
	loadErrors sync.Map
	
	// Metrics
	loadedPlugins   int64
	registrations   int64
	invocations     int64
	loadDuration    int64
}

// RegistryConfig contains configuration for the decorator registry
type RegistryConfig struct {
	// Plugin settings
	PluginDirs         []string
	PluginPattern      string
	EnableHotReload    bool
	ReloadInterval     time.Duration
	ParallelLoading    bool
	LoadWorkers        int
	
	// Registry settings
	MaxDecorators      int
	AllowOverride      bool
	ValidateOnRegister bool
	EnableMetrics      bool
	
	// Security settings
	AllowedPlugins     []string
	BlockedPlugins     []string
	RequireSignature   bool
	TrustedKeys        []string
}

// RegisteredDecorator represents a registered decorator
type RegisteredDecorator struct {
	Name         string                 `json:"name"`
	Type         string                 `json:"type"`
	Description  string                 `json:"description"`
	Version      string                 `json:"version"`
	Author       string                 `json:"author"`
	Handler      DecoratorHandler       `json:"-"`
	Schema       *DecoratorSchema       `json:"schema,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	RegisteredAt time.Time             `json:"registered_at"`
	Plugin       string                 `json:"plugin,omitempty"`
}

// LoadedPlugin represents a loaded plugin
type LoadedPlugin struct {
	Path         string
	Name         string
	Version      string
	Plugin       *plugin.Plugin
	Decorators   []string
	LoadedAt     time.Time
	Dependencies []string
}

// DecoratorHandler is the function signature for decorator handlers
type DecoratorHandler func(ctx context.Context, args DecoratorArgs) (DecoratorResult, error)

// DecoratorArgs contains arguments passed to a decorator
type DecoratorArgs struct {
	Target     interface{}            `json:"target"`
	Arguments  []interface{}          `json:"arguments"`
	Properties map[string]interface{} `json:"properties"`
	Context    map[string]interface{} `json:"context"`
}

// DecoratorResult contains the result of decorator execution
type DecoratorResult struct {
	Success    bool                   `json:"success"`
	Modified   interface{}            `json:"modified,omitempty"`
	Output     interface{}            `json:"output,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	Duration   time.Duration          `json:"duration"`
	Error      string                 `json:"error,omitempty"`
}

// DecoratorSchema defines the schema for decorator validation
type DecoratorSchema struct {
	Arguments  []ArgumentSchema       `json:"arguments,omitempty"`
	Properties map[string]PropertyDef `json:"properties,omitempty"`
	Required   []string              `json:"required,omitempty"`
}

// ArgumentSchema defines schema for decorator arguments
type ArgumentSchema struct {
	Name        string      `json:"name"`
	Type        string      `json:"type"`
	Required    bool        `json:"required"`
	Default     interface{} `json:"default,omitempty"`
	Description string      `json:"description,omitempty"`
}

// PropertyDef defines a property definition
type PropertyDef struct {
	Type        string      `json:"type"`
	Required    bool        `json:"required"`
	Default     interface{} `json:"default,omitempty"`
	Enum        []string    `json:"enum,omitempty"`
	Description string      `json:"description,omitempty"`
}

// PluginInterface defines the interface that plugins must implement
type PluginInterface interface {
	GetDecorators() []RegisteredDecorator
	Initialize(config map[string]interface{}) error
	Shutdown() error
}

// DefaultRegistryConfig returns the default configuration
func DefaultRegistryConfig() *RegistryConfig {
	return &RegistryConfig{
		PluginDirs:         []string{"./plugins", "../decorators"},
		PluginPattern:      "*.so",
		EnableHotReload:    false,
		ReloadInterval:     30 * time.Second,
		ParallelLoading:    true,
		LoadWorkers:        4,
		MaxDecorators:      1000,
		AllowOverride:      false,
		ValidateOnRegister: true,
		EnableMetrics:      true,
		RequireSignature:   false,
	}
}

// NewDecoratorRegistry creates a new decorator registry
func NewDecoratorRegistry(config *RegistryConfig) *DecoratorRegistry {
	if config == nil {
		config = DefaultRegistryConfig()
	}
	
	dr := &DecoratorRegistry{
		config:     config,
		decorators: make(map[string]*RegisteredDecorator),
		plugins:    make(map[string]*LoadedPlugin),
		handlers:   make(map[string]DecoratorHandler),
	}
	
	// Register built-in decorators
	dr.registerBuiltinDecorators()
	
	// Start hot reload if enabled
	if config.EnableHotReload {
		go dr.startHotReload()
	}
	
	return dr
}

// registerBuiltinDecorators registers built-in decorators
func (dr *DecoratorRegistry) registerBuiltinDecorators() {
	// REST decorators
	dr.Register(&RegisteredDecorator{
		Name:        "GET",
		Type:        "rest",
		Description: "HTTP GET endpoint",
		Handler:     dr.restHandler("GET"),
		Schema: &DecoratorSchema{
			Arguments: []ArgumentSchema{
				{Name: "path", Type: "string", Required: true},
			},
		},
	})
	
	dr.Register(&RegisteredDecorator{
		Name:        "POST",
		Type:        "rest",
		Description: "HTTP POST endpoint",
		Handler:     dr.restHandler("POST"),
		Schema: &DecoratorSchema{
			Arguments: []ArgumentSchema{
				{Name: "path", Type: "string", Required: true},
			},
		},
	})
	
	// Validation decorators
	dr.Register(&RegisteredDecorator{
		Name:        "Required",
		Type:        "validation",
		Description: "Field is required",
		Handler:     dr.validationHandler("required"),
	})
	
	dr.Register(&RegisteredDecorator{
		Name:        "MinLength",
		Type:        "validation",
		Description: "Minimum length validation",
		Handler:     dr.validationHandler("minlength"),
		Schema: &DecoratorSchema{
			Arguments: []ArgumentSchema{
				{Name: "length", Type: "int", Required: true},
			},
		},
	})
	
	// Security decorators
	dr.Register(&RegisteredDecorator{
		Name:        "Auth",
		Type:        "security",
		Description: "Authentication required",
		Handler:     dr.securityHandler("auth"),
	})
	
	dr.Register(&RegisteredDecorator{
		Name:        "RateLimit",
		Type:        "security",
		Description: "Rate limiting",
		Handler:     dr.securityHandler("ratelimit"),
		Schema: &DecoratorSchema{
			Arguments: []ArgumentSchema{
				{Name: "limit", Type: "int", Required: true},
				{Name: "window", Type: "duration", Required: false, Default: "1m"},
			},
		},
	})
}

// restHandler creates a REST decorator handler
func (dr *DecoratorRegistry) restHandler(method string) DecoratorHandler {
	return func(ctx context.Context, args DecoratorArgs) (DecoratorResult, error) {
		startTime := time.Now()
		
		// Extract path from arguments
		if len(args.Arguments) == 0 {
			return DecoratorResult{
				Success: false,
				Error:   "path argument required",
				Duration: time.Since(startTime),
			}, nil
		}
		
		path, ok := args.Arguments[0].(string)
		if !ok {
			return DecoratorResult{
				Success: false,
				Error:   "path must be a string",
				Duration: time.Since(startTime),
			}, nil
		}
		
		// Create REST endpoint metadata
		metadata := map[string]interface{}{
			"method": method,
			"path":   path,
			"handler": args.Target,
		}
		
		// Add properties if provided
		for key, value := range args.Properties {
			metadata[key] = value
		}
		
		return DecoratorResult{
			Success:  true,
			Modified: args.Target,
			Metadata: metadata,
			Duration: time.Since(startTime),
		}, nil
	}
}

// validationHandler creates a validation decorator handler
func (dr *DecoratorRegistry) validationHandler(validationType string) DecoratorHandler {
	return func(ctx context.Context, args DecoratorArgs) (DecoratorResult, error) {
		startTime := time.Now()
		
		metadata := map[string]interface{}{
			"validation_type": validationType,
			"target":         args.Target,
		}
		
		// Add validation parameters
		if len(args.Arguments) > 0 {
			metadata["parameters"] = args.Arguments
		}
		
		return DecoratorResult{
			Success:  true,
			Modified: args.Target,
			Metadata: metadata,
			Duration: time.Since(startTime),
		}, nil
	}
}

// securityHandler creates a security decorator handler
func (dr *DecoratorRegistry) securityHandler(securityType string) DecoratorHandler {
	return func(ctx context.Context, args DecoratorArgs) (DecoratorResult, error) {
		startTime := time.Now()
		
		metadata := map[string]interface{}{
			"security_type": securityType,
			"target":       args.Target,
		}
		
		// Add security parameters
		if len(args.Arguments) > 0 {
			metadata["parameters"] = args.Arguments
		}
		
		return DecoratorResult{
			Success:  true,
			Modified: args.Target,
			Metadata: metadata,
			Duration: time.Since(startTime),
		}, nil
	}
}

// LoadPlugins loads decorator plugins from configured directories
func (dr *DecoratorRegistry) LoadPlugins(ctx context.Context) error {
	if dr.config.ParallelLoading {
		return dr.loadPluginsParallel(ctx)
	}
	return dr.loadPluginsSequential(ctx)
}

// loadPluginsParallel loads plugins in parallel
func (dr *DecoratorRegistry) loadPluginsParallel(ctx context.Context) error {
	var pluginFiles []string
	
	// Find all plugin files
	for _, dir := range dr.config.PluginDirs {
		matches, err := filepath.Glob(filepath.Join(dir, dr.config.PluginPattern))
		if err != nil {
			continue
		}
		pluginFiles = append(pluginFiles, matches...)
	}
	
	if len(pluginFiles) == 0 {
		return nil
	}
	
	// Load plugins in parallel
	type loadResult struct {
		path   string
		plugin *LoadedPlugin
		err    error
	}
	
	resultChan := make(chan loadResult, len(pluginFiles))
	semaphore := make(chan struct{}, dr.config.LoadWorkers)
	
	var wg sync.WaitGroup
	for _, file := range pluginFiles {
		// Check if allowed
		if !dr.isPluginAllowed(file) {
			continue
		}
		
		wg.Add(1)
		go func(pluginPath string) {
			defer wg.Done()
			
			select {
			case <-ctx.Done():
				resultChan <- loadResult{path: pluginPath, err: ctx.Err()}
				return
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
			}
			
			loadedPlugin, err := dr.loadPlugin(pluginPath)
			resultChan <- loadResult{
				path:   pluginPath,
				plugin: loadedPlugin,
				err:    err,
			}
		}(file)
	}
	
	go func() {
		wg.Wait()
		close(resultChan)
	}()
	
	// Collect results
	var errors []string
	for result := range resultChan {
		if result.err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", result.path, result.err))
			dr.loadErrors.Store(result.path, result.err)
		} else if result.plugin != nil {
			dr.mu.Lock()
			dr.plugins[result.path] = result.plugin
			dr.mu.Unlock()
			atomic.AddInt64(&dr.loadedPlugins, 1)
		}
	}
	
	if len(errors) > 0 {
		return fmt.Errorf("plugin loading errors: %s", strings.Join(errors, "; "))
	}
	
	return nil
}

// loadPluginsSequential loads plugins sequentially
func (dr *DecoratorRegistry) loadPluginsSequential(ctx context.Context) error {
	for _, dir := range dr.config.PluginDirs {
		matches, err := filepath.Glob(filepath.Join(dir, dr.config.PluginPattern))
		if err != nil {
			continue
		}
		
		for _, file := range matches {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
			
			if !dr.isPluginAllowed(file) {
				continue
			}
			
			loadedPlugin, err := dr.loadPlugin(file)
			if err != nil {
				dr.loadErrors.Store(file, err)
				continue
			}
			
			dr.mu.Lock()
			dr.plugins[file] = loadedPlugin
			dr.mu.Unlock()
			atomic.AddInt64(&dr.loadedPlugins, 1)
		}
	}
	
	return nil
}

// loadPlugin loads a single plugin
func (dr *DecoratorRegistry) loadPlugin(path string) (*LoadedPlugin, error) {
	startTime := time.Now()
	
	// Mark as loading
	dr.loading.Store(path, true)
	defer dr.loading.Delete(path)
	
	// Open plugin
	p, err := plugin.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open plugin: %w", err)
	}
	
	// Look for required symbols
	getDecoratorsSym, err := p.Lookup("GetDecorators")
	if err != nil {
		return nil, fmt.Errorf("plugin missing GetDecorators: %w", err)
	}
	
	getDecorators, ok := getDecoratorsSym.(func() []RegisteredDecorator)
	if !ok {
		return nil, fmt.Errorf("GetDecorators has wrong signature")
	}
	
	// Get decorators from plugin
	decorators := getDecorators()
	
	// Create loaded plugin
	loaded := &LoadedPlugin{
		Path:       path,
		Name:       filepath.Base(path),
		Plugin:     p,
		Decorators: make([]string, 0, len(decorators)),
		LoadedAt:   time.Now(),
	}
	
	// Register decorators
	for _, decorator := range decorators {
		decorator.Plugin = path
		if err := dr.Register(&decorator); err != nil {
			// Log error but continue
			continue
		}
		loaded.Decorators = append(loaded.Decorators, decorator.Name)
	}
	
	// Update metrics
	duration := time.Since(startTime)
	atomic.AddInt64(&dr.loadDuration, int64(duration))
	
	return loaded, nil
}

// isPluginAllowed checks if a plugin is allowed to load
func (dr *DecoratorRegistry) isPluginAllowed(path string) bool {
	name := filepath.Base(path)
	
	// Check blocked list
	for _, blocked := range dr.config.BlockedPlugins {
		if matched, _ := filepath.Match(blocked, name); matched {
			return false
		}
	}
	
	// Check allowed list if specified
	if len(dr.config.AllowedPlugins) > 0 {
		for _, allowed := range dr.config.AllowedPlugins {
			if matched, _ := filepath.Match(allowed, name); matched {
				return true
			}
		}
		return false
	}
	
	return true
}

// Register registers a decorator
func (dr *DecoratorRegistry) Register(decorator *RegisteredDecorator) error {
	if decorator.Name == "" {
		return fmt.Errorf("decorator name is required")
	}
	
	dr.mu.Lock()
	defer dr.mu.Unlock()
	
	// Check if already registered
	if existing, exists := dr.decorators[decorator.Name]; exists {
		if !dr.config.AllowOverride {
			return fmt.Errorf("decorator %s already registered", decorator.Name)
		}
		// Keep some metadata from existing
		decorator.RegisteredAt = existing.RegisteredAt
	} else {
		decorator.RegisteredAt = time.Now()
	}
	
	// Validate if required
	if dr.config.ValidateOnRegister && decorator.Schema != nil {
		if err := dr.validateSchema(decorator.Schema); err != nil {
			return fmt.Errorf("invalid schema: %w", err)
		}
	}
	
	// Store decorator
	dr.decorators[decorator.Name] = decorator
	
	// Store handler if provided
	if decorator.Handler != nil {
		dr.handlers[decorator.Name] = decorator.Handler
	}
	
	atomic.AddInt64(&dr.registrations, 1)
	
	return nil
}

// validateSchema validates a decorator schema
func (dr *DecoratorRegistry) validateSchema(schema *DecoratorSchema) error {
	// Basic validation
	for _, arg := range schema.Arguments {
		if arg.Name == "" {
			return fmt.Errorf("argument name is required")
		}
		if arg.Type == "" {
			return fmt.Errorf("argument type is required")
		}
	}
	
	for name, prop := range schema.Properties {
		if prop.Type == "" {
			return fmt.Errorf("property %s type is required", name)
		}
	}
	
	return nil
}

// Get retrieves a registered decorator
func (dr *DecoratorRegistry) Get(name string) (*RegisteredDecorator, error) {
	dr.mu.RLock()
	defer dr.mu.RUnlock()
	
	decorator, exists := dr.decorators[name]
	if !exists {
		return nil, fmt.Errorf("decorator %s not found", name)
	}
	
	return decorator, nil
}

// Invoke invokes a decorator
func (dr *DecoratorRegistry) Invoke(ctx context.Context, name string, args DecoratorArgs) (DecoratorResult, error) {
	atomic.AddInt64(&dr.invocations, 1)
	
	// Get handler
	dr.mu.RLock()
	handler, exists := dr.handlers[name]
	dr.mu.RUnlock()
	
	if !exists {
		return DecoratorResult{
			Success: false,
			Error:   fmt.Sprintf("decorator %s not found", name),
		}, nil
	}
	
	// Invoke handler
	return handler(ctx, args)
}

// List returns all registered decorators
func (dr *DecoratorRegistry) List() []*RegisteredDecorator {
	dr.mu.RLock()
	defer dr.mu.RUnlock()
	
	decorators := make([]*RegisteredDecorator, 0, len(dr.decorators))
	for _, decorator := range dr.decorators {
		decorators = append(decorators, decorator)
	}
	
	// Sort by name
	sort.Slice(decorators, func(i, j int) bool {
		return decorators[i].Name < decorators[j].Name
	})
	
	return decorators
}

// ListByType returns decorators of a specific type
func (dr *DecoratorRegistry) ListByType(decoratorType string) []*RegisteredDecorator {
	dr.mu.RLock()
	defer dr.mu.RUnlock()
	
	var decorators []*RegisteredDecorator
	for _, decorator := range dr.decorators {
		if decorator.Type == decoratorType {
			decorators = append(decorators, decorator)
		}
	}
	
	return decorators
}

// ExportSchema exports the schema of all decorators
func (dr *DecoratorRegistry) ExportSchema(writer io.Writer) error {
	decorators := dr.List()
	
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	
	return encoder.Encode(decorators)
}

// startHotReload starts the hot reload monitor
func (dr *DecoratorRegistry) startHotReload() {
	ticker := time.NewTicker(dr.config.ReloadInterval)
	defer ticker.Stop()
	
	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		_ = dr.LoadPlugins(ctx)
		cancel()
	}
}

// GetStatistics returns registry statistics
func (dr *DecoratorRegistry) GetStatistics() map[string]interface{} {
	dr.mu.RLock()
	decoratorCount := len(dr.decorators)
	pluginCount := len(dr.plugins)
	handlerCount := len(dr.handlers)
	dr.mu.RUnlock()
	
	// Count errors
	errorCount := 0
	dr.loadErrors.Range(func(_, _ interface{}) bool {
		errorCount++
		return true
	})
	
	return map[string]interface{}{
		"decorators":        decoratorCount,
		"plugins":          pluginCount,
		"handlers":         handlerCount,
		"loaded_plugins":   atomic.LoadInt64(&dr.loadedPlugins),
		"registrations":    atomic.LoadInt64(&dr.registrations),
		"invocations":      atomic.LoadInt64(&dr.invocations),
		"load_duration_ns": atomic.LoadInt64(&dr.loadDuration),
		"load_errors":      errorCount,
	}
}

// Shutdown shuts down the registry
func (dr *DecoratorRegistry) Shutdown() error {
	dr.mu.Lock()
	defer dr.mu.Unlock()
	
	// Clear all data
	dr.decorators = make(map[string]*RegisteredDecorator)
	dr.plugins = make(map[string]*LoadedPlugin)
	dr.handlers = make(map[string]DecoratorHandler)
	
	return nil
}