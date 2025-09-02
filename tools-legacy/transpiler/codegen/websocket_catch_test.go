package codegen

import (
	"strings"
	"testing"
)

func TestWebSocketCatchDecoratorParsing(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string // Expected error types in catch decorators
	}{
		{
			name: "Single error type in WebSocket gateway method",
			input: `
package main

@WebSocketGateway(8080)
type ChatGateway struct {}

@Catch(BadRequestError)
@SubscribeMessage("send_message")
func HandleSendMessage(@ConnectedSocket() client *WebSocketClient) {}
`,
			expected: []string{"BadRequestError"},
		},
		{
			name: "Multiple error types in WebSocket method",
			input: `
package main

@WebSocketGateway(8080)
type ChatGateway struct {}

@Catch(BadRequestError, ValidationError)
@SubscribeMessage("send_message")
func HandleSendMessage(@ConnectedSocket() client *WebSocketClient) {}
`,
			expected: []string{"BadRequestError", "ValidationError"},
		},
		{
			name: "Global WebSocket gateway error handler",
			input: `
package main

@WebSocketGateway(8080)
@Catch()
type ChatGateway struct {}

@SubscribeMessage("send_message")
func HandleSendMessage(@ConnectedSocket() client *WebSocketClient) {}
`,
			expected: []string{}, // Empty means global handler
		},
		{
			name: "Gateway and method level WebSocket handlers",
			input: `
package main

@WebSocketGateway(8080)
@Catch(NotFoundError)
type ChatGateway struct {}

@Catch(ValidationError)
@SubscribeMessage("send_message")
func HandleSendMessage(@ConnectedSocket() client *WebSocketClient) {}
`,
			expected: []string{"NotFoundError", "ValidationError"},
		},
		{
			name: "WebSocket lifecycle handler with catch",
			input: `
package main

@WebSocketGateway(8080)
type ChatGateway struct {}

@Catch(AuthenticationError)
@OnGatewayConnection()
func HandleConnection(@ConnectedSocket() client *WebSocketClient) {}
`,
			expected: []string{"AuthenticationError"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			parser := NewParser(lexer)
			file, err := parser.ParseFile()

			if err != nil {
				t.Fatalf("Parsing failed: %v", err)
			}

			if len(file.Declarations) == 0 {
				t.Fatal("No declarations found")
			}

			// Collect all catch decorators from all declarations
			var foundErrorTypes []string
			errorTypeSet := make(map[string]bool) // To deduplicate error types


			// Check all declarations for @Catch decorators
			for _, decl := range file.Declarations {
				switch d := decl.(type) {
				case *WebSocketGatewayDeclaration:
					// Check gateway-level decorators
					for _, decorator := range d.Decorators {
						if decorator.Name == "Catch" {
							for _, arg := range decorator.Args {
								if errorType, ok := arg.Value.(string); ok {
									if !errorTypeSet[errorType] {
										errorTypeSet[errorType] = true
										foundErrorTypes = append(foundErrorTypes, errorType)
									}
								}
							}
						}
					}
				case *WebSocketFunctionDeclaration:
					// Check standalone function decorators
					for _, decorator := range d.Decorators {
						if decorator.Name == "Catch" {
							for _, arg := range decorator.Args {
								if errorType, ok := arg.Value.(string); ok {
									if !errorTypeSet[errorType] {
										errorTypeSet[errorType] = true
										foundErrorTypes = append(foundErrorTypes, errorType)
									}
								}
							}
						}
					}
				}
			}

			// Verify expected error types
			if len(foundErrorTypes) != len(tt.expected) {
				t.Errorf("Expected %d error types, got %d. Found: %v", len(tt.expected), len(foundErrorTypes), foundErrorTypes)
			}

			for i, expectedType := range tt.expected {
				if i >= len(foundErrorTypes) || foundErrorTypes[i] != expectedType {
					t.Errorf("Expected error type %s at index %d, got %s", expectedType, i, foundErrorTypes[i])
				}
			}
		})
	}
}

