package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/healtronlabs/gofasta/packages/core"
)

// HTTPServer represents the Gofasta HTTP server
type HTTPServer struct {
	router           *mux.Router
	server           *http.Server
	container        *core.DIContainer
	middleware       []core.MiddlewareFunc
	guards           []core.Guard
	pipes            []core.Pipe
	interceptors     []core.Interceptor
	exceptionHandler *core.ExceptionHandler
}

// NewHTTPServer creates a new HTTP server
func NewHTTPServer(container *core.DIContainer) *HTTPServer {
	router := mux.NewRouter()

	return &HTTPServer{
		router:           router,
		container:        container,
		middleware:       make([]core.MiddlewareFunc, 0),
		guards:           make([]core.Guard, 0),
		pipes:            make([]core.Pipe, 0),
		interceptors:     make([]core.Interceptor, 0),
		exceptionHandler: core.NewExceptionHandler(),
	}
}

// RegisterController registers a controller with the HTTP server
func (s *HTTPServer) RegisterController(controller core.Controller) error {
	routes, err := core.ExtractRouteMetadata(controller)
	if err != nil {
		return fmt.Errorf("failed to extract route metadata: %w", err)
	}

	for _, route := range routes {
		s.registerRoute(route, controller)
	}

	return nil
}

// registerRoute registers a single route with the router
func (s *HTTPServer) registerRoute(route *core.RouteMetadata, controller core.Controller) {
	handler := s.createRouteHandler(route, controller)
	s.router.HandleFunc(route.Path, handler).Methods(route.Method)
}

// createRouteHandler creates an HTTP handler for a route
func (s *HTTPServer) createRouteHandler(route *core.RouteMetadata, controller core.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Create request context
		ctx := &core.RequestContext{
			Request:        r,
			ResponseWriter: w,
			Context:        r.Context(),
			Params:         mux.Vars(r),
			Query:          s.extractQueryParams(r),
			Headers:        s.extractHeaders(r),
		}

		// Execute pipeline
		response := s.executePipeline(ctx, route, controller)

		// Write response
		s.writeResponse(w, response)
	}
}

// executePipeline executes the request processing pipeline
func (s *HTTPServer) executePipeline(ctx *core.RequestContext, route *core.RouteMetadata, controller core.Controller) *core.Response {
	defer func() {
		if err := recover(); err != nil {
			// Handle panic and convert to response
			response := s.exceptionHandler.Handle(err)
			s.writeResponse(ctx.ResponseWriter, response)
		}
	}()

	// 1. Execute global guards
	for _, guard := range s.guards {
		if !guard.CanActivate(ctx) {
			return &core.Response{
				StatusCode: 401,
				Body:       map[string]string{"error": "Unauthorized"},
			}
		}
	}

	// 2. Execute route-specific guards
	for _, guard := range route.Guards {
		if !guard.CanActivate(ctx) {
			return &core.Response{
				StatusCode: 403,
				Body:       map[string]string{"error": "Forbidden"},
			}
		}
	}

	// 3. Execute global pipes (validation, transformation)
	for _, pipe := range s.pipes {
		transformedValue, err := pipe.Transform(ctx.Body, &core.PipeMetadata{
			Type: "global",
		})
		if err != nil {
			return s.exceptionHandler.Handle(err)
		}
		ctx.Body = transformedValue
	}

	// 4. Execute route-specific pipes
	for _, pipe := range route.Pipes {
		transformedValue, err := pipe.Transform(ctx.Body, &core.PipeMetadata{
			Type: "route",
		})
		if err != nil {
			return s.exceptionHandler.Handle(err)
		}
		ctx.Body = transformedValue
	}

	// 5. Execute handler with interceptors
	handler := func(ctx *core.RequestContext) *core.Response {
		return s.invokeControllerMethod(route, controller, ctx)
	}

	// Apply interceptors in reverse order
	for i := len(s.interceptors) - 1; i >= 0; i-- {
		interceptor := s.interceptors[i]
		currentHandler := handler
		handler = func(ctx *core.RequestContext) *core.Response {
			return interceptor.Intercept(ctx, currentHandler)
		}
	}

	// Apply route-specific interceptors
	for i := len(route.Interceptors) - 1; i >= 0; i-- {
		interceptor := route.Interceptors[i]
		currentHandler := handler
		handler = func(ctx *core.RequestContext) *core.Response {
			return interceptor.Intercept(ctx, currentHandler)
		}
	}

	return handler(ctx)
}

