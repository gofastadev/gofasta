module github.com/healtronlabs/gofasta/packages/validation

go 1.22.5

require (
	github.com/go-playground/validator/v10 v10.22.0
	github.com/go-playground/locales v0.14.1
	github.com/go-playground/universal-translator v0.18.1
	github.com/healtronlabs/gofasta/packages/core v0.0.0
)

replace github.com/healtronlabs/gofasta/packages/core => ../core
