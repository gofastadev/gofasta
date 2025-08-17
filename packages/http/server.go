package http

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/healtronlabs/gofasta/packages/core"
)

// HTTPServer represents the Gofasta HTTP server
type HTTPServer struct {
	router           *mux.Router
	server           *http.Server
	container        *core.DIContainer
	middleware       []MiddlewareFunc
	guards           []core.Guard
	pipes            []core.Pipe
	interceptors     []core.Interceptor
	exceptionFilters []core.ExceptionFilter
	staticDirs       map[string]string
	wsUpgrader       websocket.Upgrader
	config           *ServerConfig
	mutex            sync.RWMutex
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
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

// MiddlewareFunc represents HTTP middleware
type MiddlewareFunc func(http.Handler) http.Handler

// RequestContext represents HTTP request context
type RequestContext struct {
	Request        *http.Request
	Response       http.ResponseWriter
	Params         map[string]string
	Query          map[string][]string
	Body           []byte
	Application    core.Application
	Container      *core.DIContainer
	User           interface{}
	Data           map[string]interface{}
	StartTime      time.Time
	ResponseSent   bool
	StatusCode     int
}

// ResponseWriter wraps http.ResponseWriter with additional functionality
type ResponseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	StatusCode int    `json:"statusCode"`
	Message    string `json:"message"`
	Error      string `json:"error"`
	Timestamp  string `json:"timestamp"`
}

// NewHTTPServer creates a new HTTP server
func NewHTTPServer(container *core.DIContainer, config ...*ServerConfig) *HTTPServer {
	cfg := DefaultServerConfig()
	if len(config) > 0 && config[0] != nil {
		cfg = config[0]
	}

	router := mux.NewRouter()
	
	server := &HTTPServer{
		router:           router,
		container:        container,
		middleware:       make([]MiddlewareFunc, 0),
		guards:           make([]core.Guard, 0),
		pipes:            make([]core.Pipe, 0),
		interceptors:     make([]core.Interceptor, 0),
		exceptionFilters: make([]core.ExceptionFilter, 0),
		staticDirs:       make(map[string]string),
		config:           cfg,
		wsUpgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Configure as needed
			},
		},
	}

	// Apply default middleware
	server.applyDefaultMiddleware()

	return server
}

// DefaultServerConfig returns default server configuration
func DefaultServerConfig() *ServerConfig {
	return &ServerConfig{
		Host:            "localhost",
		Port:            8080,
		ReadTimeout:     15 * time.Second,
		WriteTimeout:    15 * time.Second,
		IdleTimeout:     60 * time.Second,
		MaxHeaderBytes:  1 << 20, // 1MB
		EnableGzip:      true,
		CORSEnabled:     true,
		CORSOrigins:     []string{"*"},
		CORSMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		CORSHeaders:     []string{"*"},
		StaticFileCache: 24 * time.Hour,
	}
}

// applyDefaultMiddleware applies default middleware stack
func (s *HTTPServer) applyDefaultMiddleware() {
	// Recovery middleware
	s.Use(s.recoveryMiddleware())
	
	// CORS middleware
	if s.config.CORSEnabled {
		s.Use(s.corsMiddleware())
	}
	
	// Gzip compression middleware
	if s.config.EnableGzip {
		s.Use(s.gzipMiddleware())
	}
	
	// Request context middleware
	s.Use(s.contextMiddleware())
}

// Use adds middleware to the server
func (s *HTTPServer) Use(middleware MiddlewareFunc) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.middleware = append(s.middleware, middleware)
}

// UseGuards adds guards to the server
func (s *HTTPServer) UseGuards(guards ...core.Guard) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.guards = append(s.guards, guards...)
}

// UsePipes adds pipes to the server
func (s *HTTPServer) UsePipes(pipes ...core.Pipe) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.pipes = append(s.pipes, pipes...)
}

// UseInterceptors adds interceptors to the server
func (s *HTTPServer) UseInterceptors(interceptors ...core.Interceptor) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.interceptors = append(s.interceptors, interceptors...)
}

// UseExceptionFilters adds exception filters to the server
func (s *HTTPServer) UseExceptionFilters(filters ...core.ExceptionFilter) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.exceptionFilters = append(s.exceptionFilters, filters...)
}

