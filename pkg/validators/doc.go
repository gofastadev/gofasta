// Package validators wraps go-playground/validator for input validation. It
// provides AppValidator with ValidateStruct() that returns structured error
// DTOs, plus common validators: UUID validation, record existence checks, URL
// validation, and record deletability checks. Projects register their own
// custom validators on top.
package validators
