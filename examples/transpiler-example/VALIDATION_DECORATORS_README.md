# 🛡️ Built-in Validation Decorators

The GOFASTA transpiler supports comprehensive built-in validation decorators that automatically generate Go validation code for your DTOs (Data Transfer Objects). These decorators provide type-safe validation with detailed error messages and codes.

## 🎯 Overview

Validation decorators are applied to struct fields using the `validate` tag with `@` prefixed decorator names. The transpiler automatically generates:

- **ValidationError struct**: Standardized error representation
- **ValidationResult struct**: Container for validation results  
- **Validation functions**: Generated `Validate{StructName}()` functions
- **Helper functions**: Regex-based validation helpers
- **Required imports**: Automatic import management

## 📝 Type Validation Decorators

### Basic Type Validation
```gofa
type UserDto struct {
    Name     string  `validate:"@IsString()"`   // Validates string type
    Age      int     `validate:"@IsInt()"`      // Validates integer type
    Score    float64 `validate:"@IsFloat()"`    // Validates floating point
    Active   bool    `validate:"@IsBoolean()"`  // Validates boolean type
    Tags     []string `validate:"@IsArray()"`   // Validates array/slice
    Profile  UserProfile `validate:"@IsObject()"` // Validates object/struct
}
```

### Advanced Type Validation
```gofa
type EventDto struct {
    CreatedAt time.Time `validate:"@IsDate()"`    // Validates date
    UUID      string    `validate:"@IsUUID()"`    // Validates UUID format
    Count     interface{} `validate:"@IsNumber()"` // Validates numeric (int/float)
}
```

## 📧 Format Validation Decorators

### Communication Formats
```gofa
type ContactDto struct {
    Email       string `validate:"@IsEmail()"`       // Email format validation
    Website     string `validate:"@IsURL()"`         // URL format validation
    Phone       string `validate:"@IsPhoneNumber()"` // Phone number format
    IPAddress   string `validate:"@IsIP()"`          // IP address format
}
```

### Text Content Validation
```gofa
type ContentDto struct {
    Title       string `validate:"@IsAlpha()"`        // Letters only
    Slug        string `validate:"@IsAlphanumeric()"` // Letters and numbers
    Code        string `validate:"@IsNumeric()"`      // Numbers only  
    Color       string `validate:"@IsHexColor()"`     // Hex color format
    ISBN        string `validate:"@IsISBN()"`         // ISBN format
    CreditCard  string `validate:"@IsCreditCard()"`   // Credit card format
    Data        string `validate:"@IsBase64()"`       // Base64 encoded
    Config      string `validate:"@IsJSON()"`         // JSON format
}
```

## 🔢 Range & Length Validation Decorators

### Numeric Range Validation
```gofa
type ProductDto struct {
    Price       float64 `validate:"@Min(0.01) @Max(9999.99)"` // Price range
    Stock       int     `validate:"@Min(0)"`                  // Minimum stock
    Rating      float64 `validate:"@Min(0) @Max(5)"`          // Rating range
    Discount    int     `validate:"@Max(100)"`                // Maximum discount
}
```

### String Length Validation  
```gofa
type UserDto struct {
    Username    string `validate:"@Length(3,20)"`      // Exact length range
    Password    string `validate:"@MinLength(8)"`      // Minimum length
    Bio         string `validate:"@MaxLength(500)"`    // Maximum length
    FirstName   string `validate:"@Length(2,30)"`      // Name length range
}
```

### Array Size Validation
```gofa
type OrderDto struct {
    Items       []string `validate:"@ArrayMinSize(1)"`     // At least 1 item
    Categories  []string `validate:"@ArrayMaxSize(10)"`    // At most 10 items
    Tags        []string `validate:"@ArrayMinSize(1) @ArrayMaxSize(5)"` // Range
    Options     []string `validate:"@ArrayNotEmpty()"`     // Not empty array
}
```

## 🔍 Content Validation Decorators

### Presence Validation
```gofa
type RequiredFieldsDto struct {
    Name        string `validate:"@IsNotEmpty()"`   // Cannot be empty string
    Description string `validate:"@IsEmpty()"`     // Must be empty string  
    Email       string `validate:"@IsDefined()"`   // Cannot be nil/undefined
    Phone       string `validate:"@IsOptional()"`  // Skip validation if empty
}
```

### Value Comparison
```gofa
type ComparisonDto struct {
    Status      string `validate:"@Equals(active)"`        // Must equal "active"
    Role        string `validate:"@NotEquals(admin)"`      // Cannot be "admin"
    Category    string `validate:"@IsIn(tech,business)"`   // Must be in list
    Type        string `validate:"@IsNotIn(spam,deleted)"` // Cannot be in list
}
```

### Content Matching
```gofa
type SearchDto struct {
    Title       string `validate:"@Contains(keyword)"`     // Must contain substring
    Content     string `validate:"@NotContains(banned)"`   // Cannot contain substring
}
```

## 🔄 Pattern & Custom Validation Decorators

### Text Pattern Validation
```gofa
type PatternDto struct {
    Code        string `validate:"@Matches(^[A-Z]{3}-[0-9]{4}$)"` // Regex pattern
    Name        string `validate:"@IsLowercase()"`               // Lowercase only
    Title       string `validate:"@IsUppercase()"`               // Uppercase only
}
```