func TestWebSocketCatchDecoratorCodeGeneration(t *testing.T) {
	tests := []struct {
		name            string
		input           string
		expectedSnippets []string // Code snippets we expect to find
	}{
		{
			name: "Single error type WebSocket handler",
			input: `
package main

@WebSocketGateway(8080)
type ChatGateway struct {}

@Catch(BadRequestError)
@SubscribeMessage("send_message")
func HandleSendMessage(@ConnectedSocket() client *WebSocketClient) {}
`,
			expectedSnippets: []string{
				"func (g *ChatGateway) handleHandleSendMessageError(",
				"err error,",
				"client *WebSocketClient,",
				"eventName string",
				"case *BadRequestError:",
				"client.Emit(\"error\"",
				"\"type\": \"BadRequestError\"",
				"\"event\": eventName",
			},
		},
		{
			name: "Multiple error types WebSocket handler",
			input: `
package main

@WebSocketGateway(8080)
type ChatGateway struct {}

@Catch(BadRequestError, ValidationError)
@SubscribeMessage("send_message")
func HandleSendMessage(@ConnectedSocket() client *WebSocketClient) {}
`,
			expectedSnippets: []string{
				"func (g *ChatGateway) handleHandleSendMessageError(",
				"case *BadRequestError:",
				"case *ValidationError:",
				"client.Emit(\"error\"",
				"\"type\": \"BadRequestError\"",
				"\"type\": \"ValidationError\"",
				"\"message\": \"Validation failed\"",
			},
		},
		{
			name: "Global WebSocket gateway error handler",
			input: `
package main

@WebSocketGateway(8080)
@Catch()
type ChatGateway struct {}

@SubscribeMessage("send_message")
func HandleSendMessage(@ConnectedSocket() client *WebSocketClient) {}
`,
			expectedSnippets: []string{
				"func (g *ChatGateway) handleError(",
				"err error,",
				"client *WebSocketClient,",
				"eventName string",
				"// Global WebSocket error handler - catches all errors",
				"client.Emit(\"error\"",
				"\"type\": \"InternalServerError\"",
			},
		},
		{
			name: "Gateway and method level WebSocket handlers",
			input: `
package main

@WebSocketGateway(8080)
@Catch(NotFoundError)
type ChatGateway struct {}

@Catch(ValidationError)
@SubscribeMessage("send_message")
func HandleSendMessage(@ConnectedSocket() client *WebSocketClient) {}
`,
			expectedSnippets: []string{
				"func (g *ChatGateway) handleNotFoundErrorError(",
				"func (g *ChatGateway) handleHandleSendMessageError(",
				"case *NotFoundError:",
				"case *ValidationError:",
				"\"type\": \"NotFoundError\"",
				"\"type\": \"ValidationError\"",
			},
		},
		{
			name: "WebSocket lifecycle handler with catch",
			input: `
package main

@WebSocketGateway(8080)
type ChatGateway struct {}

@Catch(AuthenticationError)
@OnGatewayConnection()
func HandleConnection(@ConnectedSocket() client *WebSocketClient) {}
`,
			expectedSnippets: []string{
				"func (g *ChatGateway) handleHandleConnectionError(",
				"case *AuthenticationError:",
				"client.Emit(\"error\"",
				"\"type\": \"AuthenticationError\"",
				"\"message\": \"Authentication failed\"",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the input
			file, err := ParseGofaFile(tt.input)
			if err != nil {
				t.Fatalf("Parsing failed: %v", err)
			}

			// Generate code
			generator := NewCodeGenerator("main")
			generatedCode, err := generator.GenerateGoCode(file)
			if err != nil {
				t.Fatalf("Code generation failed: %v", err)
			}

			// Check for expected code snippets
			for _, snippet := range tt.expectedSnippets {
				if !strings.Contains(generatedCode, snippet) {
					t.Errorf("Expected code snippet not found: %s", snippet)
				}
			}
		})
	}
}

func TestWebSocketCatchDecoratorWithParameterDecorators(t *testing.T) {
	tests := []struct {
		name            string
		input           string
		expectedSnippets []string
	}{
		{
			name: "WebSocket error handler with @Exception() parameter",
			input: `
package main

@WebSocketGateway(8080)
type ChatGateway struct {}

@Catch(ValidationError)
@SubscribeMessage("send_message")
func HandleSendMessage(
	@Exception() err error,
	@ConnectedSocket() client *WebSocketClient,
	@EventName() eventName string
) {}
`,
			expectedSnippets: []string{
				"err := wsCtx.Error",
				"client := wsCtx.Client",
				"eventName := wsCtx.EventName",
				"case *ValidationError:",
				"client.Emit(\"error\"",
			},
		},
		{
			name: "WebSocket error handler with mixed parameters",
			input: `
package main

@WebSocketGateway(8080)
type ChatGateway struct {}

@Catch(BadRequestError)
@SubscribeMessage("send_message")
func HandleSendMessage(
	@MessageBody() message *ChatMessage,
	@Exception() err error,
	@ConnectedSocket() client *WebSocketClient,
	@Session("userId") userId string
) {}
`,
			expectedSnippets: []string{
				"var message *ChatMessage",
				"err := wsCtx.Error",
				"client := wsCtx.Client",
				"if sessionValue := client.Session().Get(\"userId\"); sessionValue != nil {",
				"case *BadRequestError:",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, err := ParseGofaFile(tt.input)
			if err != nil {
				t.Fatalf("Parsing failed: %v", err)
			}

			generator := NewCodeGenerator("main")
			generatedCode, err := generator.GenerateGoCode(file)
			if err != nil {
				t.Fatalf("Code generation failed: %v", err)
			}

			for _, snippet := range tt.expectedSnippets {
				if !strings.Contains(generatedCode, snippet) {
					t.Errorf("Expected code snippet not found: %s", snippet)
				}
			}
		})
	}
}

