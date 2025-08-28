package codegen

import (
	"fmt"
	"strings"
)

// generateTestSuiteDeclaration generates Go code for a test suite
func (g *CodeGenerator) generateTestSuiteDeclaration(testSuite *TestSuiteDeclaration) error {
	// Generate test suite struct
	g.writeLine(fmt.Sprintf("type %s struct {", testSuite.Name))
	g.indent()
	g.writeLine("suite.Suite")
	
	// Add fields from the test suite
	for _, field := range testSuite.Fields {
		injectTag := g.generateTestSuiteFieldInjectTag(field)
		g.writeLine(fmt.Sprintf("%-11s %s %s", field.Name, field.Type, injectTag))
	}
	
	g.unindent()
	g.writeLine("}")
	g.writeLine("")

	// Generate lifecycle methods
	g.generateTestSuiteLifecycleMethods(testSuite)

	// Generate test methods
	for _, method := range testSuite.Methods {
		if err := g.generateTestMethod(testSuite, method); err != nil {
			return err
		}
		g.writeLine("")
	}

	// Generate test runner
	g.generateTestSuiteRunner(testSuite)

	return nil
}

// generateTestSuiteLifecycleMethods generates lifecycle methods for test suite
func (g *CodeGenerator) generateTestSuiteLifecycleMethods(testSuite *TestSuiteDeclaration) {
	// SetupSuite - runs once before all tests
	g.writeLine(fmt.Sprintf("func (suite *%s) SetupSuite() {", testSuite.Name))
	g.indent()
	g.writeLine("// TODO: Add suite setup logic")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")

	// SetupTest - runs before each test
	g.writeLine(fmt.Sprintf("func (suite *%s) SetupTest() {", testSuite.Name))
	g.indent()
	g.writeLine("// TODO: Add test setup logic")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")

	// TearDownTest - runs after each test
	g.writeLine(fmt.Sprintf("func (suite *%s) TearDownTest() {", testSuite.Name))
	g.indent()
	g.writeLine("// TODO: Add test teardown logic")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")

	// TearDownSuite - runs once after all tests
	g.writeLine(fmt.Sprintf("func (suite *%s) TearDownSuite() {", testSuite.Name))
	g.indent()
	g.writeLine("// TODO: Add suite teardown logic")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
}

// generateTestMethod generates a test method
func (g *CodeGenerator) generateTestMethod(testSuite *TestSuiteDeclaration, method *MethodNode) error {
	// Check for @Test decorator to determine method naming
	testDecorator := g.findDecorator(method.Decorators, "Test")
	var methodName string
	var testDescription string
	
	if testDecorator != nil {
		// This is a test method - use Test prefix
		methodName = "Test" + strings.Title(method.Name[4:]) // Remove 'test' prefix and add 'Test'
		if len(methodName) <= 4 { // Handle edge cases
			methodName = "Test" + strings.Title(method.Name)
		}
		
		// Get test description from decorator argument
		if len(testDecorator.Args) > 0 {
			if desc, ok := testDecorator.Args[0].Value.(string); ok {
				testDescription = desc
			}
		}
	} else {
		// Non-test method (setup, teardown) - keep original name
		methodName = method.Name
	}
	
	g.writeLine(fmt.Sprintf("func (suite *%s) %s() {", testSuite.Name, methodName))
	g.indent()

	// Add test description comment if available
	if testDescription != "" {
		g.writeLine(fmt.Sprintf("// %s", testDescription))
	}

	// Generate method body placeholder
	g.writeLine("// TODO: Implement test logic")
	g.writeLine("suite.T().Skip(\"Test not implemented\")")

	g.unindent()
	g.writeLine("}")

	return nil
}

// generateTestSuiteRunner generates test runner function
func (g *CodeGenerator) generateTestSuiteRunner(testSuite *TestSuiteDeclaration) {
	g.writeLine(fmt.Sprintf("func Test%s(t *testing.T) {", testSuite.Name))
	g.indent()
	g.writeLine(fmt.Sprintf("suite.Run(t, new(%s))", testSuite.Name))
	g.unindent()
	g.writeLine("}")
}

