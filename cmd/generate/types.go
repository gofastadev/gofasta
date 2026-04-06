package generate

// Field represents a single field in a resource (parsed from "name:type" CLI args).
type Field struct {
	Name      string // PascalCase: ProductName
	JSONName  string // camelCase: productName
	SnakeName string // snake_case: product_name
	GoType    string // Go type: string
	GormType  string // GORM tag: gorm:"not null"
	GQLType   string // GraphQL type: String
	SQLType   string // SQL type: VARCHAR(255) NOT NULL
}

// ScaffoldData holds all computed names and fields for template rendering.
type ScaffoldData struct {
	Name         string  // PascalCase: Product
	LowerName    string  // camelCase: product
	SnakeName    string  // snake_case: product
	PluralName   string  // PascalCase plural: Products
	PluralSnake  string  // snake_case plural: products
	PluralLower  string  // camelCase plural: products
	Fields            []Field
	MigrationNum      string
	IncludeController bool
	IncludeGraphQL    bool
}

// Step is a single unit of work in a generator pipeline.
type Step struct {
	Label string
	Fn    func(ScaffoldData) error
}
