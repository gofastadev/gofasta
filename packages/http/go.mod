module github.com/healtronlabs/gofasta/packages/http

go 1.22.5

require (
	github.com/healtronlabs/gofasta/packages/core v0.0.0
	github.com/gorilla/mux v1.8.1
	github.com/gorilla/websocket v1.5.1
)

replace github.com/healtronlabs/gofasta/packages/core => ../core