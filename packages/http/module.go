package http

import (
	"github.com/healtronlabs/gofasta/packages/core"
)

// HTTPModule provides HTTP server capabilities
type HTTPModule struct {
	*core.BaseModule
	server *HTTPServer
	port   int
}

// NewHTTPModule creates a new HTTP module
func NewHTTPModule(port int) *HTTPModule {
	return &HTTPModule{
		BaseModule: core.NewBaseModule(),
		port:       port,
	}
}

// Configure configures the HTTP module
func (m *HTTPModule) Configure(container *core.DIContainer) error {
	// Create HTTP server
	m.server = NewHTTPServer(container)
	
	// Register HTTP server as a provider
	container.RegisterProvider(m.server)
	
	return nil
}

// GetHTTPServer returns the HTTP server instance
func (m *HTTPModule) GetHTTPServer() *HTTPServer {
	return m.server
}

// GetPort returns the configured port
func (m *HTTPModule) GetPort() int {
	return m.port
}