// RegisterController registers a controller with the HTTP server
func (s *HTTPServer) RegisterController(controller core.Controller) error {
	// Extract controller metadata
	metadata, err := core.ExtractControllerMetadata(controller)
	if err != nil {
		return fmt.Errorf("failed to extract controller metadata: %w", err)
	}

	// Register routes
	for _, route := range metadata.Routes {
		handler := s.createControllerHandler(controller, route)
		
		// Apply middleware, guards, pipes, and interceptors
		handler = s.applyMiddlewareStack(handler, route)
		
		s.router.HandleFunc(route.Path, handler).Methods(route.Method)
	}

	return nil
}

// RegisterRoute registers a single route
func (s *HTTPServer) RegisterRoute(method, path string, handler http.HandlerFunc) {
	s.router.HandleFunc(path, handler).Methods(method)
}

// GET registers a GET route
func (s *HTTPServer) GET(path string, handler http.HandlerFunc) {
	s.RegisterRoute("GET", path, handler)
}

// POST registers a POST route
func (s *HTTPServer) POST(path string, handler http.HandlerFunc) {
	s.RegisterRoute("POST", path, handler)
}

// PUT registers a PUT route
func (s *HTTPServer) PUT(path string, handler http.HandlerFunc) {
	s.RegisterRoute("PUT", path, handler)
}

// DELETE registers a DELETE route
func (s *HTTPServer) DELETE(path string, handler http.HandlerFunc) {
	s.RegisterRoute("DELETE", path, handler)
}

// PATCH registers a PATCH route
func (s *HTTPServer) PATCH(path string, handler http.HandlerFunc) {
	s.RegisterRoute("PATCH", path, handler)
}

// createControllerHandler creates a handler for a controller method
func (s *HTTPServer) createControllerHandler(controller core.Controller, route *core.RouteMetadata) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := s.createRequestContext(w, r)
		
		// Execute controller method
		result, err := s.executeControllerMethod(controller, route, ctx)
		if err != nil {
			s.handleError(ctx, err)
			return
		}
		
		// Send response
		s.sendResponse(ctx, result)
	}
}

// applyMiddlewareStack applies the complete middleware stack
func (s *HTTPServer) applyMiddlewareStack(handler http.HandlerFunc, route *core.RouteMetadata) http.HandlerFunc {
	// Apply interceptors (reverse order for proper nesting)
	for i := len(s.interceptors) - 1; i >= 0; i-- {
		handler = s.wrapWithInterceptor(handler, s.interceptors[i])
	}
	
	// Apply pipes
	for i := len(s.pipes) - 1; i >= 0; i-- {
		handler = s.wrapWithPipe(handler, s.pipes[i])
	}
	
	// Apply guards
	for i := len(s.guards) - 1; i >= 0; i-- {
		handler = s.wrapWithGuard(handler, s.guards[i])
	}
	
	// Apply global middleware
	finalHandler := http.Handler(handler)
	for i := len(s.middleware) - 1; i >= 0; i-- {
		finalHandler = s.middleware[i](finalHandler)
	}
	
	return finalHandler.ServeHTTP
}

// createRequestContext creates a new request context
func (s *HTTPServer) createRequestContext(w http.ResponseWriter, r *http.Request) *RequestContext {
	// Parse query parameters
	query := make(map[string][]string)
	for k, v := range r.URL.Query() {
		query[k] = v
	}
	
	// Extract route parameters
	params := make(map[string]string)
	if vars := mux.Vars(r); vars != nil {
		params = vars
	}
	
	// Read body
	body, _ := io.ReadAll(r.Body)
	r.Body.Close()
	
	return &RequestContext{
		Request:     r,
		Response:    &ResponseWriter{ResponseWriter: w, statusCode: 200},
		Params:      params,
		Query:       query,
		Body:        body,
		Container:   s.container,
		Data:        make(map[string]interface{}),
		StartTime:   time.Now(),
		StatusCode:  200,
	}
}

