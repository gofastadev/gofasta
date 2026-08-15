package apperrors

// Wire codes for the error taxonomy.
//
// A GraphQL error's message is human copy — it gets reworded, translated, and
// tuned for tone. Clients that need to branch on the *kind* of failure (retry
// this, redirect to login for that, highlight a form field for the other) need
// something stable to branch on, and these codes are it. They are the half of
// the contract that does not change.
//
// The scaffold's GraphQL schema documents them by name, so a project generated
// by `gofasta new --graphql` and a client written against its schema agree
// without either side inventing its own strings.
const (
	CodeNotFound             = "NOT_FOUND"
	CodeValidationFailed     = "VALIDATION_FAILED"
	CodeConflict             = "CONFLICT"
	CodeUnauthorized         = "UNAUTHORIZED"
	CodeForbidden            = "FORBIDDEN"
	CodeBadRequest           = "BAD_REQUEST"
	CodePreconditionFailed   = "PRECONDITION_FAILED"
	CodePreconditionRequired = "PRECONDITION_REQUIRED"
	CodeInternal             = "INTERNAL"
)

// Code returns the wire code for an error type.
//
// An unrecognized type reports INTERNAL rather than an empty string: a client
// switching on the code should fall into its "something went wrong" branch, not
// into a branch that matches nothing.
func (t ErrorType) Code() string {
	switch t {
	case NotFound:
		return CodeNotFound
	case Validation:
		return CodeValidationFailed
	case Conflict:
		return CodeConflict
	case Unauthorized:
		return CodeUnauthorized
	case Forbidden:
		return CodeForbidden
	case BadRequest:
		return CodeBadRequest
	case PreconditionFailed:
		return CodePreconditionFailed
	case PreconditionRequired:
		return CodePreconditionRequired
	case Internal:
		return CodeInternal
	default:
		return CodeInternal
	}
}

// String makes ErrorType printable in logs and test failures. It reports the
// wire code, so a log line and the error the client saw name the same thing.
func (t ErrorType) String() string { return t.Code() }
