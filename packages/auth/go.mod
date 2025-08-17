module github.com/healtronlabs/gofasta/packages/auth

go 1.22.5

require (
	github.com/golang-jwt/jwt/v5 v5.2.1
	github.com/healtronlabs/gofasta/packages/core v0.0.0
	golang.org/x/crypto v0.25.0
)

replace github.com/healtronlabs/gofasta/packages/core => ../core
