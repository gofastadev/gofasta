# 🔧 GoFasta Transpiler Development Roadmap

> **Goal:** Code Generation & Decorator Transpilation for Go Framework

[![Progress](https://img.shields.io/badge/Progress-70--80%25-yellow)](https://github.com/healtronlabs/gofasta)
[![Status](https://img.shields.io/badge/Status-Active%20Development-green)](https://github.com/healtronlabs/gofasta)
[![Timeline](https://img.shields.io/badge/Timeline-3--4%20months-blue)](https://github.com/healtronlabs/gofasta)

## 📊 Current Status

| ✅**Completed**  | ❌**Pending**              |
| ---------------------- | -------------------------------- |
| Basic REST API transpilation | Advanced decorator patterns |
| Parameter decorators   | WebSocket decorators |
| Controller/Service generation | GraphQL decorators |
| Dependency injection wiring | Testing decorators |
| **Validation decorators** | Route versioning |
| Middleware decorators | Microservices decorators |

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
- [X] **`@UseInterceptors()` transpilation** - Generate interceptor chains
  ```gofa
  @UseInterceptors(LoggingInterceptor, CacheInterceptor)
  @Get("/data")
  func getData() {
      // Generate interceptor pipeline
  }
  ```
- [X] **`@UsePipes()` transpilation** - Generate pipe validation
  ```gofa
  @UsePipes(ValidationPipe, TransformPipe)
  @Post("/users")
  func createUser(@Body() data CreateUserDto) {
      // Generate pipe processing
  }
  ```

#### 🔥 2.2 Validation Decorators `[HIGH PRIORITY]`

- [🔄] **Built-in validation decorator transpilation (PARTIALLY COMPLETE)**
  ```gofa
  @Injectable()
  type CreateUserDto struct {
      @IsEmail()
      Email    string
      
      @Min(18)
      @Max(120)
      Age      int
      
      @IsNotEmpty()
      @Length(2,50)
      Name     string
      
      @IsArray()
      @ArrayMinSize(1)
      Tags     []string
      
      @ValidateNested()
      Profile  UserProfile
  }
  ```

##### **🎯 Complete Built-in Validation Decorators List**

**📝 Type Validation Decorators**
- [X] `@IsString()` - Validate value is string type
- [X] `@IsNumber()` - Validate value is numeric (int, float, etc.)
- [X] `@IsInt()` - Validate value is integer
- [X] `@IsFloat()` - Validate value is floating point
- [X] `@IsBoolean()` - Validate value is boolean
- [X] `@IsArray()` - Validate value is array/slice
- [X] `@IsDate()` - Validate value is valid date
- [X] `@IsUUID()` - Validate value is valid UUID format

**📧 Format Validation Decorators**
- [X] `@IsEmail()` - Validate email format
- [X] `@IsURL()` - Validate URL format  
- [X] `@IsIP()` - Validate IP address format
- [X] `@IsJSON()` - Validate JSON format
- [X] `@IsAlpha()` - Validate contains only letters
- [X] `@IsAlphanumeric()` - Validate contains only letters and numbers
- [X] `@IsNumeric()` - Validate contains only numbers
- [X] `@IsHexColor()` - Validate hex color format
- [X] `@IsPhoneNumber()` - Validate phone number format
- [X] `@IsCreditCard()` - Validate credit card number format
- [X] `@IsISBN()` - Validate ISBN format
- [X] `@IsBase64()` - Validate base64 encoded string

**🔢 Range & Length Validation Decorators**  
- [X] `@Min(value)` - Validate minimum value (numbers) or length (strings/arrays)
- [X] `@Max(value)` - Validate maximum value (numbers) or length (strings/arrays)
- [X] `@Length(min, max)` - Validate string/array length range
- [X] `@MinLength(length)` - Validate minimum string/array length
- [X] `@MaxLength(length)` - Validate maximum string/array length
- [X] `@ArrayMinSize(size)` - Validate minimum array size
- [X] `@ArrayMaxSize(size)` - Validate maximum array size
- [X] `@ArrayNotEmpty()` - Validate array is not empty

**🔍 Content Validation Decorators**
- [X] `@IsNotEmpty()` - Validate value is not empty
- [X] `@IsEmpty()` - Validate value is empty  
- [X] `@IsOptional()` - Mark field as optional (skip validation if nil/empty)
- [X] `@IsDefined()` - Validate value is defined (not nil)
- [X] `@NotEquals(value)` - Validate value does not equal specified value
- [X] `@Equals(value)` - Validate value equals specified value
- [X] `@Contains(substring)` - Validate string contains substring
- [X] `@NotContains(substring)` - Validate string does not contain substring
- [X] `@IsIn(values...)` - Validate value is in allowed list
- [X] `@IsNotIn(values...)` - Validate value is not in forbidden list

**🔄 Pattern & Custom Validation Decorators**
- [X] `@Matches(pattern)` - Validate against regex pattern
- [X] `@IsLowercase()` - Validate string is lowercase
- [X] `@IsUppercase()` - Validate string is uppercase
- [X] `@ValidateNested()` - Validate nested object/struct
- [⚠️] `@ValidateIf(condition)` - Conditional validation (architecture limitation - needs redesign)
- [X] `@Custom(validatorFunc)` - Custom validation function

**🏢 Business Logic Validation Decorators**
- [X] `@IsPositive()` - Validate number is positive
- [ ] `@IsNegative()` - Validate number is negative
- [ ] `@IsPastDate()` - Validate date is in the past
- [ ] `@IsFutureDate()` - Validate date is in the future
- [ ] `@IsUnique(field)` - Validate field value is unique (requires DB check)
- [ ] `@Exists(entity, field)` - Validate referenced entity exists (requires DB check)

**📋 Generated Validation Code Structure**
```go
// Generated validation function for CreateUserDto
func ValidateCreateUserDto(dto *CreateUserDto) []ValidationError {
    var errors []ValidationError
    
    // @IsEmail() validation for Email field
    if !isValidEmail(dto.Email) {
        errors = append(errors, ValidationError{
            Field:   "Email",
            Value:   dto.Email,
            Message: "Email must be a valid email address",
            Code:    "IS_EMAIL",
        })
    }
    
    // @Min(18) @Max(120) validation for Age field
    if dto.Age < 18 {
        errors = append(errors, ValidationError{
            Field:   "Age", 
            Value:   dto.Age,
            Message: "Age must be at least 18",
            Code:    "MIN_VALUE",
        })
    }
    if dto.Age > 120 {
        errors = append(errors, ValidationError{
            Field:   "Age",
            Value:   dto.Age, 
            Message: "Age must be at most 120",
            Code:    "MAX_VALUE",
        })
    }
    
    // @IsNotEmpty() @Length(2,50) validation for Name field
    if strings.TrimSpace(dto.Name) == "" {
        errors = append(errors, ValidationError{
            Field:   "Name",
            Value:   dto.Name,
            Message: "Name must not be empty",
            Code:    "IS_NOT_EMPTY",
        })
    }
    if len(dto.Name) < 2 || len(dto.Name) > 50 {
        errors = append(errors, ValidationError{
            Field:   "Name",
            Value:   dto.Name,
            Message: "Name must be between 2 and 50 characters",
            Code:    "LENGTH",
        })
    }
    
    // @IsArray() @ArrayMinSize(1) validation for Tags field
    if dto.Tags == nil {
        errors = append(errors, ValidationError{
            Field:   "Tags",
            Value:   dto.Tags,
            Message: "Tags must be an array",
            Code:    "IS_ARRAY", 
        })
    } else if len(dto.Tags) < 1 {
        errors = append(errors, ValidationError{
            Field:   "Tags",
            Value:   dto.Tags,
            Message: "Tags must contain at least 1 item",
            Code:    "ARRAY_MIN_SIZE",
        })
    }
    
    // @ValidateNested() validation for Profile field
    if nestedErrors := ValidateUserProfile(&dto.Profile); len(nestedErrors) > 0 {
        for _, err := range nestedErrors {
            err.Field = "Profile." + err.Field
            errors = append(errors, err)
        }
    }
    
    return errors
}

// ValidationError represents a validation error
type ValidationError struct {
    Field   string      `json:"field"`
    Value   interface{} `json:"value"`
    Message string      `json:"message"`
    Code    string      `json:"code"`
}

// ValidationResult represents the result of validation
type ValidationResult struct {
    IsValid bool              `json:"isValid"`
    Errors  []ValidationError `json:"errors,omitempty"`
}
```

**🔄 Implementation Strategy - PARTIALLY COMPLETED**
1. ✅ **Parse validation decorators** from field-level decorators (not tags)
2. ✅ **Generate validation functions** for each DTO struct  
3. 🔄 **Generate helper validation functions** (isValidEmail, isValidURL, isAlpha, isAlphanumeric, isNumeric, isValidUUID - **only 6 helpers implemented**)
4. ✅ **Generate ValidationError struct** and related types
5. ✅ **Integrate with HTTP parameter extraction** (@Body decorator)
6. ❌ **Support nested validation** with @ValidateNested() - **NOT IMPLEMENTED**
7. ❌ **Generate conditional validation** logic for @IsOptional() - **NOT IMPLEMENTED**
8. ✅ **Field-level decorator syntax** for better developer experience

**📊 Current Implementation Status:**
- **~15 out of 60+ validation decorators** actually implemented
- **Core validation infrastructure** complete (ValidationError, parsing, generation)
- **Field-level decorators** working correctly 
- **Basic validation types** working (email, URL, min/max, length, array checks)
- **Missing implementations**: Most format validations, content validations, pattern validations, business logic validations

**⚠️ Still Needed:**
- ~45 more validation decorators to complete the full specification
- Pattern matching (@Matches, @IsLowercase, @IsUppercase)
- Content validation (@IsEmpty, @IsOptional, @IsDefined, @Contains, etc.)
- Advanced format validation (@IsIP, @IsJSON, @IsPhoneNumber, etc.)
- Business logic validation (@IsNegative, @IsPastDate, @IsFutureDate, etc.)
- Nested validation (@ValidateNested)
- Conditional validation (@ValidateIf)

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
- ✅ Middleware decorators fully functional
- 🔄 **Validation decorators partially complete (~25% done)**
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

| Priority | Feature | Type | Status |
|----------|---------|------|--------|
| 1️⃣ | `@Catch()` decorator | Transpiler | ✅ Complete |
| 2️⃣ | `@HttpCode()` decorator | Transpiler | ✅ Complete |
| 3️⃣ | **Validation decorators** | Transpiler | 🔄 **~25% Complete** |
| 4️⃣ | Middleware decorators | Transpiler | ✅ Complete |
| 5️⃣ | AST optimization | Transpiler | ⏳ Pending |

---

<div align="center">

### 🔗 Quick Navigation

[Phase 1](#-phase-1-core-decorator-transpilation) • [Phase 2](#️-phase-2-advanced-decorator-patterns) • [Phase 3](#-phase-3-specialized-transpilation) • [Phase 4](#-phase-4-transpiler-optimization)

---

**🔧 Transpiler-focused development roadmap**  
**📦 See [Framework Roadmap](GOFASTA_FRAMEWORK_ROADMAP.md) for runtime features**

</div>