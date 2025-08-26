package codegen

import "fmt"

// generateFactoryDeclaration generates Go code for a factory
func (g *CodeGenerator) generateFactoryDeclaration(factory *FactoryDeclaration) error {
	// Generate factory struct
	g.writeLine(fmt.Sprintf("type %s struct {", factory.Name))
	g.indent()
	
	// Add built-in factory fields
	g.writeLine("sequenceCounters map[string]int")
	g.writeLine("rand             *rand.Rand")
	
	// Add user-defined fields with proper inject tags
	for _, field := range factory.Fields {
		injectTag := g.generateFactoryFieldInjectTag(field)
		g.writeLine(fmt.Sprintf("%-16s %s %s", field.Name, field.Type, injectTag))
	}
	
	g.unindent()
	g.writeLine("}")
	g.writeLine("")

	// Generate constructor
	g.generateFactoryConstructor(factory)
	g.writeLine("")

	// Generate build method
	g.generateFactoryBuildMethod(factory)
	g.writeLine("")

	// Generate helper methods
	g.generateFactoryHelperMethods(factory)

	// Generate trait methods
	for _, method := range factory.Methods {
		if g.hasTraitDecorator(method) {
			if err := g.generateFactoryTraitMethod(factory, method); err != nil {
				return err
			}
			g.writeLine("")
		}
	}

	return nil
}

// generateFactoryConstructor generates factory constructor
func (g *CodeGenerator) generateFactoryConstructor(factory *FactoryDeclaration) {
	g.writeLine(fmt.Sprintf("func New%s() *%s {", factory.Name, factory.Name))
	g.indent()
	g.writeLine("return &" + factory.Name + "{")
	g.indent()
	g.writeLine("sequenceCounters: make(map[string]int),")
	g.writeLine("rand: rand.New(rand.NewSource(time.Now().UnixNano())),")
	g.unindent()
	g.writeLine("}")
	g.unindent()
	g.writeLine("}")
}

// generateFactoryBuildMethod generates factory build method
func (g *CodeGenerator) generateFactoryBuildMethod(factory *FactoryDeclaration) {
	g.writeLine(fmt.Sprintf("func (f *%s) Build(overrides interface{}) *%s {", factory.Name, factory.TargetType))
	g.indent()
	g.writeLine(fmt.Sprintf("instance := &%s{", factory.TargetType))
	g.indent()
	g.writeLine("// TODO: Add default field values")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")
	g.writeLine("// TODO: Apply overrides if provided")
	g.writeLine("")
	g.writeLine("return instance")
	g.unindent()
	g.writeLine("}")
}

// generateFactoryHelperMethods generates helper methods for factory
func (g *CodeGenerator) generateFactoryHelperMethods(factory *FactoryDeclaration) {
	// Generate getSequence method
	g.writeLine(fmt.Sprintf("func (f *%s) getSequence(name string) int {", factory.Name))
	g.indent()
	g.writeLine("f.sequenceCounters[name]++")
	g.writeLine("return f.sequenceCounters[name]")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")

	// Generate random string method
	g.writeLine(fmt.Sprintf("func (f *%s) generateRandomString() string {", factory.Name))
	g.indent()
	g.writeLine("const charset = \"abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789\"")
	g.writeLine("length := 10")
	g.writeLine("result := make([]byte, length)")
	g.writeLine("for i := range result {")
	g.indent()
	g.writeLine("result[i] = charset[f.rand.Intn(len(charset))]")
	g.unindent()
	g.writeLine("}")
	g.writeLine("return string(result)")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")

	// Generate random int method
	g.writeLine(fmt.Sprintf("func (f *%s) generateRandomInt(min, max int) int {", factory.Name))
	g.indent()
	g.writeLine("return f.rand.Intn(max-min) + min")
	g.unindent()
	g.writeLine("}")
}

// hasTraitDecorator checks if a method has a @Trait decorator
func (g *CodeGenerator) hasTraitDecorator(method *MethodNode) bool {
	return g.hasDecorator(method.Decorators, "Trait")
}

// generateFactoryTraitMethod generates a trait method for the factory
func (g *CodeGenerator) generateFactoryTraitMethod(factory *FactoryDeclaration, method *MethodNode) error {
	traitDecorator := g.findDecorator(method.Decorators, "Trait")
	if traitDecorator == nil {
		return fmt.Errorf("trait method %s missing @Trait decorator", method.Name)
	}

	// Get trait name from decorator argument
	traitName := method.Name
	if len(traitDecorator.Args) > 0 {
		if strVal, ok := traitDecorator.Args[0].Value.(string); ok {
			traitName = strVal
		}
	}

	// Generate trait method signature
	g.writeLine(fmt.Sprintf("func (f *%s) %s(instance *%s) *%s {",
		factory.Name, method.Name, factory.TargetType, factory.TargetType))
	g.indent()

	// Generate method body
	g.writeLine(fmt.Sprintf("// Trait: %s", traitName))
	g.writeLine("// TODO: Add trait-specific modifications here")
	g.writeLine("return instance")

	g.unindent()
	g.writeLine("}")

	return nil
}

