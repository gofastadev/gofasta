package transpiler

import (
	"strings"
	"testing"
)

// TestTestSuiteDecorator tests the @TestSuite() decorator parsing and generation
func TestTestSuiteDecorator(t *testing.T) {
	input := `package main

@TestSuite()
type UserServiceTests struct {
	@Inject()
	userService *UserService
	
	@Inject("mockDB")
	mockDB      *MockDatabase
}

@BeforeEach()
func setup() {
	// Setup logic
}

@Test("should create user successfully") 
func testCreateUser() {
	// Test logic
}

@AfterEach()
func cleanup() {
	// Cleanup logic
}`

	file, err := ParseGofaFile(input)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	// Verify parsing
	if len(file.Declarations) != 1 {
		t.Fatalf("Expected 1 declaration, got %d", len(file.Declarations))
	}

	testSuite, ok := file.Declarations[0].(*TestSuiteDeclaration)
	if !ok {
		t.Fatalf("Expected TestSuiteDeclaration, got %T", file.Declarations[0])
	}

	if testSuite.Name != "UserServiceTests" {
		t.Errorf("Expected test suite name 'UserServiceTests', got '%s'", testSuite.Name)
	}

	if len(testSuite.Decorators) != 1 {
		t.Fatalf("Expected 1 decorator, got %d", len(testSuite.Decorators))
	}

	if testSuite.Decorators[0].Name != "TestSuite" {
		t.Errorf("Expected decorator 'TestSuite', got '%s'", testSuite.Decorators[0].Name)
	}

	// Check fields
	if len(testSuite.Fields) != 2 {
		t.Fatalf("Expected 2 fields, got %d", len(testSuite.Fields))
	}

	// Check methods
	if len(testSuite.Methods) != 3 {
		t.Logf("Test suite: %+v", testSuite)
		for i, method := range testSuite.Methods {
			t.Logf("Method %d: %s with %d decorators", i, method.Name, len(method.Decorators))
		}
		t.Fatalf("Expected 3 methods, got %d", len(testSuite.Methods))
	}

	// Verify method decorators
	setupMethod := testSuite.Methods[0]
	if len(setupMethod.Decorators) != 1 || setupMethod.Decorators[0].Name != "BeforeEach" {
		t.Errorf("Expected BeforeEach decorator on setup method")
	}

	testMethod := testSuite.Methods[1]
	if len(testMethod.Decorators) != 1 || testMethod.Decorators[0].Name != "Test" {
		t.Errorf("Expected Test decorator on test method")
	}

	// Verify test description
	if len(testMethod.Decorators[0].Args) != 1 {
		t.Errorf("Expected 1 argument on Test decorator")
	} else if testMethod.Decorators[0].Args[0].Value != "should create user successfully" {
		t.Errorf("Expected test description 'should create user successfully', got '%v'", testMethod.Decorators[0].Args[0].Value)
	}

	cleanupMethod := testSuite.Methods[2]
	if len(cleanupMethod.Decorators) != 1 || cleanupMethod.Decorators[0].Name != "AfterEach" {
		t.Errorf("Expected AfterEach decorator on cleanup method")
	}
}

// TestTestSuiteCodeGeneration tests the code generation for @TestSuite() decorator
func TestTestSuiteCodeGeneration(t *testing.T) {
	input := `package main

@TestSuite()
type UserServiceTests struct {
	@Inject()
	userService *UserService
	
	@Inject("mockDB")
	mockDB      *MockDatabase
}

@BeforeEach()
func setup() {
	// Setup logic
}

@Test("should create user successfully") 
func testCreateUser() {
	// Test logic
}

@AfterEach()
func cleanup() {
	// Cleanup logic
}`

	file, err := ParseGofaFile(input)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	generator := NewCodeGenerator("main")
	generatedCode, err := generator.GenerateGoCode(file)
	if err != nil {
		t.Fatalf("Failed to generate code: %v", err)
	}

	// Debug: print generated code
	t.Logf("Generated code:\n%s", generatedCode)

	// Verify generated code contains expected elements
	expectedElements := []string{
		"type UserServiceTests struct {",
		"suite.Suite",
		"userService *UserService",
		"mockDB",
		"*MockDatabase",
		"func (suite *UserServiceTests) SetupSuite() {",
		"func (suite *UserServiceTests) SetupTest() {",
		"func (suite *UserServiceTests) TearDownTest() {",
		"func (suite *UserServiceTests) TearDownSuite() {",
		"func (suite *UserServiceTests) TestCreateUser() {",
		"// should create user successfully",
		"func TestUserServiceTests(t *testing.T) {",
		"suite.Run(t, new(UserServiceTests))",
	}

	for _, expected := range expectedElements {
		if !strings.Contains(generatedCode, expected) {
			t.Errorf("Generated code missing expected element: %s", expected)
		}
	}

	// Verify imports are added
	expectedImports := []string{
		`"testing"`,
		`"github.com/stretchr/testify/assert"`,
		`"github.com/stretchr/testify/suite"`,
	}

	for _, expectedImport := range expectedImports {
		if !strings.Contains(generatedCode, expectedImport) {
			t.Errorf("Generated code missing expected import: %s", expectedImport)
		}
	}
}

