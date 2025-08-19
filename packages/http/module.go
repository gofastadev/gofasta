package http

import (
	"context"
	"reflect"
	"time"

	"github.com/healtronlabs/gofasta/packages/core"
)

// HTTPModule represents the HTTP module for Gofasta
type HTTPModule struct {
	*core.BaseModule
	server *HTTPServer
	config *ServerConfig
}

// NewHTTPModule creates a new HTTP module
func NewHTTPModule(config ...*ServerConfig) *HTTPModule {
	cfg := DefaultServerConfig()
	if len(config) > 0 && config[0] != nil {
		cfg = config[0]
	}

	module := &HTTPModule{
		BaseModule: core.NewBaseModule(),
		config:     cfg,
	}

	return module
}

// Configure configures the HTTP module
func (m *HTTPModule) Configure(container *core.DIContainer) error {
	// Register HTTP server as a service
	m.server = NewHTTPServer(container, m.config)
	
	// Register the server instance
	if err := container.RegisterInstance(reflect.TypeOf((*HTTPServer)(nil)).Elem(), m.server); err != nil {
		return err
	}

	return m.BaseModule.Configure(container)
}

// Initialize initializes the HTTP module
func (m *HTTPModule) Initialize() error {
	if err := m.BaseModule.Initialize(); err != nil {
		return err
	}

	// HTTP server is ready but not started yet
	return nil
}

// Cleanup cleans up the HTTP module
func (m *HTTPModule) Cleanup() error {
	if m.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		
		if err := m.server.Shutdown(ctx); err != nil {
			return err
		}
	}

	return m.BaseModule.Cleanup()
}

// GetServer returns the HTTP server instance
func (m *HTTPModule) GetServer() *HTTPServer {
	return m.server
}

// GetConfig returns the module configuration
func (m *HTTPModule) GetConfig() *ServerConfig {
	return m.config
}

// Listen starts the HTTP server
func (m *HTTPModule) Listen(port ...int) error {
	if len(port) > 0 {
		m.config.Port = port[0]
	}
	
	return m.server.Listen()
}

// ListenTLS starts the HTTPS server
func (m *HTTPModule) ListenTLS(certFile, keyFile string, port ...int) error {
	if len(port) > 0 {
		m.config.Port = port[0]
	}
	
	return m.server.ListenTLS(certFile, keyFile)
}

// HTTPModuleBuilder provides a fluent API for building HTTP modules
type HTTPModuleBuilder struct {
	config *ServerConfig
}

// NewHTTPModuleBuilder creates a new HTTP module builder
func NewHTTPModuleBuilder() *HTTPModuleBuilder {
	return &HTTPModuleBuilder{
		config: DefaultServerConfig(),
	}
}

// WithHost sets the server host
func (b *HTTPModuleBuilder) WithHost(host string) *HTTPModuleBuilder {
	b.config.Host = host
	return b
}

// WithPort sets the server port
func (b *HTTPModuleBuilder) WithPort(port int) *HTTPModuleBuilder {
	b.config.Port = port
	return b
}

// WithTimeouts sets the server timeouts
func (b *HTTPModuleBuilder) WithTimeouts(read, write, idle time.Duration) *HTTPModuleBuilder {
	b.config.ReadTimeout = read
	b.config.WriteTimeout = write
	b.config.IdleTimeout = idle
	return b
}

// WithGzip enables or disables gzip compression
func (b *HTTPModuleBuilder) WithGzip(enabled bool) *HTTPModuleBuilder {
	b.config.EnableGzip = enabled
	return b
}

// WithCORS configures CORS settings
func (b *HTTPModuleBuilder) WithCORS(enabled bool, origins, methods, headers []string) *HTTPModuleBuilder {
	b.config.CORSEnabled = enabled
	if origins != nil {
		b.config.CORSOrigins = origins
	}
	if methods != nil {
		b.config.CORSMethods = methods
	}
	if headers != nil {
		b.config.CORSHeaders = headers
	}
	return b
}

// WithStaticFileCache sets the static file cache duration
func (b *HTTPModuleBuilder) WithStaticFileCache(duration time.Duration) *HTTPModuleBuilder {
	b.config.StaticFileCache = duration
	return b
}

// Build creates the HTTP module
func (b *HTTPModuleBuilder) Build() *HTTPModule {
	return NewHTTPModule(b.config)
}

// HTTPModuleOptions provides options for HTTP module
type HTTPModuleOptions struct {
	Host            string
	Port            int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	MaxHeaderBytes  int
	EnableGzip      bool
	CORSEnabled     bool
	CORSOrigins     []string
	CORSMethods     []string
	CORSHeaders     []string
	StaticFileCache time.Duration
}

// CreateHTTPModule creates an HTTP module with options
func CreateHTTPModule(opts *HTTPModuleOptions) *HTTPModule {
	config := DefaultServerConfig()
	
	if opts != nil {
		if opts.Host != "" {
			config.Host = opts.Host
		}
		if opts.Port != 0 {
			config.Port = opts.Port
		}
		if opts.ReadTimeout > 0 {
			config.ReadTimeout = opts.ReadTimeout
		}
		if opts.WriteTimeout > 0 {
			config.WriteTimeout = opts.WriteTimeout
		}
		if opts.IdleTimeout > 0 {
			config.IdleTimeout = opts.IdleTimeout
		}
		if opts.MaxHeaderBytes > 0 {
			config.MaxHeaderBytes = opts.MaxHeaderBytes
		}
		config.EnableGzip = opts.EnableGzip
		config.CORSEnabled = opts.CORSEnabled
		if opts.CORSOrigins != nil {
			config.CORSOrigins = opts.CORSOrigins
		}
		if opts.CORSMethods != nil {
			config.CORSMethods = opts.CORSMethods
		}
		if opts.CORSHeaders != nil {
			config.CORSHeaders = opts.CORSHeaders
		}
		if opts.StaticFileCache > 0 {
			config.StaticFileCache = opts.StaticFileCache
		}
	}
	
	return NewHTTPModule(config)
}