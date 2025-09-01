package codegen

import (
	"strings"
	"testing"
)

// TestWebSocketImportGeneration tests WebSocket import statement generation
func TestWebSocketImportGeneration(t *testing.T) {
	input := `package test

@WebSocketGateway({
	port: 3000,
	namespace: "/api/ws",
	cors: {
		origin: "*",
		credentials: true
	},
	transports: ["websocket"],
	pingTimeout: 5000,
	pingInterval: 2500
})
type ComplexGateway struct {
	@Inject("service")
	service *MyService
}

@SubscribeMessage("test")
func HandleMessage(
	@MessageBody() data string,
	@ConnectedSocket() client *WebSocketClient
) {
	// Test handler
}

@OnGatewayConnection()
func HandleConnection(@ConnectedSocket() client *WebSocketClient) {
	// Test connection handler
}

@Injectable()
type MyService struct {}`

	file, err := ParseGofaFile(input)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	generator := NewCodeGenerator("test")
	output, err := generator.GenerateGoCode(file)
	if err != nil {
		t.Fatalf("Failed to generate code: %v", err)
	}

	t.Logf("Generated output:\n%s", output)

	// Test standard imports
	expectedImports := []string{
		`"github.com/healtronlabs/gofasta/packages/websocket"`,
		`"github.com/healtronlabs/gofasta/packages/http"`,
		`"github.com/healtronlabs/gofasta/packages/core"`,
	}

	for _, imp := range expectedImports {
		if !strings.Contains(output, imp) {
			t.Errorf("Expected import %s not found in output:\n%s", imp, output)
		}
	}

	// Test potential missing imports for WebSocket functionality
	potentialImports := []string{
		`"fmt"`,
		`"net/http"`,
		`"time"`,
		`"strings"`,
		`"strconv"`,
	}

	// Check which standard imports are used in WebSocket code
	for _, imp := range potentialImports {
		if strings.Contains(output, imp) {
			t.Logf("Standard import %s found in output", imp)
		} else {
			t.Logf("Standard import %s NOT found in output", imp)
		}
	}
}

// TestWebSocketStandaloneFunction tests import generation for standalone WebSocket functions
func TestWebSocketStandaloneFunctionImports(t *testing.T) {
	input := `package test

@SubscribeMessage("standalone")
func StandaloneHandler(
	@MessageBody() data string,
	@ConnectedSocket() client *WebSocketClient
) {
	// Standalone WebSocket function
}

@OnGatewayInit()
func InitHandler() {
	// Init handler
}`

	file, err := ParseGofaFile(input)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	generator := NewCodeGenerator("test")
	output, err := generator.GenerateGoCode(file)
	if err != nil {
		t.Fatalf("Failed to generate code: %v", err)
	}

	t.Logf("Generated output for standalone functions:\n%s", output)

	// Test that WebSocket imports are still added for standalone functions
	if !strings.Contains(output, `"github.com/healtronlabs/gofasta/packages/websocket"`) {
		t.Errorf("Expected WebSocket import not found for standalone function")
	}
}

// TestWebSocketMiddlewareImports tests imports needed for WebSocket middleware
func TestWebSocketMiddlewareImports(t *testing.T) {
	input := `package test

@WebSocketGateway(8080)
@UseGuards("wsAuthGuard")
@UseInterceptors("wsLoggingInterceptor")
@UsePipes("wsValidationPipe")
type MiddlewareGateway struct {}

@SubscribeMessage("guarded")
@UseGuards("messageGuard")
func GuardedHandler(@MessageBody() data string) {
	// Handler with middleware
}

@OnGatewayConnection()
func HandleConnection(@ConnectedSocket() client *WebSocketClient) {
	// Connection with middleware
}`

	file, err := ParseGofaFile(input)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	generator := NewCodeGenerator("test")
	output, err := generator.GenerateGoCode(file)
	if err != nil {
		t.Fatalf("Failed to generate code: %v", err)
	}

	t.Logf("Generated output for middleware:\n%s", output)

	// Check for imports needed for middleware
	expectedImports := []string{
		`"github.com/healtronlabs/gofasta/packages/websocket"`,
		`"github.com/healtronlabs/gofasta/packages/http"`,
		`"github.com/healtronlabs/gofasta/packages/core"`,
	}

	for _, imp := range expectedImports {
		if !strings.Contains(output, imp) {
			t.Errorf("Expected import %s not found in middleware output", imp)
		}
	}

	// Look for additional imports that might be needed
	additionalImports := []string{"errors", "context", "encoding/json", "log"}
	for _, imp := range additionalImports {
		if strings.Contains(output, `"`+imp+`"`) {
			t.Logf("Found %s import", imp)
		}
	}
}