// TestTestSuiteWithHTTPTest tests @TestSuite() combined with @HTTPTest()
func TestTestSuiteWithHTTPTest(t *testing.T) {
	input := `package main

@TestSuite()
@HTTPTest()
type APITests struct {
	@Inject("httpClient")
	client *TestClient
}

@Test("should return user data")
func testGetUser() {
	// HTTP test logic
}`

	file, err := ParseGofaFile(input)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	testSuite := file.Declarations[0].(*TestSuiteDeclaration)
	
	// Check that both decorators are present
	if len(testSuite.Decorators) != 2 {
		t.Fatalf("Expected 2 decorators, got %d", len(testSuite.Decorators))
	}

	decoratorNames := make(map[string]bool)
	for _, decorator := range testSuite.Decorators {
		decoratorNames[decorator.Name] = true
	}

	if !decoratorNames["TestSuite"] {
		t.Error("Missing TestSuite decorator")
	}

	if !decoratorNames["HTTPTest"] {
		t.Error("Missing HTTPTest decorator")
	}
}

// TestTestSuiteWithDatabaseTest tests @TestSuite() combined with @DatabaseTest()
func TestTestSuiteWithDatabaseTest(t *testing.T) {
	input := `package main

@TestSuite()
@DatabaseTest({migrations: "./testdata/migrations"})
type IntegrationTests struct {
	@Inject("database")
	db *Database
}

@BeforeAll()
func setupDatabase() {
	// Database setup
}

@Test("should save entity")
func testSaveEntity() {
	// Database test logic
}

@AfterAll()
func cleanupDatabase() {
	// Database cleanup
}`

	file, err := ParseGofaFile(input)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	testSuite := file.Declarations[0].(*TestSuiteDeclaration)
	
	// Check decorators
	if len(testSuite.Decorators) != 2 {
		t.Fatalf("Expected 2 decorators, got %d", len(testSuite.Decorators))
	}

	// Check DatabaseTest decorator arguments
	var dbTestDecorator *DecoratorNode
	for _, decorator := range testSuite.Decorators {
		if decorator.Name == "DatabaseTest" {
			dbTestDecorator = decorator
			break
		}
	}

	if dbTestDecorator == nil {
		t.Fatal("Missing DatabaseTest decorator")
	}

	if len(dbTestDecorator.Args) != 1 {
		t.Fatalf("Expected 1 argument on DatabaseTest decorator, got %d", len(dbTestDecorator.Args))
	}

	// The argument is parsed as an object: {migrations: "./testdata/migrations"}
	if argValue, ok := dbTestDecorator.Args[0].Value.(map[string]any); ok {
		if migrations, exists := argValue["migrations"]; exists {
			if migrations != "./testdata/migrations" {
				t.Errorf("Expected migrations path './testdata/migrations', got '%v'", migrations)
			}
		} else {
			t.Error("Missing 'migrations' key in DatabaseTest argument")
		}
	} else {
		t.Errorf("Expected DatabaseTest argument to be an object, got %T: %v", dbTestDecorator.Args[0].Value, dbTestDecorator.Args[0].Value)
	}

	// Check BeforeAll and AfterAll methods
	var hasBeforeAll, hasAfterAll bool
	for _, method := range testSuite.Methods {
		for _, decorator := range method.Decorators {
			if decorator.Name == "BeforeAll" {
				hasBeforeAll = true
			}
			if decorator.Name == "AfterAll" {
				hasAfterAll = true
			}
		}
	}

	if !hasBeforeAll {
		t.Error("Missing BeforeAll decorator")
	}

	if !hasAfterAll {
		t.Error("Missing AfterAll decorator")
	}
}

