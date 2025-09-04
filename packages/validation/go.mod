module github.com/healtronlabs/gofasta/packages/validation

go 1.22.5

require (
	github.com/go-playground/locales v0.14.1
	github.com/go-playground/universal-translator v0.18.1
	github.com/go-playground/validator/v10 v10.22.0
	github.com/healtronlabs/gofasta/packages/core v0.0.0
)

require (
	github.com/gabriel-vasile/mimetype v1.4.3 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/stretchr/testify v1.9.0 // indirect
	golang.org/x/crypto v0.30.0 // indirect
	golang.org/x/net v0.32.0 // indirect
	golang.org/x/sys v0.28.0 // indirect
	golang.org/x/text v0.21.0 // indirect
)

replace github.com/healtronlabs/gofasta/packages/core => ../core