// executeControllerMethod executes a controller method
func (s *HTTPServer) executeControllerMethod(controller core.Controller, route *core.RouteMetadata, ctx *RequestContext) (interface{}, error) {
	// Get method by name
	method := reflect.ValueOf(controller).MethodByName(route.Handler)
	if !method.IsValid() {
		return nil, fmt.Errorf("handler method %s not found", route.Handler)
	}
	
	// Prepare method arguments
	args := []reflect.Value{reflect.ValueOf(ctx)}
	
	// Call method
	results := method.Call(args)
	
	// Handle results
	if len(results) == 0 {
		return nil, nil
	}
	
	if len(results) == 2 {
		// Return (result, error) pattern
		if !results[1].IsNil() {
			return nil, results[1].Interface().(error)
		}
		return results[0].Interface(), nil
	}
	
	// Single return value
	return results[0].Interface(), nil
}

// sendResponse sends the response
func (s *HTTPServer) sendResponse(ctx *RequestContext, result interface{}) {
	if ctx.ResponseSent {
		return
	}
	
	if result == nil {
		ctx.Response.WriteHeader(http.StatusNoContent)
		ctx.ResponseSent = true
		return
	}
	
	// Determine content type and serialize
	switch v := result.(type) {
	case string:
		ctx.Response.Header().Set("Content-Type", "text/plain")
		ctx.Response.WriteHeader(ctx.StatusCode)
		ctx.Response.Write([]byte(v))
	case []byte:
		ctx.Response.Header().Set("Content-Type", "application/octet-stream")
		ctx.Response.WriteHeader(ctx.StatusCode)
		ctx.Response.Write(v)
	default:
		// JSON serialization
		data, err := json.Marshal(result)
		if err != nil {
			s.handleError(ctx, err)
			return
		}
		ctx.Response.Header().Set("Content-Type", "application/json")
		ctx.Response.WriteHeader(ctx.StatusCode)
		ctx.Response.Write(data)
	}
	
	ctx.ResponseSent = true
}

// handleError handles errors
func (s *HTTPServer) handleError(ctx *RequestContext, err error) {
	if ctx.ResponseSent {
		return
	}
	
	// Try exception filters first
	for _, filter := range s.exceptionFilters {
		if response := filter.Catch(err, &core.RequestContext{
			Request: ctx.Request,
		}); response != nil {
			ctx.Response.Header().Set("Content-Type", "application/json")
			ctx.Response.WriteHeader(response.StatusCode)
			data, _ := json.Marshal(response)
			ctx.Response.Write(data)
			ctx.ResponseSent = true
			return
		}
	}
	
	// Default error handling
	errorResponse := &ErrorResponse{
		StatusCode: http.StatusInternalServerError,
		Message:    "Internal Server Error",
		Error:      err.Error(),
		Timestamp:  time.Now().Format(time.RFC3339),
	}
	
	// Check for Gofasta exceptions
	if gofastaErr, ok := err.(*core.GofastaError); ok {
		errorResponse.StatusCode = gofastaErr.StatusCode
		errorResponse.Message = gofastaErr.Message
		errorResponse.Error = gofastaErr.Error()
	}
	
	ctx.Response.Header().Set("Content-Type", "application/json")
	ctx.Response.WriteHeader(errorResponse.StatusCode)
	
	data, _ := json.Marshal(errorResponse)
	ctx.Response.Write(data)
	ctx.ResponseSent = true
}

// Static serves static files
func (s *HTTPServer) Static(prefix, dir string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	s.staticDirs[prefix] = dir
	
	fileServer := http.FileServer(http.Dir(dir))
	handler := http.StripPrefix(prefix, fileServer)
	
	// Apply caching and compression
	handler = s.staticFileMiddleware(handler)
	
	s.router.PathPrefix(prefix).Handler(handler)
}

// Listen starts the HTTP server
func (s *HTTPServer) Listen() error {
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	
	s.server = &http.Server{
		Addr:           addr,
		Handler:        s.router,
		ReadTimeout:    s.config.ReadTimeout,
		WriteTimeout:   s.config.WriteTimeout,
		IdleTimeout:    s.config.IdleTimeout,
		MaxHeaderBytes: s.config.MaxHeaderBytes,
	}
	
	fmt.Printf("🌐 HTTP server listening on http://%s\n", addr)
	
	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start HTTP server: %w", err)
	}
	
	return nil
}