// TestTestSuiteCodeGenerationWithLifecycleMethods tests lifecycle method generation
func TestTestSuiteCodeGenerationWithLifecycleMethods(t *testing.T) {
	input := `package main

@TestSuite()
type UserServiceTests struct {}

@BeforeAll()
func initializeResources() {}

@BeforeEach()
func setup() {}

@Test("sample test")
func testSomething() {}

@AfterEach()
func cleanup() {}

@AfterAll()
func finalizeResources() {}`

	file, err := ParseGofaFile(input)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	generator := NewCodeGenerator("main")
	generatedCode, err := generator.GenerateGoCode(file)
	if err != nil {
		t.Fatalf("Failed to generate code: %v", err)
	}

	// Verify that lifecycle methods call the decorated methods
	expectedCalls := []string{
		"initializeResources()", // BeforeAll
		"setup()",               // BeforeEach
		"cleanup()",             // AfterEach  
		"finalizeResources()",   // AfterAll
	}

	for _, expectedCall := range expectedCalls {
		if !strings.Contains(generatedCode, expectedCall) {
			t.Errorf("Generated code missing expected method call: %s", expectedCall)
		}
	}

	// Verify that the actual test method is generated but lifecycle methods are not duplicated as tests
	if !strings.Contains(generatedCode, "func (suite *UserServiceTests) TestSomething() {") {
		t.Error("Missing generated test method TestSomething")
	}

	// Lifecycle methods should not be generated as test methods
	unwantedMethods := []string{
		"func (suite *UserServiceTests) TestInitializeResources() {",
		"func (suite *UserServiceTests) TestSetup() {",
		"func (suite *UserServiceTests) TestCleanup() {",
		"func (suite *UserServiceTests) TestFinalizeResources() {",
	}

	for _, unwanted := range unwantedMethods {
		if strings.Contains(generatedCode, unwanted) {
			t.Errorf("Generated code contains unwanted lifecycle method as test: %s", unwanted)
		}
	}
}

// TestMultipleTestSuites tests multiple test suites in the same file
func TestMultipleTestSuites(t *testing.T) {
	input := `package main

@TestSuite()
type UserServiceTests struct {
	@Inject()
	userService *UserService
}

@Test("should create user")
func testCreateUser() {}

@TestSuite()
type ProductServiceTests struct {
	@Inject()
	productService *ProductService
}

@Test("should create product")
func testCreateProduct() {}`

	file, err := ParseGofaFile(input)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	if len(file.Declarations) != 2 {
		t.Fatalf("Expected 2 declarations, got %d", len(file.Declarations))
	}

	// Verify first test suite
	testSuite1, ok := file.Declarations[0].(*TestSuiteDeclaration)
	if !ok || testSuite1.Name != "UserServiceTests" {
		t.Error("First declaration should be UserServiceTests")
	}

	// Verify second test suite
	testSuite2, ok := file.Declarations[1].(*TestSuiteDeclaration)
	if !ok || testSuite2.Name != "ProductServiceTests" {
		t.Error("Second declaration should be ProductServiceTests")
	}

	// Test code generation for multiple test suites
	generator := NewCodeGenerator("main")
	generatedCode, err := generator.GenerateGoCode(file)
	if err != nil {
		t.Fatalf("Failed to generate code: %v", err)
	}

	// Should generate test runners for both
	expectedRunners := []string{
		"func TestUserServiceTests(t *testing.T) {",
		"func TestProductServiceTests(t *testing.T) {",
	}

	for _, expectedRunner := range expectedRunners {
		if !strings.Contains(generatedCode, expectedRunner) {
			t.Errorf("Generated code missing expected runner: %s", expectedRunner)
		}
	}
}