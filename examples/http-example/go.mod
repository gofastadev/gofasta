module github.com/healtronlabs/gofasta/examples/http-example

go 1.22.5

toolchain go1.24.5

replace github.com/healtronlabs/gofasta/packages/core => ../../packages/core

replace github.com/healtronlabs/gofasta/packages/http => ../../packages/http

require (
	github.com/healtronlabs/gofasta/packages/core v0.0.0
	github.com/healtronlabs/gofasta/packages/http v0.0.0-00010101000000-000000000000
)

require (
	github.com/gorilla/mux v1.8.1 // indirect
	github.com/gorilla/websocket v1.5.1 // indirect
	golang.org/x/net v0.17.0 // indirect
)
