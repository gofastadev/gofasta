module github.com/healtronlabs/gofasta/tests/integration

go 1.24

require github.com/healtronlabs/gofasta/tools/transpiler v0.0.0

require golang.org/x/sync v0.16.0 // indirect

replace github.com/healtronlabs/gofasta/tools/transpiler => ../../tools/transpiler
