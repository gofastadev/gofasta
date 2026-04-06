// Gofasta CLI — the global command-line tool for the gofasta framework.
//
// Install:
//
//	go install github.com/gofastadev/gofasta/cmd/gofasta@latest
//
// Usage:
//
//	gofasta new myapp
//	gofasta g s Product name:string price:float
//	gofasta dev
package main

import "github.com/gofastadev/gofasta/cmd"

func main() {
	cmd.Execute()
}
