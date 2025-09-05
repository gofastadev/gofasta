module github.com/healtronlabs/gofasta/packages/http

go 1.22.5

require (
	github.com/gorilla/mux v1.8.1
	github.com/gorilla/websocket v1.5.1
	github.com/healtronlabs/gofasta/packages/core v0.0.0
)

require golang.org/x/net v0.32.0 // indirect

replace github.com/healtronlabs/gofasta/packages/core => ../core