// generateMockDeclaration generates Go code for a mock
func (g *CodeGenerator) generateMockDeclaration(mock *MockDeclaration) error {
	// Generate mock struct
	g.writeLine(fmt.Sprintf("type %s struct {", mock.Name))
	g.indent()
	g.writeLine("CallLog      []MockCall")
	g.writeLine("expectations []MockExpectation")
	g.writeLine("t            *testing.T")
	
	// Add mock fields with inject tags
	for _, field := range mock.Fields {
		injectTag := g.generateFactoryFieldInjectTag(field)
		g.writeLine(fmt.Sprintf("%-13s %s %s", field.Name, field.Type, injectTag))
	}
	
	g.unindent()
	g.writeLine("}")
	g.writeLine("")

	// Generate support structures
	g.generateMockSupportStructures(mock)
	g.writeLine("")

	// Generate constructor
	g.generateMockConstructor(mock)
	g.writeLine("")

	// Generate expectation methods
	g.generateMockExpectationMethods(mock)
	g.writeLine("")

	// Generate mock methods
	for _, method := range mock.Methods {
		if err := g.generateMockMethod(mock, method); err != nil {
			return err
		}
		g.writeLine("")
	}

	return nil
}

// generateMockSupportStructures generates support structures for mock tracking
func (g *CodeGenerator) generateMockSupportStructures(mock *MockDeclaration) {
	// Generate MockCall structure
	g.writeLine("// MockCall represents a method call made to the mock")
	g.writeLine("type MockCall struct {")
	g.indent()
	g.writeLine("Method string")
	g.writeLine("Args   []interface{}")
	g.writeLine("Result []interface{}")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")

	// Generate MockExpectation structure
	g.writeLine("// MockExpectation represents an expected method call")
	g.writeLine("type MockExpectation struct {")
	g.indent()
	g.writeLine("Method   string")
	g.writeLine("Args     []interface{}")
	g.writeLine("Returns  []interface{}")
	g.writeLine("Error    error")
	g.writeLine("Called   bool")
	g.unindent()
	g.writeLine("}")
}

// generateMockConstructor generates the mock constructor
func (g *CodeGenerator) generateMockConstructor(mock *MockDeclaration) {
	g.writeLine(fmt.Sprintf("func New%s(t *testing.T) *%s {", mock.Name, mock.Name))
	g.indent()
	g.writeLine("return &" + mock.Name + "{")
	g.indent()
	g.writeLine("CallLog: make([]MockCall, 0),")
	g.writeLine("expectations: make([]MockExpectation, 0),")
	g.writeLine("t: t,")
	g.unindent()
	g.writeLine("}")
	g.unindent()
	g.writeLine("}")
}

// generateMockExpectationMethods generates expectation setup methods
func (g *CodeGenerator) generateMockExpectationMethods(mock *MockDeclaration) {
	// Generate On method for setting expectations
	g.writeLine(fmt.Sprintf("func (m *%s) On(method string, args ...interface{}) *MockExpectation {", mock.Name))
	g.indent()
	g.writeLine("expectation := MockExpectation{")
	g.indent()
	g.writeLine("Method: method,")
	g.writeLine("Args:   args,")
	g.unindent()
	g.writeLine("}")
	g.writeLine("m.expectations = append(m.expectations, expectation)")
	g.writeLine("return &m.expectations[len(m.expectations)-1]")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")

	// Generate Return method for setting return values
	g.writeLine("func (e *MockExpectation) Return(values ...interface{}) *MockExpectation {")
	g.indent()
	g.writeLine("e.Returns = values")
	g.writeLine("return e")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")

	// Generate verification methods
	g.writeLine(fmt.Sprintf("func (m *%s) AssertExpectations(t *testing.T) {", mock.Name))
	g.indent()
	g.writeLine("for _, expectation := range m.expectations {")
	g.indent()
	g.writeLine("if !expectation.Called {")
	g.indent()
	g.writeLine("t.Errorf(\"Expected method %s was not called\", expectation.Method)")
	g.unindent()
	g.writeLine("}")
	g.unindent()
	g.writeLine("}")
	g.unindent()
	g.writeLine("}")
}

// generateMockMethod generates a mock method implementation
func (g *CodeGenerator) generateMockMethod(mock *MockDeclaration, method *MethodNode) error {
	g.writeLine(fmt.Sprintf("func (m *%s) %s() {", mock.Name, method.Name))
	g.indent()

	// Generate call logging
	g.writeLine("call := MockCall{")
	g.indent()
	g.writeLine(fmt.Sprintf("Method: \"%s\",", method.Name))
	g.writeLine("Args:   []interface{}{},")
	g.unindent()
	g.writeLine("}")
	g.writeLine("m.CallLog = append(m.CallLog, call)")
	g.writeLine("")

	// Generate expectation matching
	g.writeLine("// Find matching expectation")
	g.writeLine("for i := range m.expectations {")
	g.indent()
	g.writeLine(fmt.Sprintf("if m.expectations[i].Method == \"%s\" {", method.Name))
	g.indent()
	g.writeLine("m.expectations[i].Called = true")
	g.writeLine("// TODO: Return expected values")
	g.writeLine("return")
	g.unindent()
	g.writeLine("}")
	g.unindent()
	g.writeLine("}")
	g.writeLine("")

	g.writeLine("// Method was called but not expected")
	g.writeLine(fmt.Sprintf("m.t.Errorf(\"Unexpected call to method %s\")", method.Name))

	g.unindent()
	g.writeLine("}")

	return nil
}

// generateFactoryFieldInjectTag generates inject tag for factory field based on decorators
func (g *CodeGenerator) generateFactoryFieldInjectTag(field *FieldNode) string {
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