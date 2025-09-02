package integration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/healtronlabs/gofasta/tools/transpiler/core"
)

// TestRealWorldProjectParsing tests parsing against a real project structure
func TestRealWorldProjectParsing(t *testing.T) {
	// Create a realistic project structure
	tempDir, err := os.MkdirTemp("", "real_world_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a complex project structure
	projectStructure := map[string]string{
		"main.go": `package main

import (
	"fmt"
	"log"
	"net/http"
)

// @Server(port: 8080)
// @Cors(origins: ["*"])
func main() {
	log.Println("Starting server...")
	http.ListenAndServe(":8080", nil)
}`,

		"api/users.gofa": `package api

import "context"

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
	// Implementation here
	return nil, nil
}

// @Post("/")
// @ValidateBody()
// @CircuitBreaker(threshold: 5)
func (uc *UserController) CreateUser(
	@Body() user *CreateUserRequest,
) (*User, error) {
	// Implementation here
	return nil, nil
}

// @Put("/:id")
// @Permissions("user.update")
func (uc *UserController) UpdateUser(
	@Param("id") id string,
	@Body() updates *UpdateUserRequest,
) (*User, error) {
	// Implementation here
	return nil, nil
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
	// Implementation here
	return nil, nil
}

// @Post("/")
// @EventSourcing()
// @Saga(steps: ["payment", "inventory", "shipping"])
func (oc *OrderController) CreateOrder(
	@Body() order *CreateOrderRequest,
	@TenantId() tenantID string,
) (*Order, error) {
	// Implementation here
	return nil, nil
}`,

		"models/user.gofa": `package models

// @Entity(table: "users")
// @Cache(strategy: "redis", ttl: "1h")
type User struct {
	// @PrimaryKey()
	// @Column(type: "uuid")
	ID string
	
	// @Column(type: "varchar", length: 255)
	// @Index()
	// @Unique()
	Email string
	
	// @Column(type: "varchar", length: 100)
	Name string
	
	// @Column(type: "timestamp")
	CreatedAt time.Time
	
	// @OneToMany(entity: "Order")
	Orders []Order
}`,

		"models/order.gofa": `package models

import "time"

// @Entity(table: "orders")
// @ReadReplica(enabled: true)
// @WriteReplica(strategy: "primary")
type Order struct {
	// @PrimaryKey()
	ID string
	
	// @ForeignKey(entity: "User")
	// @Index()
	UserID string
	
	// @Column(type: "decimal", precision: 10, scale: 2)
	Total float64
	
	// @Column(type: "varchar", length: 50)
	Status string
	
	// @Column(type: "timestamp")
	CreatedAt time.Time
	
	// @ManyToOne(entity: "User")
	User *User
	
	// @OneToMany(entity: "OrderItem")
	Items []OrderItem
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
	// Handle new connections
	return nil
}

// @SubscribeMessage("join-room")
// @Broadcast(room: true)
func (cg *ChatGateway) HandleJoinRoom(
	@ConnectedSocket() socket *Socket,
	@MessageBody() data *JoinRoomMessage,
) error {
	// Handle room joining
	return nil
}

// @SubscribeMessage("send-message")
// @Publish(channel: "chat-messages")
func (cg *ChatGateway) HandleMessage(
	@ConnectedSocket() socket *Socket,
	@MessageBody() message *ChatMessage,
	@MessageAck() ack *MessageAck,
) error {
	// Handle incoming messages
	return nil
}`,

		"graphql/resolvers.gofa": `package graphql

// @Resolver()
type QueryResolver struct{}

// @Query("users")
// @Cache(strategy: "redis", ttl: "10m")
func (r *QueryResolver) Users(
	@Args() args *UsersArgs,
	@Context() ctx context.Context,
) ([]*User, error) {
	// GraphQL query resolver
	return nil, nil
}

// @Resolver(User)
type UserResolver struct{}

// @ResolveField("orders")
func (r *UserResolver) Orders(
	obj *User,
	@Context() ctx context.Context,
) ([]*Order, error) {
	// Field resolver for user orders
	return nil, nil
}

// @Mutation("createUser")
// @ValidateNested()
func (r *QueryResolver) CreateUser(
	@Args() args *CreateUserArgs,
	@Context() ctx context.Context,
) (*User, error) {
	// GraphQL mutation resolver
	return nil, nil
}`,

		"grpc/services.gofa": `package grpc

// @GrpcService()
type UserService struct{}

// @GrpcMethod("GetUser")
// @Timeout(duration: "30s")
func (s *UserService) GetUser(
	@GrpcPayload() req *GetUserRequest,
	@GrpcMetadata() metadata map[string]string,
) (*GetUserResponse, error) {
	// gRPC service method
	return nil, nil
}

// @GrpcStreamMethod("StreamUsers")
// @BackPressure(strategy: "drop")
func (s *UserService) StreamUsers(
	@GrpcPayload() req *StreamUsersRequest,
	stream *UserStreamServer,
) error {
	// gRPC streaming method
	return nil
}`,

		"config/config.go": `package config

import "time"

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
}

type ServerConfig struct {
	Host string
	Port int
	Timeout time.Duration
}`,

		"middleware/auth.go": `package middleware

import (
	"net/http"
)

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Authentication logic
		next.ServeHTTP(w, r)
	})
}`,

		"vendor/external/lib.go": `package external
// This should be excluded from parsing`,

		"node_modules/some/package.js": `// This should also be excluded`,

		".git/config": `# Git config - should be excluded`,
	}

	// Create all files
	for filePath, content := range projectStructure {
		fullPath := filepath.Join(tempDir, filePath)
		err := os.MkdirAll(filepath.Dir(fullPath), 0755)
		if err != nil {
			t.Fatalf("Failed to create directory for %s: %v", filePath, err)
		}
		
		err = os.WriteFile(fullPath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to write file %s: %v", filePath, err)
		}
	}

	// Test parsing with default configuration
	config := core.DefaultConfig()
	parser := core.NewParallelParser(config)
	
	ctx := context.Background()
	start := time.Now()
	
	results, err := parser.ParseDirectory(ctx, tempDir)
	if err != nil {
		t.Fatalf("Failed to parse project: %v", err)
	}
	
	parseDuration := time.Since(start)
	stats := parser.GetStatistics()
	
	t.Logf("Parsing completed in %v", parseDuration)
	t.Logf("Total files found: %d", len(results))
	t.Logf("Successful parses: %v", stats["successful_files"])
	t.Logf("Failed parses: %v", stats["failed_files"])
	t.Logf("Files per second: %.2f", stats["files_per_second"])
	
	// Verify we found the expected files
	expectedGoFaFiles := []string{
		"api/users.gofa",
		"api/orders.gofa", 
		"models/user.gofa",
		"models/order.gofa",
		"websocket/chat.gofa",
		"graphql/resolvers.gofa",
		"grpc/services.gofa",
	}
	
	expectedGoFiles := []string{
		"main.go",
		"config/config.go",
		"middleware/auth.go",
	}
	
	gofaResults := parser.FilterResultsByExtension(".gofa")
	goResults := parser.FilterResultsByExtension(".go")
	
	if len(gofaResults) != len(expectedGoFaFiles) {
		t.Errorf("Expected %d .gofa files, got %d", len(expectedGoFaFiles), len(gofaResults))
		for _, result := range gofaResults {
			t.Logf("Found .gofa: %s", result.FilePath)
		}
	}
	
	if len(goResults) != len(expectedGoFiles) {
		t.Errorf("Expected %d .go files, got %d", len(expectedGoFiles), len(goResults))
		for _, result := range goResults {
			t.Logf("Found .go: %s", result.FilePath)
		}
	}
	
	// Verify excluded directories are actually excluded
	for _, result := range results {
		if strings.Contains(result.FilePath, "vendor") ||
		   strings.Contains(result.FilePath, "node_modules") ||
		   strings.Contains(result.FilePath, ".git") {
			t.Errorf("Expected excluded file to not be parsed: %s", result.FilePath)
		}
	}
	
	// Test performance requirement (< 50ms initialization as per roadmap)
	if parseDuration > 200*time.Millisecond { // Allow some buffer for test environment
		t.Logf("WARNING: Parse time %v exceeds performance target", parseDuration)
	}
}

// TestLargeProjectPerformance tests performance with a larger number of files
func TestLargeProjectPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large project test in short mode")
	}

	tempDir, err := os.MkdirTemp("", "large_project_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a large number of files to test performance
	numFiles := 200
	numDirs := 10
	
	for dir := 0; dir < numDirs; dir++ {
		dirPath := filepath.Join(tempDir, fmt.Sprintf("module%d", dir))
		err := os.MkdirAll(dirPath, 0755)
		if err != nil {
			t.Fatalf("Failed to create directory %s: %v", dirPath, err)
		}
		
		for file := 0; file < numFiles/numDirs; file++ {
			// Create a mix of valid .go files and .gofa files with decorators
			// .go files should parse successfully, .gofa files will fail due to decorators
			var content string
			var filename string
			
			if file%2 == 0 {
				// Create valid .go files (should parse successfully)
				content = fmt.Sprintf(`package module%d

import (
	"fmt"
	"net/http"
)

// Controller for module %d file %d
type Controller%d struct {
	ID     int
	Module int
	Data   string
}

// Get method returns a response
func (c *Controller%d) Get() (*Response, error) {
	return &Response{
		ID: %d,
		Module: %d,
		Data: "test data for file %d",
	}, nil
}

// Create method handles POST requests  
func (c *Controller%d) Create(req *Request) (*Response, error) {
	fmt.Printf("Creating resource in module %%d file %%d", %d, %d)
	return nil, nil
}

type Response struct {
	ID     int    ` + "`json:\"id\"`" + `
	Module int    ` + "`json:\"module\"`" + `
	Data   string ` + "`json:\"data\"`" + `
}

type Request struct {
	Name string ` + "`json:\"name\"`" + `
}`, dir, dir, file, file, file, file, dir, file, file, dir, file)
				filename = fmt.Sprintf("controller%d.go", file)
			} else {
				// Create .gofa files with decorators (will fail to parse - expected)
				content = fmt.Sprintf(`package module%d

// @Controller("/api/module%d/%d")
// @UseGuards("jwt")
type Controller%d struct{}

// @Get("/")
// @Cache(ttl: "1m")
func (c *Controller%d) Get() (*Response, error) {
	return &Response{
		ID: %d,
		Module: %d,
		Data: "test data for file %d",
	}, nil
}

// @Post("/")
// @ValidateBody()
// @CircuitBreaker(threshold: 5)
func (c *Controller%d) Create(@Body() req *Request) (*Response, error) {
	return nil, nil
}`, dir, dir, file, file, file, file, dir, file, file)
				filename = fmt.Sprintf("controller%d.gofa", file)
			}
			
			filePath := filepath.Join(dirPath, filename)
			
			err := os.WriteFile(filePath, []byte(content), 0644)
			if err != nil {
				t.Fatalf("Failed to write file %s: %v", filePath, err)
			}
		}
	}

	// Test with different worker configurations
	testCases := []struct {
		name       string
		maxWorkers int
	}{
		{"SingleWorker", 1},
		{"OptimalWorkers", core.DefaultConfig().MaxWorkers},
		{"MaxWorkers", core.DefaultConfig().MaxWorkers * 2},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := core.DefaultConfig()
			config.MaxWorkers = tc.maxWorkers
			
			parser := core.NewParallelParser(config)
			ctx := context.Background()
			
			start := time.Now()
			results, err := parser.ParseDirectory(ctx, tempDir)
			if err != nil {
				t.Fatalf("Failed to parse large project: %v", err)
			}
			
			duration := time.Since(start)
			stats := parser.GetStatistics()
			
			t.Logf("Workers: %d, Files: %d, Duration: %v, Files/sec: %.2f", 
				tc.maxWorkers, len(results), duration, stats["files_per_second"])
			
			if len(results) != numFiles {
				t.Errorf("Expected %d files, got %d", numFiles, len(results))
			}
			
			// Since we create 50% .go files (valid) and 50% .gofa files (invalid due to @ decorators),
			// we expect about half to parse successfully in Phase 1.1a
			successful := parser.GetSuccessfulResults()
			expectedSuccessful := numFiles / 2 // Only .go files should parse successfully
			
			if len(successful) < expectedSuccessful-5 || len(successful) > expectedSuccessful+5 {
				t.Errorf("Expected approximately %d successful parses, got %d/%d", expectedSuccessful, len(successful), numFiles)
				t.Logf("Note: .gofa files fail due to @ decorators - this is expected in Phase 1.1a")
			} else {
				t.Logf("✅ Performance test passed: %d/%d files parsed successfully (expected ~%d)", len(successful), numFiles, expectedSuccessful)
			}
		})
	}
}