### Nested & Conditional Validation
```gofa
type ComplexDto struct {
    Address     AddressDto `validate:"@ValidateNested()"`    // Validate nested struct
    SecondaryEmail string  `validate:"@ValidateIf(hasSecondaryEmail)"` // Conditional
}
```

### Custom Validation
```gofa
type CustomDto struct {
    CustomField string `validate:"@Custom(myValidator)"` // Custom validation function
}
```

## 🏢 Business Logic Validation Decorators

### Numeric Business Rules
```gofa
type FinancialDto struct {
    Amount      float64 `validate:"@IsPositive()"`     // Must be positive
    Debt        float64 `validate:"@IsNegative()"`     // Must be negative
}
```

### Date Business Rules
```gofa
type EventDto struct {
    Birthday    time.Time `validate:"@IsPastDate()"`   // Must be in the past
    EventDate   time.Time `validate:"@IsFutureDate()"` // Must be in the future
}
```

### Database Validation (Requires Integration)
```gofa
type EntityDto struct {
    Email       string `validate:"@IsUnique(email)"`        // Must be unique in DB
    CategoryID  int    `validate:"@Exists(category,id)"`    // Must exist in DB
}
```

## 🔧 Generated Code Structure

When you use validation decorators, the transpiler generates:

### ValidationError Struct
```go
type ValidationError struct {
    Field   string      `json:"field"`
    Value   interface{} `json:"value"`
    Message string      `json:"message"`
    Code    string      `json:"code"`
}
```

### Validation Functions
```go
func ValidateUserDto(dto *UserDto) []ValidationError {
    var errors []ValidationError
    
    // @IsEmail() validation for Email field
    if !isValidEmail(dto.Email) {
        errors = append(errors, ValidationError{
            Field:   "Email",
            Value:   dto.Email,
            Message: "must be a valid email address",
            Code:    "IS_EMAIL",
        })
    }
    
    // Additional validation rules...
    
    return errors
}
```

### Helper Functions
```go
func isValidEmail(email string) bool {
    emailRegex := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
    matched, _ := regexp.MatchString(emailRegex, email)
    return matched
}
```

## 💡 Usage Examples

### Basic Usage
```go
// Use the generated validation function
userData := UserDto{
    Email:    "invalid-email",
    Username: "user123",
    Age:      17, // Below minimum
}

validationErrors := ValidateUserDto(&userData)
if len(validationErrors) > 0 {
    // Handle validation errors
    for _, err := range validationErrors {
        fmt.Printf("Field: %s, Error: %s\n", err.Field, err.Message)
    }
    return
}

// Process valid data...
```

### HTTP Handler Integration
```go
func CreateUserHandler(w http.ResponseWriter, r *http.Request) {
    var userData UserDto
    if err := json.NewDecoder(r.Body).Decode(&userData); err != nil {
        http.Error(w, "Invalid JSON", http.StatusBadRequest)
        return
    }
    
    // Validate using generated function
    validationErrors := ValidateUserDto(&userData)
    if len(validationErrors) > 0 {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusBadRequest)
        json.NewEncoder(w).Encode(map[string]interface{}{
            "error": "Validation failed",
            "details": validationErrors,
        })
        return
    }
    
    // Process valid user data...
}
```

## 🔗 Error Codes Reference

| Decorator | Generated Code | Message Template |
|-----------|---------------|------------------|
| `@IsEmail()` | `IS_EMAIL` | "must be a valid email address" |
| `@Min(n)` | `MIN_VALUE` | "must be at least {n}" |
| `@Max(n)` | `MAX_VALUE` | "must be at most {n}" |
| `@IsNotEmpty()` | `IS_NOT_EMPTY` | "must not be empty" |
| `@Length(min,max)` | `LENGTH` | "must be between {min} and {max} characters" |
| `@IsArray()` | `IS_ARRAY` | "must be an array" |
| `@ArrayMinSize(n)` | `ARRAY_MIN_SIZE` | "must contain at least {n} item(s)" |
| `@IsPositive()` | `IS_POSITIVE` | "must be a positive number" |
| `@IsURL()` | `IS_URL` | "must be a valid URL" |
| `@IsNumeric()` | `IS_NUMERIC` | "must contain only numbers" |
| `@IsAlphanumeric()` | `IS_ALPHANUMERIC` | "must contain only letters and numbers" |
| `@IsAlpha()` | `IS_ALPHA` | "must contain only letters" |

## ⚠️ Important Notes

1. **Automatic Imports**: Required packages (`strings`, `regexp`, `net/url`) are automatically imported
2. **Type Safety**: Generated validation functions are fully type-safe
3. **Performance**: Uses compiled regex patterns for efficient validation
4. **Extensible**: Easy to add custom validation logic alongside generated code
5. **Standards Compliant**: Follows standard validation patterns and error formats

## 🚀 Next Steps

- Run comprehensive test cases with `go test -v -run TestValidation`
- Check generated validation code in transpiled `.go` files
- Integrate validation functions into your HTTP handlers
- Extend with custom validation logic as needed