package codegen

import (
	"strings"
	"testing"
)

func TestMessageBodyDecoratorBasic(t *testing.T) {
	input := `
package main

@WebSocketGateway(8080)
type ChatGateway struct {
}

@SubscribeMessage("message")
func HandleMessage(@MessageBody() data *ChatMessage, @ConnectedSocket() client *WebSocketClient) {
}
`

	expected := []string{
		"// Extract message body of type *ChatMessage",
		"var data *ChatMessage",
		"if err := wsCtx.ParseMessageBody(&data); err != nil {",
		"wsCtx.SendError(\"Invalid message body: \" + err.Error())",
		"return",
		"}",
	}

	testMessageBodyGeneration(t, input, expected, "basic MessageBody parameter")
}

func TestMessageBodyDecoratorString(t *testing.T) {
	input := `
package main

@WebSocketGateway(8080)
type ChatGateway struct {
}

@SubscribeMessage("message")
func HandleMessage(@MessageBody() content string, @ConnectedSocket() client *WebSocketClient) {
}
`

	expected := []string{
		"// Extract message body of type string",
		"var content string",
		"if rawMessage := wsCtx.GetRawMessage(); rawMessage != nil {",
		"content = string(rawMessage)",
		"}",
	}

	testMessageBodyGeneration(t, input, expected, "string MessageBody parameter")
}

func TestMessageBodyDecoratorBytes(t *testing.T) {
	input := `
package main

@WebSocketGateway(8080)
type ChatGateway struct {
}

@SubscribeMessage("message")
func HandleMessage(@MessageBody() rawData []byte, @ConnectedSocket() client *WebSocketClient) {
}
`

	expected := []string{
		"// Extract message body of type []byte",
		"var rawData []byte",
		"rawData = wsCtx.GetRawMessage()",
	}

	testMessageBodyGeneration(t, input, expected, "bytes MessageBody parameter")
}

func TestMessageBodyDecoratorInterface(t *testing.T) {
	input := `
package main

@WebSocketGateway(8080)
type ChatGateway struct {
}

@SubscribeMessage("message")
func HandleMessage(@MessageBody() data interface{}, @ConnectedSocket() client *WebSocketClient) {
}
`

	expected := []string{
		"// Extract message body of type interface{}",
		"var data interface{}",
		"if err := wsCtx.ParseMessageBody(&data); err != nil {",
		"wsCtx.SendError(\"Invalid message body: \" + err.Error())",
		"return",
		"}",
	}

	testMessageBodyGeneration(t, input, expected, "interface{} MessageBody parameter")
}

func TestMessageBodyDecoratorMultipleParameters(t *testing.T) {
	input := `
package main

@WebSocketGateway(8080)
type ChatGateway struct {
}

@SubscribeMessage("message")
func HandleMessage(@MessageBody() data *ChatMessage, @ConnectedSocket() client *WebSocketClient, @MessageAck() ack *AckCallback) {
}
`

	expected := []string{
		"// Extract message body of type *ChatMessage",
		"var data *ChatMessage", 
		"if err := wsCtx.ParseMessageBody(&data); err != nil {",
		"wsCtx.SendError(\"Invalid message body: \" + err.Error())",
		"return",
		"}",
	}

	testMessageBodyGeneration(t, input, expected, "MessageBody with multiple parameters")
}

func TestMessageBodyDecoratorCustomStruct(t *testing.T) {
	input := `
package main

type UserMessage struct {
	UserID  string
	Content string
	RoomID  string
}

@WebSocketGateway(8080)
type ChatGateway struct {
}

@SubscribeMessage("user_message")
func HandleUserMessage(@MessageBody() msg UserMessage, @ConnectedSocket() client *WebSocketClient) {
}
`

	expected := []string{
		"// Extract message body of type UserMessage",
		"var msg UserMessage",
		"if err := wsCtx.ParseMessageBody(&msg); err != nil {",
		"wsCtx.SendError(\"Invalid message body: \" + err.Error())",
		"return",
		"}",
	}

	testMessageBodyGeneration(t, input, expected, "MessageBody with custom struct")
}

// testMessageBodyGeneration helper function to test MessageBody parameter generation
func testMessageBodyGeneration(t *testing.T, input string, expectedStrings []string, testName string) {
	// Parse the input
	file, err := ParseGofaFile(input)
	if err != nil {
		t.Fatalf("Failed to parse .gofa file for %s: %v", testName, err)
	}

	// Generate code
	generator := NewCodeGenerator("main")
	goCode, err := generator.GenerateGoCode(file)
	if err != nil {
		t.Fatalf("Failed to generate Go code for %s: %v", testName, err)
	}

	// Check that all expected strings are present
	for _, expectedStr := range expectedStrings {
		if !strings.Contains(goCode, expectedStr) {
			t.Errorf("Generated code for %s missing expected string: %q", testName, expectedStr)
			t.Logf("Generated code:\n%s", goCode)
		}
	}
}