// invokeControllerMethod invokes the appropriate controller method
func (s *HTTPServer) invokeControllerMethod(route *core.RouteMetadata, controller core.Controller, ctx *core.RequestContext) *core.Response {
	controllerValue := reflect.ValueOf(controller)
	method := controllerValue.MethodByName(route.Handler.Name)

	if !method.IsValid() {
		return &core.Response{
			StatusCode: 500,
			Body:       map[string]string{"error": "Method not found"},
		}
	}

	// Prepare method arguments
	args := s.prepareMethodArguments(method.Type(), ctx)

	// Invoke method
	results := method.Call(args)

	// Process results
	return s.processMethodResults(results)
}

// prepareMethodArguments prepares arguments for method invocation
func (s *HTTPServer) prepareMethodArguments(methodType reflect.Type, ctx *core.RequestContext) []reflect.Value {
	args := make([]reflect.Value, methodType.NumIn())

	for i := 0; i < methodType.NumIn(); i++ {
		paramType := methodType.In(i)

		// Handle different parameter types
		switch paramType.String() {
		case "*core.RequestContext":
			args[i] = reflect.ValueOf(ctx)
		default:
			// Try to extract from request based on parameter name/type
			// This would be enhanced with proper parameter binding
			args[i] = reflect.Zero(paramType)
		}
	}

	return args
}

// processMethodResults processes the results from method invocation
func (s *HTTPServer) processMethodResults(results []reflect.Value) *core.Response {
	if len(results) == 0 {
		return &core.Response{StatusCode: 204}
	}

	// Handle (data, error) pattern
	if len(results) == 2 {
		errorValue := results[1]
		if !errorValue.IsNil() {
			err := errorValue.Interface().(error)
			return s.exceptionHandler.Handle(err)
		}

		dataValue := results[0]
		return &core.Response{
			StatusCode: 200,
			Body:       dataValue.Interface(),
		}
	}

	// Handle single return value
	result := results[0]
	if result.Type().Implements(reflect.TypeOf((*error)(nil)).Elem()) {
		if !result.IsNil() {
			err := result.Interface().(error)
			return s.exceptionHandler.Handle(err)
		}
		return &core.Response{StatusCode: 204}
	}

	return &core.Response{
		StatusCode: 200,
		Body:       result.Interface(),
	}
}

// writeResponse writes the response to the HTTP response writer
func (s *HTTPServer) writeResponse(w http.ResponseWriter, response *core.Response) {
	// Set headers
	if response.Headers != nil {
		for key, value := range response.Headers {
			w.Header().Set(key, value)
		}
	}

	// Set content type if not already set
	if w.Header().Get("Content-Type") == "" {
		w.Header().Set("Content-Type", "application/json")
	}

	// Set status code
	w.WriteHeader(response.StatusCode)

	// Write body
	if response.Body != nil {
		if err := json.NewEncoder(w).Encode(response.Body); err != nil {
			// Log error (would use logging module in real implementation)
			fmt.Printf("Error encoding response: %v\n", err)
		}
	}
}

// extractQueryParams extracts query parameters from the request
func (s *HTTPServer) extractQueryParams(r *http.Request) map[string]string {
	params := make(map[string]string)
	for key, values := range r.URL.Query() {
		if len(values) > 0 {
			params[key] = values[0]
		}
	}
	return params
}

// extractHeaders extracts headers from the request
func (s *HTTPServer) extractHeaders(r *http.Request) map[string]string {
	headers := make(map[string]string)
	for key, values := range r.Header {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}
	return headers
}

// UseGlobalMiddleware adds global middleware
func (s *HTTPServer) UseGlobalMiddleware(middleware ...core.MiddlewareFunc) {
	s.middleware = append(s.middleware, middleware...)
}

// UseGlobalGuards adds global guards
func (s *HTTPServer) UseGlobalGuards(guards ...core.Guard) {
	s.guards = append(s.guards, guards...)
}

// UseGlobalPipes adds global pipes
func (s *HTTPServer) UseGlobalPipes(pipes ...core.Pipe) {
	s.pipes = append(s.pipes, pipes...)
}

// UseGlobalInterceptors adds global interceptors
func (s *HTTPServer) UseGlobalInterceptors(interceptors ...core.Interceptor) {
	s.interceptors = append(s.interceptors, interceptors...)
}

// Listen starts the HTTP server on the specified port
func (s *HTTPServer) Listen(port int) error {
	s.server = &http.Server{
		Addr:         ":" + strconv.Itoa(port),
		Handler:      s.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	fmt.Printf("Gofasta HTTP server listening on port %d\n", port)
	return s.server.ListenAndServe()
}

// Shutdown gracefully shuts down the HTTP server
func (s *HTTPServer) Shutdown(ctx context.Context) error {
	if s.server != nil {
		return s.server.Shutdown(ctx)
	}
	return nil
}