// generateTestModuleDeclaration generates Go code for a test module
func (g *CodeGenerator) generateTestModuleDeclaration(testModule *TestModuleDeclaration) error {
	// Generate test module struct
	g.writeLine(fmt.Sprintf("type %s struct {", testModule.Name))
	g.indent()
	
	// Always add container field
	g.writeLine("container *core.DIContainer")
	
	// Add fields with inject tags
	for _, field := range testModule.Fields {
		injectTag := g.generateTestModuleFieldInjectTag(field)
		g.writeLine(fmt.Sprintf("%-8s %s %s", field.Name, field.Type, injectTag))
	}
	
	g.unindent()
	g.writeLine("}")
	g.writeLine("")

	// Generate constructor
	g.generateTestModuleConstructor(testModule)
	g.writeLine("")

	// Generate setup method
	g.generateTestModuleSetup(testModule)
	g.writeLine("")

	// Generate teardown method
	g.generateTestModuleTeardown(testModule)

	return nil
}

// generateTestModuleConstructor generates constructor for test module
func (g *CodeGenerator) generateTestModuleConstructor(testModule *TestModuleDeclaration) {
	g.writeLine(fmt.Sprintf("func New%s() *%s {", testModule.Name, testModule.Name))
	g.indent()
	g.writeLine(fmt.Sprintf("return &%s{", testModule.Name))
	g.indent()
	g.writeLine("container: core.NewDIContainer(),")
	g.unindent()
	g.writeLine("}")
	g.unindent()
	g.writeLine("}")
}

// generateTestModuleSetup generates setup method for test module
func (g *CodeGenerator) generateTestModuleSetup(testModule *TestModuleDeclaration) {
	g.writeLine(fmt.Sprintf("func (tm *%s) SetupTest(t *testing.T) {", testModule.Name))
	g.indent()
	
	// Add provider registration logic for each provider
	for _, provider := range testModule.Providers {
		providerName := g.convertToLowercase(provider)
		g.writeLine(fmt.Sprintf("tm.container.RegisterProvider(\"%s\", func() interface{} {", providerName))
		g.indent()
		
		// Check if provider is a Mock (starts with "Mock")
		if strings.HasPrefix(provider, "Mock") {
			g.writeLine(fmt.Sprintf("return New%s(t)", provider))
		} else {
			g.writeLine(fmt.Sprintf("return &%s{}", provider))
		}
		
		g.unindent()
		g.writeLine("})")
	}
	
	if len(testModule.Providers) > 0 {
		g.writeLine("")
	}
	
	g.writeLine("// TODO: Add additional test setup logic")
	
	g.unindent()
	g.writeLine("}")
}

// generateTestModuleTeardown generates teardown method for test module
func (g *CodeGenerator) generateTestModuleTeardown(testModule *TestModuleDeclaration) {
	g.writeLine(fmt.Sprintf("func (tm *%s) TeardownTest() {", testModule.Name))
	g.indent()
	g.writeLine("// TODO: Add test module teardown logic")
	g.unindent()
	g.writeLine("}")
}

// generateTestModuleFieldInjectTag generates inject tag for test module field
func (g *CodeGenerator) generateTestModuleFieldInjectTag(field *FieldNode) string {
	// Look for @Inject decorator on the field
	for _, decorator := range field.Decorators {
		if decorator.Name == "Inject" {
			if len(decorator.Args) > 0 {
				// Use the provided inject name
				if strVal, ok := decorator.Args[0].Value.(string); ok {
					return fmt.Sprintf("`inject:\"%s\"`", strVal)
				}
			}
			// Default to field name if no argument provided
			return fmt.Sprintf("`inject:\"%s\"`", field.Name)
		}
	}
	
	// Field without @Inject decorator gets empty inject tag
	return "`inject:\"\"`"
}

// convertToLowercase converts provider name to lowercase for registration
func (g *CodeGenerator) convertToLowercase(name string) string {
	if len(name) == 0 {
		return name
	}
	return strings.ToLower(name[:1]) + strings.ToLower(name[1:])
}

// generateTestSuiteFieldInjectTag generates inject tag for test suite field
func (g *CodeGenerator) generateTestSuiteFieldInjectTag(field *FieldNode) string {
	// Look for @Inject decorator on the field
	for _, decorator := range field.Decorators {
		if decorator.Name == "Inject" {
			if len(decorator.Args) > 0 {
				// Use the provided inject name
				if strVal, ok := decorator.Args[0].Value.(string); ok {
					return fmt.Sprintf("`inject:\"%s\"`", strVal)
				}
			}
			// Default to field name if no argument provided
			return fmt.Sprintf("`inject:\"%s\"`", field.Name)
		}
	}
	
	// Field without @Inject decorator - no tag for test suite fields
	return ""
}