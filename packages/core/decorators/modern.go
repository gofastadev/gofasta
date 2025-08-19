package decorators

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

// Decorator parsing regex patterns
var (
	controllerPattern = regexp.MustCompile(`@Controller\(([^)]*)\)`)
	routePattern      = regexp.MustCompile(`@(Get|Post|Put|Delete|Patch|Options|Head)\(([^)]*)\)`)
	middlewarePattern = regexp.MustCompile(`@UseMiddleware\(([^)]*)\)`)
	guardsPattern     = regexp.MustCompile(`@UseGuards\(([^)]*)\)`)
	pipesPattern      = regexp.MustCompile(`@UsePipes\(([^)]*)\)`)
	filtersPattern    = regexp.MustCompile(`@UseFilters\(([^)]*)\)`)
	bodyPattern       = regexp.MustCompile(`@Body\(([^)]*)\)`)
	paramPattern      = regexp.MustCompile(`@Param\(([^)]*)\)`)
	queryPattern      = regexp.MustCompile(`@Query\(([^)]*)\)`)
	headerPattern     = regexp.MustCompile(`@Header\(([^)]*)\)`)
	statusPattern     = regexp.MustCompile(`@HttpCode\(([^)]*)\)`)
	versionPattern    = regexp.MustCompile(`@Version\(([^)]*)\)`)
)

// ExtractModernDecorators extracts Gofasta-style decorators from multiple sources
func ExtractModernDecorators(instance interface{}) (*ControllerDecoratorMetadata, map[string]*RouteDecoratorMetadata, error) {
	// Get the type information
	instanceType := reflect.TypeOf(instance)
	if instanceType.Kind() == reflect.Ptr {
		instanceType = instanceType.Elem()
	}

	// First, check the programmatic registry
	controllerMeta, routesMeta, found := GetControllerMetadata(instanceType)
	if found {
		return controllerMeta, routesMeta, nil
	}

	// Try to find the source file for this type and parse comments
	sourceFile, err := findSourceFile(instanceType)
	if err != nil {
		// Fallback to convention-based routing if source not found
		return extractConventionBased(instanceType)
	}

	// Parse the Go source file
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, sourceFile, nil, parser.ParseComments)
	if err != nil {
		return extractConventionBased(instanceType)
	}

	// Extract controller metadata from comments
	controllerMeta = extractControllerDecorators(node, instanceType.Name())
	
	// Extract route metadata for each method from comments
	routesMeta = extractRouteDecorators(node, instanceType)

	return controllerMeta, routesMeta, nil
}

// findSourceFile attempts to find the source file for a given type
func findSourceFile(t reflect.Type) (string, error) {
	// This is a simplified implementation
	// In a real implementation, you might use build constraints or other mechanisms
	pkgPath := t.PkgPath()
	if pkgPath == "" {
		return "", fmt.Errorf("cannot determine package path for type %s", t.Name())
	}
	
	// For this implementation, we'll assume the source is in the current directory
	// In production, you'd want a more sophisticated approach
	return "./" + strings.ToLower(t.Name()) + "_controller.go", fmt.Errorf("source file location not implemented")
}

// extractConventionBased provides fallback convention-based routing
func extractConventionBased(instanceType reflect.Type) (*ControllerDecoratorMetadata, map[string]*RouteDecoratorMetadata, error) {
	controllerMeta := &ControllerDecoratorMetadata{
		Prefix: "/" + strings.ToLower(strings.TrimSuffix(instanceType.Name(), "Controller")),
	}

	routesMeta := make(map[string]*RouteDecoratorMetadata)

	// Extract routes from method names
	for i := 0; i < instanceType.NumMethod(); i++ {
		method := instanceType.Method(i)
		if !method.IsExported() {
			continue
		}

		routeMeta := extractRouteFromMethodName(method.Name)
		if routeMeta != nil {
			routesMeta[method.Name] = routeMeta
		}
	}

	return controllerMeta, routesMeta, nil
}

// extractControllerDecorators extracts @Controller decorator from AST
func extractControllerDecorators(node *ast.File, typeName string) *ControllerDecoratorMetadata {
	controllerMeta := &ControllerDecoratorMetadata{}

	// Find the type declaration
	for _, decl := range node.Decls {
		if genDecl, ok := decl.(*ast.GenDecl); ok {
			for _, spec := range genDecl.Specs {
				if typeSpec, ok := spec.(*ast.TypeSpec); ok && typeSpec.Name.Name == typeName {
					// Look for comments above the type declaration
					if genDecl.Doc != nil {
						for _, comment := range genDecl.Doc.List {
							parseControllerComment(comment.Text, controllerMeta)
						}
					}
				}
			}
		}
	}

	return controllerMeta
}

// extractRouteDecorators extracts route decorators from method comments
func extractRouteDecorators(node *ast.File, instanceType reflect.Type) map[string]*RouteDecoratorMetadata {
	routesMeta := make(map[string]*RouteDecoratorMetadata)

	// Find method declarations
	for _, decl := range node.Decls {
		if funcDecl, ok := decl.(*ast.FuncDecl); ok {
			// Check if this is a method of our type
			if funcDecl.Recv != nil && len(funcDecl.Recv.List) > 0 {
				methodName := funcDecl.Name.Name
				routeMeta := &RouteDecoratorMetadata{}

				// Parse comments above the method
				if funcDecl.Doc != nil {
					for _, comment := range funcDecl.Doc.List {
						parseRouteComment(comment.Text, routeMeta)
					}
				}

				// If no decorators found, use convention
				if routeMeta.Method == "" {
					conventionRoute := extractRouteFromMethodName(methodName)
					if conventionRoute != nil {
						routeMeta = conventionRoute
					}
				}

				if routeMeta.Method != "" {
					routesMeta[methodName] = routeMeta
				}
			}
		}
	}

	return routesMeta
}

