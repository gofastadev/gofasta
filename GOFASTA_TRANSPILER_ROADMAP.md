# 🔧 GoFasta Transpiler Development Roadmap

> **Goal:** Code Generation & Decorator Transpilation for Go Framework

[![Progress](https://img.shields.io/badge/Progress-70--80%25-yellow)](https://github.com/healtronlabs/gofasta)
[![Status](https://img.shields.io/badge/Status-Active%20Development-green)](https://github.com/healtronlabs/gofasta)
[![Timeline](https://img.shields.io/badge/Timeline-3--4%20months-blue)](https://github.com/healtronlabs/gofasta)

## 📊 Current Status

| ✅**Completed**  | ❌**Pending**              |
| ---------------------- | -------------------------------- |
| Basic REST API transpilation | Advanced decorator patterns |
| Parameter decorators   | Error handling decorators |
| Controller/Service generation | Validation decorators |
| Dependency injection wiring | Middleware decorators |

---

## 🎯 Transpiler-Specific Development Phases

### 📈 Phase 1: Core Decorator Transpilation

> **Target:** All basic decorators working | **Timeline:** 3-4 weeks

#### 🔥 1.1 Enhanced Parameter Decorators `[HIGH PRIORITY]`

- [X] `@Query()` - Generate query parameter extraction code
- [X] `@Headers()` - Generate header extraction code
- [X] `@Req()` - Generate raw request object access
- [X] `@Res()` - Generate raw response object access
- [X] `@Session()` - Generate session data access code
- [X] `@Ip()` - Generate client IP extraction code
- [X] `@HostParam()` - Generate host parameter extraction code

#### 🔥 1.2 Error Handling Decorators `[HIGH PRIORITY]`

- [X] **`@Catch()` decorator transpilation** - Generate error filter registration
  ```gofa
  @Catch(BadRequestError, ValidationError)
  func handleValidationErrors(err error, ctx *RequestContext) {
      // Generate error handling code
  }
  ```
  - Parse error type parameters: `@Catch(BadRequestError)`
  - Generate filter registration code
  - Support multiple error types: `@Catch(Error1, Error2)`
  - Global error filters: `@Catch()`
  - Method/Controller/Global scope handling

#### 🔥 1.3 HTTP Method & Response Decorators `[HIGH PRIORITY]`

- [X] `@Get()`, `@Post()`, `@Put()`, `@Delete()` - Basic route generation
- [X] **`@HttpCode()` decorator** - Generate status code setting
  ```gofa
  @HttpCode(201)
  @Post("/users")
  func createUser() User {
      // Generate: ctx.Status(201)
  }
  ```
- [X] **`@Redirect()` decorator** - Generate redirect responses
  ```gofa
  @Redirect("https://example.com", 302)
  @Get("/old-route")
  func redirectOldRoute() {
      // Generate: ctx.Redirect(302, "https://example.com")
  }
  ```
- [X] **`@Header()` decorator** - Generate custom header setting
  ```gofa
  @Header("X-Custom-Header", "value")
  @Get("/api")
  func getAPI() {
      // Generate: ctx.Header("X-Custom-Header", "value")
  }
  ```

#### 🟡 1.4 Dependency Injection Enhancement `[MEDIUM PRIORITY]`

- [X] **`@Inject()` decorator with tokens** - Generate DI with custom tokens
  ```gofa
  type UserService struct {
      DB *Database `inject:"database"`
      Cache *Redis `inject:"redis"`
  }
  ```
- [X] **Provider factory generation** - Generate custom provider code
- [X] **Scope decorators** - Generate scope-aware DI code
  ```gofa
  @Scope("singleton")
  type UserService struct {}
  
  @Scope("transient")
  type TaskProcessor struct {}
  
  @Scope("request")
  type UserContext struct {}
  ```

---

### 🛠️ Phase 2: Advanced Decorator Patterns

> **Target:** Complex decorators working | **Timeline:** 4-5 weeks

#### 🔥 2.1 Middleware Decorators `[HIGH PRIORITY]`

- [X] **`@UseGuards()` transpilation** - Generate guard middleware
  ```gofa
  @UseGuards(AuthGuard, RoleGuard)
  @Get("/admin")
  func getAdminData() {
      // Generate guard middleware chain
  }
  ```
- [ ] **`@UseInterceptors()` transpilation** - Generate interceptor chains
  ```gofa
  @UseInterceptors(LoggingInterceptor, CacheInterceptor)
  @Get("/data")
  func getData() {
      // Generate interceptor pipeline
  }
  ```
- [ ] **`@UsePipes()` transpilation** - Generate pipe validation
  ```gofa
  @UsePipes(ValidationPipe, TransformPipe)
  @Post("/users")
  func createUser(@Body() data CreateUserDto) {
      // Generate pipe processing
  }
  ```

#### 🔥 2.2 Validation Decorators `[HIGH PRIORITY]`

- [ ] **Built-in validation decorator transpilation**
  ```gofa
  type CreateUserDto struct {
      Email    string `validate:"@IsEmail()"`
      Age      int    `validate:"@Min(18) @Max(120)"`
      Name     string `validate:"@IsNotEmpty() @Length(2,50)"`
      Tags     []string `validate:"@IsArray() @ArrayMinSize(1)"`
  }
  ```
  - Generate validation code for: `@IsString()`, `@IsNumber()`, `@IsEmail()`
  - Generate conditional validation: `@IsOptional()`, `@IsNotEmpty()`
  - Generate range validation: `@Min()`, `@Max()`, `@Length()`
  - Generate complex validation: `@IsArray()`, `@ValidateNested()`

#### 🟡 2.3 Advanced Routing Decorators `[MEDIUM PRIORITY]`

- [ ] **Route versioning transpilation** - Generate versioned routes
  ```gofa
  @Version("v1")
  @Controller("/users")
  type UsersV1Controller struct {}

  @Version("v2") 
  @Controller("/users")
  type UsersV2Controller struct {}
  ```
- [ ] **Route parameter constraints** - Generate parameter validation
  ```gofa
  @Get("/users/:id") // where id is numeric
  func getUser(@Param("id", "int") id int) {
      // Generate type conversion & validation
  }
  ```

---

### 🌟 Phase 3: Specialized Transpilation

> **Target:** All NestJS decorators | **Timeline:** 5-6 weeks

#### 🔥 3.1 Testing Decorators `[HIGH PRIORITY]`

- [ ] **`@Test()` decorator transpilation** - Generate test scaffolding
  ```gofa
  @Test()
  func TestUserService() {
      // Generate test setup/teardown
  }
  ```
- [ ] **Test module transpilation** - Generate test DI containers
- [ ] **Mock decorators** - Generate mock implementations

#### 🔥 3.2 WebSocket Decorators `[HIGH PRIORITY]`

- [ ] **`@WebSocketGateway()` transpilation** - Generate WebSocket handlers
  ```gofa
  @WebSocketGateway(8080)
  type ChatGateway struct {}

  @SubscribeMessage("message")
  func handleMessage(@MessageBody() data string) {
      // Generate WebSocket message handling
  }
  ```

#### 🟡 3.3 GraphQL Decorators `[MEDIUM PRIORITY]`

- [ ] **`@Resolver()` transpilation** - Generate GraphQL resolvers
  ```gofa
  @Resolver("User")
  type UserResolver struct {}

  @Query("users")
  func getUsers(@Args() args GetUsersArgs) []User {
      // Generate GraphQL query handling
  }
  ```

#### 🟡 3.4 Microservices Decorators `[MEDIUM PRIORITY]`

- [ ] **`@MessagePattern()` transpilation** - Generate message handlers
  ```gofa
  @MessagePattern("user.created")
  func handleUserCreated(@Payload() data UserCreatedEvent) {
      // Generate message handling code
  }
  ```

---

### ✨ Phase 4: Transpiler Optimization

> **Target:** Production-ready transpilation | **Timeline:** 3-4 weeks

#### 🔥 4.1 Code Generation Optimization `[HIGH PRIORITY]`

- [ ] **AST optimization** - Efficient code generation
- [ ] **Template caching** - Faster transpilation
- [ ] **Incremental compilation** - Only transpile changed files
- [ ] **Parallel processing** - Multi-file transpilation
- [ ] **Memory usage optimization** - Large project handling

#### 🔥 4.2 Developer Experience `[HIGH PRIORITY]`

- [ ] **Better error messages** - Clear transpilation errors
- [ ] **Source map generation** - Debug original .gofa files
- [ ] **IDE integration** - VS Code extension for .gofa files
- [ ] **Syntax highlighting** - .gofa file highlighting
- [ ] **Auto-completion** - Decorator auto-completion

#### 🟡 4.3 CLI & Tooling `[MEDIUM PRIORITY]`

- [ ] **Watch mode** - Auto-transpile on file changes
- [ ] **Build integration** - Makefile/Go build integration
- [ ] **Project scaffolding** - Generate boilerplate .gofa files
- [ ] **Migration tools** - Convert existing Go code to .gofa

---

## 🏆 Transpiler Milestones

### 🎯 Milestone 1: Core Decorators Complete (Phase 1)
- ✅ All parameter decorators working
- ✅ Basic HTTP decorators functional
- 🔄 Error handling decorators implemented

### 🎯 Milestone 2: Advanced Patterns (Phase 2)
- ⏳ Middleware decorators fully functional
- ⏳ Validation decorators complete
- ⏳ Complex routing patterns working

### 🎯 Milestone 3: Full Decorator Parity (Phase 3)
- ⏳ All NestJS decorators implemented
- ⏳ Specialized features (WebSocket, GraphQL)
- ⏳ Testing decorators complete

### 🎯 Milestone 4: Production Ready (Phase 4)
- ⏳ Performance optimized
- ⏳ Developer tooling complete
- ⏳ Enterprise-ready transpiler

---

## 📋 Transpiler Implementation Notes

### **Core Responsibilities:**
1. **Parse .gofa files** - AST generation and analysis
2. **Generate .go code** - Template-based code generation
3. **Decorator processing** - Convert decorators to framework calls
4. **Type analysis** - Understand Go types and interfaces
5. **Import management** - Handle package dependencies
6. **Error reporting** - Clear transpilation error messages

### **What Transpiler Does NOT Do:**
- ❌ Runtime error handling (Framework responsibility)
- ❌ HTTP server implementation (Framework responsibility)
- ❌ Dependency injection container (Framework responsibility)
- ❌ Validation logic (Framework responsibility)
- ❌ Authentication/authorization (Framework responsibility)

### **Generated Code Patterns:**
```go
// From: @Get("/users/:id")
// Generates:
router.GET("/users/:id", func(ctx *http.RequestContext) {
    id := ctx.Param("id")
    result := controller.GetUser(id)
    ctx.JSON(200, result)
})
```

---

## 🎯 Priority Focus

**Current Focus**: Transpiler-specific features only
**Next Phase**: Return to framework development after transpiler completion

| Priority | Feature | Type |
|----------|---------|------|
| 1️⃣ | `@Catch()` decorator | Transpiler |
| 2️⃣ | `@HttpCode()` decorator | Transpiler |
| 3️⃣ | Validation decorators | Transpiler |
| 4️⃣ | Middleware decorators | Transpiler |
| 5️⃣ | AST optimization | Transpiler |

---

<div align="center">

### 🔗 Quick Navigation

[Phase 1](#-phase-1-core-decorator-transpilation) • [Phase 2](#️-phase-2-advanced-decorator-patterns) • [Phase 3](#-phase-3-specialized-transpilation) • [Phase 4](#-phase-4-transpiler-optimization)

---

**🔧 Transpiler-focused development roadmap**  
**📦 See [Framework Roadmap](GOFASTA_FRAMEWORK_ROADMAP.md) for runtime features**

</div>