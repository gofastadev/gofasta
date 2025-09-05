// Package main demonstrates the usage of Gofasta's high-performance parallel parser.
// This example shows how to use Phase 1.1a: go/parser with parallel file processing.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/healtronlabs/gofasta/transpiler/core"
)

func main() {
	fmt.Println("🚀 Gofasta Parallel Parser Example")
	fmt.Println("==================================")

	// Create example files for demonstration
	exampleDir, err := createExampleFiles()
	if err != nil {
		log.Fatalf("Failed to create example files: %v", err)
	}
	defer os.RemoveAll(exampleDir)

	// Example 1: Basic Usage with Default Configuration
	fmt.Println("\n📋 Example 1: Basic Usage")
	basicExample(exampleDir)

	// Example 2: Custom Configuration
	fmt.Println("\n⚙️  Example 2: Custom Configuration")
	customConfigExample(exampleDir)

	// Example 3: Performance Comparison
	fmt.Println("\n⚡ Example 3: Performance Comparison")
	performanceExample(exampleDir)

	// Example 4: Error Handling
	fmt.Println("\n🛠️  Example 4: Error Handling")
	errorHandlingExample(exampleDir)

	// Example 5: File Filtering
	fmt.Println("\n🔍 Example 5: File Filtering")
	filteringExample(exampleDir)
}

func basicExample(exampleDir string) {
	// Create parser with default configuration
	parser := core.NewParallelParser(core.DefaultConfig())

	// Parse directory
	ctx := context.Background()
	results, err := parser.ParseDirectory(ctx, exampleDir)
	if err != nil {
		log.Printf("Error parsing directory: %v", err)
		return
	}

	fmt.Printf("📁 Parsed %d files successfully\n", len(results))

	// Get statistics
	stats := parser.GetStatistics()
	fmt.Printf("📊 Statistics:\n")
	fmt.Printf("   • Total files: %v\n", stats["total_files"])
	fmt.Printf("   • Successful: %v\n", stats["successful_files"])
	fmt.Printf("   • Failed: %v\n", stats["failed_files"])
	fmt.Printf("   • Duration: %vms\n", stats["total_duration_ms"])
	fmt.Printf("   • Files/second: %.2f\n", stats["files_per_second"])
}

func customConfigExample(exampleDir string) {
	// Create custom configuration
	config := &core.ParserConfig{
		MaxWorkers:    2,    // Use 2 workers
		ParseComments: true, // Include comments
		AllowErrors:   true, // Continue on errors
	}

	parser := core.NewParallelParser(config)

	ctx := context.Background()
	results, err := parser.ParseDirectory(ctx, exampleDir)
	if err != nil {
		log.Printf("Error parsing directory: %v", err)
		return
	}

	fmt.Printf("📁 Custom config parsed %d files\n", len(results))

	// Filter results by extension
	gofaResults := parser.FilterResultsByExtension(".gofa")
	fmt.Printf("🎯 Found %d .gofa files\n", len(gofaResults))

	for _, result := range gofaResults {
		fmt.Printf("   • %s (%d bytes, %v)\n",
			filepath.Base(result.FilePath),
			result.Size,
			result.Duration)
	}
}

func performanceExample(exampleDir string) {
	// Compare single worker vs multiple workers
	testConfigs := []struct {
		name       string
		maxWorkers int
	}{
		{"Single Worker", 1},
		{"Multiple Workers", 4},
		{"Optimal Workers", 0}, // Will use runtime.NumCPU()
	}

	for _, tc := range testConfigs {
		config := core.DefaultConfig()
		config.MaxWorkers = tc.maxWorkers

		parser := core.NewParallelParser(config)
		ctx := context.Background()

		start := time.Now()
		results, err := parser.ParseDirectory(ctx, exampleDir)
		duration := time.Since(start)

		if err != nil {
			log.Printf("Error in %s: %v", tc.name, err)
			continue
		}

		stats := parser.GetStatistics()
		fmt.Printf("⚡ %s: %d files in %v (%.2f files/sec)\n",
			tc.name, len(results), duration, stats["files_per_second"])
	}
}

func errorHandlingExample(exampleDir string) {
	// Create a file with syntax errors
	invalidFile := filepath.Join(exampleDir, "invalid.gofa")
	invalidContent := `package invalid

// @Controller("/api")
func BrokenFunction( {
	// Missing closing parenthesis
	return "this will fail"
}`

	err := os.WriteFile(invalidFile, []byte(invalidContent), 0644)
	if err != nil {
		log.Printf("Failed to create invalid file: %v", err)
		return
	}

	// Parse with error handling
	config := core.DefaultConfig()
	config.AllowErrors = true // Continue parsing even with errors

	parser := core.NewParallelParser(config)
	ctx := context.Background()

	results, err := parser.ParseDirectory(ctx, exampleDir)
	if err != nil {
		log.Printf("Parsing failed: %v", err)
		return
	}

	// Show successful vs failed results
	successful := parser.GetSuccessfulResults()
	fmt.Printf("✅ Successfully parsed: %d files\n", len(successful))

	// Show files with errors
	for _, result := range results {
		if result.Error != nil {
			fmt.Printf("❌ Error in %s: %v\n",
				filepath.Base(result.FilePath), result.Error)
		}
	}

	// Clean up
	os.Remove(invalidFile)
}