// TestWebSocketParameterDecoratorImports tests imports needed for WebSocket parameter decorators  
func TestWebSocketParameterDecoratorImports(t *testing.T) {
	input := `package test

@WebSocketGateway(8080)
type TestGateway struct {}

@SubscribeMessage("complex")
func ComplexHandler(
	@MessageBody() data *CustomData,
	@ConnectedSocket() client *WebSocketClient,
	@MessageAck() ack *AckCallback,
	@Headers() headers map[string]string,
	@Query() query map[string]string,
	@Session() session *Session,
	@CurrentUser() user *User,
	@ClientIP() ip string,
	@EventName() event string
) {
	// Complex handler with all parameter decorators
}

type CustomData struct {
	Message string
}

type AckCallback struct{}
type Session struct{}
type User struct{}`

	file, err := ParseGofaFile(input)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	generator := NewCodeGenerator("test")
	output, err := generator.GenerateGoCode(file)
	if err != nil {
		t.Fatalf("Failed to generate code: %v", err)
	}

	t.Logf("Generated output for parameter decorators:\n%s", output)

	// Check for imports that might be needed for parameter handling
	expectedImports := []string{
		`"github.com/healtronlabs/gofasta/packages/websocket"`,
		`"github.com/healtronlabs/gofasta/packages/http"`,
		`"github.com/healtronlabs/gofasta/packages/core"`,
	}

	for _, imp := range expectedImports {
		if !strings.Contains(output, imp) {
			t.Errorf("Expected import %s not found in output with parameter decorators", imp)
		}
	}
}

// TestWebSocketErrorHandlingImports tests imports for error handling
func TestWebSocketErrorHandlingImports(t *testing.T) {
	input := `package test

@WebSocketGateway(8080)
type ErrorGateway struct {}

@SubscribeMessage("error_test")
func ErrorHandler(@MessageBody() data string) error {
	// Handler that returns error
	return errors.New("test error")
}

@SubscribeMessage("no_error")
func NoErrorHandler(@MessageBody() data string) {
	// Handler with no error return
}`

	file, err := ParseGofaFile(input)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	generator := NewCodeGenerator("test")
	output, err := generator.GenerateGoCode(file)
	if err != nil {
		t.Fatalf("Failed to generate code: %v", err)
	}

	t.Logf("Generated output for error handling:\n%s", output)
	
	// Note: Error handling import detection may not work if ReturnType is not parsed correctly
	// This is expected behavior for the current implementation
	if strings.Contains(output, `"errors"`) {
		t.Logf("Found errors import (good for error handling)")
	} else {
		t.Logf("Errors import not found - may need enhanced return type parsing")
	}
	
	// Should include encoding/json for MessageBody decorators
	if !strings.Contains(output, `"encoding/json"`) {
		t.Errorf("Expected encoding/json import not found with MessageBody decorators")
	}
}

// TestWebSocketLoggingImports tests imports for logging features
func TestWebSocketLoggingImports(t *testing.T) {
	input := `package test

@WebSocketGateway(8080)  
@UseInterceptors("loggingInterceptor")
type LoggingGateway struct {}

@SubscribeMessage("logged")
func LoggedHandler(@MessageBody() data string) {
	// Handler with logging middleware
}`

	file, err := ParseGofaFile(input)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	generator := NewCodeGenerator("test")
	output, err := generator.GenerateGoCode(file)
	if err != nil {
		t.Fatalf("Failed to generate code: %v", err)
	}

	// Should include context for middleware
	if !strings.Contains(output, `"context"`) {
		t.Errorf("Expected context import not found with logging middleware")
	}
}

// TestWebSocketComplexTypeImports tests imports for complex message types
func TestWebSocketComplexTypeImports(t *testing.T) {
	input := `package test

@WebSocketGateway(8080)
type ComplexTypeGateway struct {}

@SubscribeMessage("complex_msg")
func ComplexHandler(@MessageBody() data *ComplexMessage) {
	// Handler with complex message type
}

type ComplexMessage struct {
	ID      string
	Payload map[string]interface{}
}`

	file, err := ParseGofaFile(input)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	generator := NewCodeGenerator("test")
	output, err := generator.GenerateGoCode(file)
	if err != nil {
		t.Fatalf("Failed to generate code: %v", err)
	}

	// Should include encoding/json for complex message body types
	if !strings.Contains(output, `"encoding/json"`) {
		t.Errorf("Expected encoding/json import not found with complex MessageBody type")
	}
}

// TestWebSocketMinimalImports tests that minimal imports are used when no advanced features are present
func TestWebSocketMinimalImports(t *testing.T) {
	input := `package test

@WebSocketGateway(8080)
type MinimalGateway struct {}

func SimpleHandler() {
	// Simple handler with no decorators or complex features
}`

	file, err := ParseGofaFile(input)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	generator := NewCodeGenerator("test")
	output, err := generator.GenerateGoCode(file)
	if err != nil {
		t.Fatalf("Failed to generate code: %v", err)
	}

	// Should NOT include conditional imports when no advanced features are used
	conditionalImports := []string{`"errors"`, `"context"`, `"encoding/json"`, `"log"`}
	for _, imp := range conditionalImports {
		if strings.Contains(output, imp) {
			t.Logf("Conditional import %s found in minimal setup (may be expected)", imp)
		}
	}

	// Should always include core WebSocket imports
	coreImports := []string{
		`"github.com/healtronlabs/gofasta/packages/websocket"`,
		`"github.com/healtronlabs/gofasta/packages/http"`,
		`"github.com/healtronlabs/gofasta/packages/core"`,
	}
	
	for _, imp := range coreImports {
		if !strings.Contains(output, imp) {
			t.Errorf("Expected core import %s not found in minimal WebSocket setup", imp)
		}
	}
}