// ListenTLS starts the HTTPS server
func (s *HTTPServer) ListenTLS(certFile, keyFile string) error {
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	
	s.server = &http.Server{
		Addr:           addr,
		Handler:        s.router,
		ReadTimeout:    s.config.ReadTimeout,
		WriteTimeout:   s.config.WriteTimeout,
		IdleTimeout:    s.config.IdleTimeout,
		MaxHeaderBytes: s.config.MaxHeaderBytes,
	}
	
	fmt.Printf("🔐 HTTPS server listening on https://%s\n", addr)
	
	if err := s.server.ListenAndServeTLS(certFile, keyFile); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start HTTPS server: %w", err)
	}
	
	return nil
}

// Shutdown gracefully shuts down the server
func (s *HTTPServer) Shutdown(ctx context.Context) error {
	if s.server == nil {
		return nil
	}
	
	fmt.Println("🛑 Shutting down HTTP server...")
	return s.server.Shutdown(ctx)
}

// GetRouter returns the underlying router
func (s *HTTPServer) GetRouter() *mux.Router {
	return s.router
}

// GetConfig returns the server configuration
func (s *HTTPServer) GetConfig() *ServerConfig {
	return s.config
}

// WriteHeader sets the status code
func (rw *ResponseWriter) WriteHeader(code int) {
	if !rw.written {
		rw.statusCode = code
		rw.ResponseWriter.WriteHeader(code)
		rw.written = true
	}
}

// Write writes data to the response
func (rw *ResponseWriter) Write(data []byte) (int, error) {
	if !rw.written {
		rw.WriteHeader(200)
	}
	return rw.ResponseWriter.Write(data)
}

// StatusCode returns the status code
func (rw *ResponseWriter) StatusCode() int {
	return rw.statusCode
}

// JSON sends a JSON response
func (ctx *RequestContext) JSON(statusCode int, data interface{}) {
	ctx.StatusCode = statusCode
	ctx.Response.Header().Set("Content-Type", "application/json")
	ctx.Response.WriteHeader(statusCode)
	
	if err := json.NewEncoder(ctx.Response).Encode(data); err != nil {
		http.Error(ctx.Response, err.Error(), http.StatusInternalServerError)
	}
	ctx.ResponseSent = true
}

// Text sends a text response
func (ctx *RequestContext) Text(statusCode int, text string) {
	ctx.StatusCode = statusCode
	ctx.Response.Header().Set("Content-Type", "text/plain")
	ctx.Response.WriteHeader(statusCode)
	ctx.Response.Write([]byte(text))
	ctx.ResponseSent = true
}

// HTML sends an HTML response
func (ctx *RequestContext) HTML(statusCode int, html string) {
	ctx.StatusCode = statusCode
	ctx.Response.Header().Set("Content-Type", "text/html")
	ctx.Response.WriteHeader(statusCode)
	ctx.Response.Write([]byte(html))
	ctx.ResponseSent = true
}

// Redirect redirects the request
func (ctx *RequestContext) Redirect(statusCode int, url string) {
	http.Redirect(ctx.Response, ctx.Request, url, statusCode)
	ctx.ResponseSent = true
}

// GetParam gets a route parameter
func (ctx *RequestContext) GetParam(key string) string {
	return ctx.Params[key]
}

// GetQuery gets a query parameter
func (ctx *RequestContext) GetQuery(key string) string {
	values := ctx.Query[key]
	if len(values) > 0 {
		return values[0]
	}
	return ""
}

// GetQueryArray gets query parameter array
func (ctx *RequestContext) GetQueryArray(key string) []string {
	return ctx.Query[key]
}

// ParseJSON parses JSON from request body
func (ctx *RequestContext) ParseJSON(v interface{}) error {
	return json.Unmarshal(ctx.Body, v)
}

// GetHeader gets a request header
func (ctx *RequestContext) GetHeader(key string) string {
	return ctx.Request.Header.Get(key)
}

// SetHeader sets a response header
func (ctx *RequestContext) SetHeader(key, value string) {
	ctx.Response.Header().Set(key, value)
}