func filteringExample(exampleDir string) {
	parser := core.NewParallelParser(core.DefaultConfig())
	ctx := context.Background()

	_, err := parser.ParseDirectory(ctx, exampleDir)
	if err != nil {
		log.Printf("Error parsing directory: %v", err)
		return
	}

	// Filter by different extensions
	goFiles := parser.FilterResultsByExtension(".go")
	gofaFiles := parser.FilterResultsByExtension(".gofa")

	fmt.Printf("📋 File breakdown:\n")
	fmt.Printf("   • .go files: %d\n", len(goFiles))
	fmt.Printf("   • .gofa files: %d\n", len(gofaFiles))

	// Show detailed information for .gofa files
	fmt.Printf("\n🎯 Gofasta files details:\n")
	for _, result := range gofaFiles {
		if result.File != nil && result.File.Doc != nil {
			fmt.Printf("   • %s: %d comments\n",
				filepath.Base(result.FilePath),
				len(result.File.Doc.List))
		} else {
			fmt.Printf("   • %s: no doc comments\n",
				filepath.Base(result.FilePath))
		}
	}
}

func createExampleFiles() (string, error) {
	tempDir, err := os.MkdirTemp("", "gofasta_parser_example")
	if err != nil {
		return "", err
	}

	// Create example files with Gofasta decorators
	files := map[string]string{
		"main.go": `package main

import (
	"fmt"
	"log"
	"net/http"
)

// @Server(port: 8080)
// @Cors(origins: ["*"])
func main() {
	fmt.Println("Starting Gofasta server...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}`,

		"api/users.gofa": `package api

// @Controller("/api/users")
// @UseGuards("jwt")
// @RateLimit(requests: 100, window: "1m")
type UserController struct{}

// @Get("/")
// @Cache(ttl: "5m")
// @Prometheus(metric: "user_requests")
func (uc *UserController) GetUsers(
	@Query("limit") limit int,
	@Query("offset") offset int,
) ([]*User, error) {
	// Get all users with pagination
	return getAllUsers(limit, offset)
}

// @Post("/")
// @ValidateBody()
// @CircuitBreaker(threshold: 5)
func (uc *UserController) CreateUser(
	@Body() user *CreateUserRequest,
) (*User, error) {
	// Create a new user
	return createUser(user)
}`,

		"api/orders.gofa": `package api

// @Controller("/api/orders")
// @Actor(name: "order-processor")
// @Supervisor(strategy: "OneForOne")
type OrderController struct{}

// @Get("/:id")
// @Transaction()
// @Tracing(enabled: true)
func (oc *OrderController) GetOrder(
	@Param("id") orderID string,
	@Headers("Authorization") auth string,
) (*Order, error) {
	// Get order by ID
	return getOrderByID(orderID)
}`,

		"models/user.gofa": `package models

// @Entity(table: "users")
// @Cache(strategy: "redis", ttl: "1h")
type User struct {
	// @PrimaryKey()
	// @Column(type: "uuid")
	ID string ` + "`json:\"id\" db:\"id\"`" + `
	
	// @Column(type: "varchar", length: 255)
	// @Index()
	// @Unique()
	Email string ` + "`json:\"email\" db:\"email\"`" + `
	
	// @Column(type: "varchar", length: 100)
	Name string ` + "`json:\"name\" db:\"name\"`" + `
}`,

		"websocket/chat.gofa": `package websocket

// @WebSocketGateway(port: 8081, cors: true)
// @Namespace("chat")
type ChatGateway struct{}

// @OnGatewayConnection()
// @Rooms()
func (cg *ChatGateway) HandleConnection(
	@ConnectedSocket() socket *Socket,
	@MessageBody() auth *AuthMessage,
) error {
	// Handle new WebSocket connections
	return joinDefaultRoom(socket)
}

// @SubscribeMessage("join-room")
// @Broadcast(room: true)
func (cg *ChatGateway) HandleJoinRoom(
	@ConnectedSocket() socket *Socket,
	@MessageBody() data *JoinRoomMessage,
) error {
	// Handle room joining
	return joinRoom(socket, data.RoomID)
}`,

		"config.go": `package main

type Config struct {
	Port     int    ` + "`json:\"port\"`" + `
	Host     string ` + "`json:\"host\"`" + `
	Database string ` + "`json:\"database\"`" + `
}

func LoadConfig() *Config {
	return &Config{
		Port:     8080,
		Host:     "localhost", 
		Database: "postgres://localhost/gofasta",
	}
}`,
	}

	// Create all files
	for filePath, content := range files {
		fullPath := filepath.Join(tempDir, filePath)
		err := os.MkdirAll(filepath.Dir(fullPath), 0755)
		if err != nil {
			return "", err
		}

		err = os.WriteFile(fullPath, []byte(content), 0644)
		if err != nil {
			return "", err
		}
	}

	return tempDir, nil
}
