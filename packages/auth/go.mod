module github.com/healtronlabs/gofasta/packages/auth

go 1.22.5

require (
	github.com/golang-jwt/jwt/v5 v5.2.1
	github.com/healtronlabs/gofasta/packages/core v0.0.0
	golang.org/x/crypto v0.30.0
)

require golang.org/x/sys v0.28.0 // indirect

replace github.com/healtronlabs/gofasta/packages/core => ../core
