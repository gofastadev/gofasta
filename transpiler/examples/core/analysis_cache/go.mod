module analysis-cache-example

go 1.24

toolchain go1.24.5

require (
	github.com/healtronlabs/gofasta/transpiler v0.0.0
	golang.org/x/tools v0.28.0
)

require (
	golang.org/x/mod v0.22.0 // indirect
	golang.org/x/sync v0.16.0 // indirect
)

replace github.com/healtronlabs/gofasta/transpiler => ../../..