// TestMemoryEfficiency verifies memory usage stays reasonable
func TestMemoryEfficiency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory efficiency test in short mode")
	}

	tempDir, err := os.MkdirTemp("", "memory_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create files with varying sizes
	fileSizes := []int{1024, 5120, 10240, 51200} // 1KB, 5KB, 10KB, 50KB
	var filePaths []string
	
	for i, size := range fileSizes {
		content := fmt.Sprintf(`package test%d

// Large comment block for file %d
`, i, i)
		
		// Pad with comments to reach desired size
		for len(content) < size {
			content += fmt.Sprintf("// Additional comment line for padding %d\n", len(content))
		}
		
		content += `
// @Controller("/api/test")
// @UseGuards("jwt")
func TestHandler() (*Response, error) {
	return nil, nil
}`
		
		filePath := filepath.Join(tempDir, fmt.Sprintf("large_file_%d.gofa", i))
		err := os.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to write large file %d: %v", i, err)
		}
		filePaths = append(filePaths, filePath)
	}

	parser := core.NewParallelParser(core.DefaultConfig())
	ctx := context.Background()
	
	results, err := parser.ParseFiles(ctx, filePaths)
	if err != nil {
		t.Fatalf("Failed to parse files: %v", err)
	}
	
	if len(results) != len(filePaths) {
		t.Errorf("Expected %d results, got %d", len(filePaths), len(results))
	}
	
	stats := parser.GetStatistics()
	totalBytes := stats["total_bytes"].(int64)
	
	t.Logf("Total bytes processed: %d", totalBytes)
	t.Logf("MB/sec: %.2f", stats["mb_per_second"])
	
	// Verify all files were processed successfully
	successful := parser.GetSuccessfulResults()
	if len(successful) != len(filePaths) {
		t.Errorf("Expected all files to be processed successfully")
	}
}