# Gofasta HTTP Example

This example demonstrates how to use the Gofasta HTTP package to create a web server with controllers, dependency injection, WebSocket support, and static file serving.

## Features Demonstrated

- **Dependency Injection**: Services and controllers with `inject:""` tags
- **HTTP Controllers**: RESTful endpoints with route metadata
- **WebSocket Support**: Real-time chat functionality
- **Static File Serving**: Serving HTML, CSS, and other static assets
- **Service Lifecycle**: Initialize and cleanup methods
- **Middleware**: CORS, compression, and recovery middleware

## Project Structure

```
http-example/
├── main.go           # Main application with HTTP server setup
├── static/           # Static files directory
│   ├── index.html    # Interactive demo page
│   └── style.css     # Styling for the demo
└── README.md         # This file
```

## Running the Example

From the project root:

```bash
# Build and run
go run examples/http-example/main.go

# Or build first
go build examples/http-example/main.go
./main
```

The server will start on `http://localhost:8080`

## API Endpoints

The example includes controllers that register the following routes (via metadata):

- **Health Controller**:
  - `GET /health` - Health check endpoint
  - `GET /info` - Application information

- **User Controller**:
  - `GET /users` - Get all users
  - `GET /users/:id` - Get user by ID
  - `POST /users` - Create new user

- **WebSocket**:
  - `GET /ws` - WebSocket chat endpoint

- **Static Files**:
  - `GET /static/*` - Serve static files

## Interactive Demo

Visit `http://localhost:8080/static/index.html` for an interactive demo that includes:

- API endpoint testing
- WebSocket chat functionality
- Real-time user management

## Example API Calls

```bash
# Health check
curl http://localhost:8080/health

# Get all users
curl http://localhost:8080/users

# Create a new user
curl -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{"name":"Alice","email":"alice@example.com"}'

# Get user by ID
curl http://localhost:8080/users/1
```

## WebSocket Testing

You can test the WebSocket endpoint using the browser demo or with a WebSocket client:

```javascript
const ws = new WebSocket('ws://localhost:8080/ws');
ws.onmessage = (event) => console.log(JSON.parse(event.data));
ws.send('Hello Gofasta!');
```

## Key Components

### Services
- **Logger**: Simple logging service
- **UserService**: User management with in-memory storage

### Controllers
- **UserController**: RESTful user operations
- **HealthController**: Health and info endpoints

### WebSocket Handler
- **ChatWebSocketHandler**: Echo server for real-time messaging

## Architecture Notes

This example demonstrates:

1. **Modular Design**: Clear separation between services, controllers, and handlers
2. **Dependency Injection**: Automatic wiring of dependencies using struct tags
3. **Controller Metadata**: Route registration via annotations (comments)
4. **Service Lifecycle**: Proper initialization and cleanup
5. **WebSocket Integration**: Real-time communication capabilities
6. **Static File Serving**: Asset delivery with proper caching headers

The HTTP package automatically handles:
- Route registration from controller metadata
- Request context creation and management
- Middleware pipeline execution
- Error handling and recovery
- CORS headers
- Gzip compression