func TestWebSocketCatchDecoratorErrorHandling(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		shouldError bool
		errorMsg    string
	}{
		{
			name: "Valid single WebSocket error type",
			input: `
package main

@WebSocketGateway(8080)
type ChatGateway struct {}

@Catch(BadRequestError)
@SubscribeMessage("send_message")
func HandleSendMessage(@ConnectedSocket() client *WebSocketClient) {}
`,
			shouldError: false,
		},
		{
			name: "Valid multiple WebSocket error types",
			input: `
package main

@WebSocketGateway(8080)
type ChatGateway struct {}

@Catch(BadRequestError, ValidationError, NotFoundError)
@SubscribeMessage("send_message")
func HandleSendMessage(@ConnectedSocket() client *WebSocketClient) {}
`,
			shouldError: false,
		},
		{
			name: "Valid global WebSocket error handler",
			input: `
package main

@WebSocketGateway(8080)
@Catch()
type ChatGateway struct {}
`,
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, err := ParseGofaFile(tt.input)

			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error but parsing succeeded")
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error message containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected parsing error: %v", err)
			}

			// Generate code to ensure no runtime errors
			generator := NewCodeGenerator("main")
			_, err = generator.GenerateGoCode(file)
			if err != nil {
				t.Fatalf("Code generation failed: %v", err)
			}
		})
	}
}

func TestWebSocketCatchDecoratorComplexScenarios(t *testing.T) {
	tests := []struct {
		name            string
		input           string
		expectedSnippets []string
	}{
		{
			name: "Multiple WebSocket methods with different error handlers",
			input: `
package main

@WebSocketGateway(8080)
type ChatGateway struct {}

@Catch(NotFoundError)
@SubscribeMessage("get_user")
func GetUser(@ConnectedSocket() client *WebSocketClient) {}

@Catch(ValidationError)
@SubscribeMessage("send_message")
func SendMessage(@ConnectedSocket() client *WebSocketClient) {}

@Catch(UnauthorizedError)
@SubscribeMessage("delete_user")
func DeleteUser(@ConnectedSocket() client *WebSocketClient) {}
`,
			expectedSnippets: []string{
				"handleGetUserError",
				"handleSendMessageError", 
				"handleDeleteUserError",
				"case *NotFoundError:",
				"case *ValidationError:",
				"case *UnauthorizedError:",
				"\"type\": \"NotFoundError\"",
				"\"type\": \"ValidationError\"",
				"\"type\": \"UnauthorizedError\"",
			},
		},
		{
			name: "WebSocket gateway with multiple error handlers",
			input: `
package main

@WebSocketGateway(8080)
@Catch(InternalServerError)
type ChatGateway struct {}

@Catch(ValidationError)
@SubscribeMessage("send_message")
func SendMessage(@ConnectedSocket() client *WebSocketClient) {}

@Catch(NotFoundError)
@SubscribeMessage("get_user")
func GetUser(@ConnectedSocket() client *WebSocketClient) {}
`,
			expectedSnippets: []string{
				"handleInternalServerErrorError",
				"handleSendMessageError",
				"handleGetUserError",
				"case *InternalServerError:",
				"case *ValidationError:",
				"case *NotFoundError:",
				"\"type\": \"InternalServerError\"",
				"\"type\": \"ValidationError\"",
				"\"type\": \"NotFoundError\"",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, err := ParseGofaFile(tt.input)
			if err != nil {
				t.Fatalf("Parsing failed: %v", err)
			}

			generator := NewCodeGenerator("main")
			generatedCode, err := generator.GenerateGoCode(file)
			if err != nil {
				t.Fatalf("Code generation failed: %v", err)
			}

			for _, snippet := range tt.expectedSnippets {
				if !strings.Contains(generatedCode, snippet) {
					t.Errorf("Expected code snippet not found: %s", snippet)
				}
			}
		})
	}
}