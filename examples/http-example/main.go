package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("🚀 Gofasta HTTP Examples")
	fmt.Println()
	fmt.Println("This directory contains three different approaches to HTTP server development:")
	fmt.Println()
	fmt.Println("1. 📝 Basic Approach (Comment Decorators)")
	fmt.Println("   go run -tags=basic examples/http-example/.")
	fmt.Println("   - Uses comment-based decorators like @Controller, @Get")
	fmt.Println("   - Good for learning the basics")
	fmt.Println()
	fmt.Println("2. 💬 Comment Decorators Approach")
	fmt.Println("   go run -tags=comments examples/http-example/.")
	fmt.Println("   - Advanced comment-based decorator system")
	fmt.Println("   - Complex routing and middleware composition")
	fmt.Println()
	fmt.Println("3. 🎯 Programmatic Approach (Recommended)")
	fmt.Println("   go run -tags=programmatic examples/http-example/.")
	fmt.Println("   - Modern fluent API for decorator registration")
	fmt.Println("   - Type-safe and production-ready")
	fmt.Println()
	fmt.Println("Choose an approach by running one of the commands above!")
	fmt.Println()
	fmt.Println("For more details, see: examples/http-example/README.md")

	// Check if a build tag was provided
	if len(os.Args) > 1 {
		fmt.Printf("\n❌ No approach selected. Use build tags to run specific examples.\n")
		os.Exit(1)
	}
}