// parseControllerComment parses @Controller decorator from comment
func parseControllerComment(comment string, meta *ControllerDecoratorMetadata) {
	comment = strings.TrimSpace(strings.TrimPrefix(comment, "//"))

	// @Controller("prefix")
	if matches := controllerPattern.FindStringSubmatch(comment); len(matches) > 1 {
		prefix := strings.Trim(matches[1], `"'`)
		meta.Prefix = prefix
	}

	// @UseMiddleware("auth", "cors")
	if matches := middlewarePattern.FindStringSubmatch(comment); len(matches) > 1 {
		meta.Middleware = parseStringArray(matches[1])
	}

	// @UseGuards("auth", "admin")
	if matches := guardsPattern.FindStringSubmatch(comment); len(matches) > 1 {
		meta.Guards = parseStringArray(matches[1])
	}

	// @Version("v1")
	if matches := versionPattern.FindStringSubmatch(comment); len(matches) > 1 {
		meta.Version = strings.Trim(matches[1], `"'`)
	}
}

// parseRouteComment parses route decorators from method comments
func parseRouteComment(comment string, meta *RouteDecoratorMetadata) {
	comment = strings.TrimSpace(strings.TrimPrefix(comment, "//"))

	// @Get("path"), @Post("path"), etc.
	if matches := routePattern.FindStringSubmatch(comment); len(matches) > 1 {
		meta.Method = strings.ToUpper(matches[1])
		if len(matches) > 2 {
			meta.Path = strings.Trim(matches[2], `"'`)
		}
	}

	// @UseMiddleware("auth", "validation")
	if matches := middlewarePattern.FindStringSubmatch(comment); len(matches) > 1 {
		meta.Middleware = parseStringArray(matches[1])
	}

	// @UseGuards("auth")
	if matches := guardsPattern.FindStringSubmatch(comment); len(matches) > 1 {
		meta.Guards = parseStringArray(matches[1])
	}

	// @UsePipes("validation")
	if matches := pipesPattern.FindStringSubmatch(comment); len(matches) > 1 {
		meta.Pipes = parseStringArray(matches[1])
	}

	// @UseFilters("http-exception")
	if matches := filtersPattern.FindStringSubmatch(comment); len(matches) > 1 {
		meta.Filters = parseStringArray(matches[1])
	}

	// @HttpCode(201)
	if matches := statusPattern.FindStringSubmatch(comment); len(matches) > 1 {
		if code, err := strconv.Atoi(strings.TrimSpace(matches[1])); err == nil {
			meta.StatusCode = code
		}
	}
}

// extractRouteFromMethodName extracts route info from method name (fallback)
func extractRouteFromMethodName(methodName string) *RouteDecoratorMetadata {
	methodNameLower := strings.ToLower(methodName)

	route := &RouteDecoratorMetadata{}

	if strings.HasPrefix(methodNameLower, "get") {
		route.Method = "GET"
		route.Path = "/" + strings.ToLower(strings.TrimPrefix(methodName, "Get"))
	} else if strings.HasPrefix(methodNameLower, "post") {
		route.Method = "POST"
		route.Path = "/" + strings.ToLower(strings.TrimPrefix(methodName, "Post"))
	} else if strings.HasPrefix(methodNameLower, "put") {
		route.Method = "PUT"
		route.Path = "/" + strings.ToLower(strings.TrimPrefix(methodName, "Put"))
	} else if strings.HasPrefix(methodNameLower, "delete") {
		route.Method = "DELETE"
		route.Path = "/" + strings.ToLower(strings.TrimPrefix(methodName, "Delete"))
	} else if strings.HasPrefix(methodNameLower, "patch") {
		route.Method = "PATCH"
		route.Path = "/" + strings.ToLower(strings.TrimPrefix(methodName, "Patch"))
	} else {
		// Skip non-route methods
		return nil
	}

	// Clean up path
	if route.Path == "/" {
		route.Path = ""
	}

	return route
}

// parseStringArray parses comma-separated string array from decorator parameter
func parseStringArray(param string) []string {
	param = strings.TrimSpace(param)
	if param == "" {
		return []string{}
	}

	// Split by comma and clean up
	parts := strings.Split(param, ",")
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

// BuildFullPath combines controller prefix with route path
func BuildFullPath(controllerMeta *ControllerDecoratorMetadata, routeMeta *RouteDecoratorMetadata) string {
	prefix := controllerMeta.Prefix
	path := routeMeta.Path

	// Handle versioning
	if controllerMeta.Version != "" {
		prefix = "/" + controllerMeta.Version + prefix
	}

	// Combine paths
	fullPath := prefix + path

	// Clean up path
	fullPath = strings.ReplaceAll(fullPath, "//", "/")
	if fullPath == "" {
		fullPath = "/"
	}

	return fullPath
}

// GetCombinedMiddleware combines controller and route middleware
func GetCombinedMiddleware(controllerMeta *ControllerDecoratorMetadata, routeMeta *RouteDecoratorMetadata) []string {
	var middleware []string
	
	// Add controller middleware first
	middleware = append(middleware, controllerMeta.Middleware...)
	
	// Add route-specific middleware
	middleware = append(middleware, routeMeta.Middleware...)
	
	return middleware
}

// GetCombinedGuards combines controller and route guards
func GetCombinedGuards(controllerMeta *ControllerDecoratorMetadata, routeMeta *RouteDecoratorMetadata) []string {
	var guards []string
	
	// Add controller guards first
	guards = append(guards, controllerMeta.Guards...)
	
	// Add route-specific guards
	guards = append(guards, routeMeta.Guards...)
	
	return guards
}