package apperrors

import (
	stderrors "errors"
	"fmt"
	"testing"
)

func TestClassify_MapsConventionalWording(t *testing.T) {
	cases := map[string]ErrorType{
		"authentication required":                Unauthorized,
		"user is unauthenticated":                Unauthorized,
		"invalid credentials":                    Unauthorized,
		"token expired":                          Unauthorized,
		"permission denied":                      Forbidden,
		"you do not have permission to edit":     Forbidden,
		"access denied for this resource":        Forbidden,
		"course not found":                       NotFound,
		"the unit does not exist":                NotFound,
		"no such enrollment":                     NotFound,
		"a course with that name already exists": Conflict,
		"duplicate submission":                   Conflict,
		"validation failed":                      Validation,
		"title is required":                      Validation,
		"name cannot be empty":                   Validation,
	}

	for msg, want := range cases {
		t.Run(msg, func(t *testing.T) {
			if got := Classify(msg).Type; got != want {
				t.Errorf("Classify(%q).Type = %v, want %v", msg, got, want)
			}
		})
	}
}

func TestClassify_UnrecognisedIsInternal(t *testing.T) {
	// The safe default: presenters replace an Internal message with a generic
	// one, so a driver error naming a column or a query never reaches a client.
	err := Classify(`pq: column "archived_at" does not exi`)
	if err.Type != Internal {
		t.Errorf("Type = %v, want Internal", err.Type)
	}
}

func TestClassify_SpecificWordingBeatsGeneral(t *testing.T) {
	// "already enrolled" must not be read as validation just because it also
	// contains no other keyword, and the bare "already" rule must not shadow
	// "already exists" into a different type.
	if got := Classify("learner is already enrolled").Type; got != Conflict {
		t.Errorf("already enrolled = %v, want Conflict", got)
	}
	if got := Classify("a record already exists").Type; got != Conflict {
		t.Errorf("already exists = %v, want Conflict", got)
	}
}

func TestClassify_AuthenticationBeatsAuthorization(t *testing.T) {
	// "unauthorized" appears inside sentences that mean either. The
	// authentication phrasings are listed first so the more specific reading
	// wins; this pins that ordering.
	if got := Classify("authentication required: unauthorized").Type; got != Unauthorized {
		t.Errorf("Type = %v, want Unauthorized", got)
	}
}

func TestClassify_IsCaseInsensitive(t *testing.T) {
	if got := Classify("COURSE NOT FOUND").Type; got != NotFound {
		t.Errorf("Type = %v, want NotFound", got)
	}
}

func TestClassify_PreservesTheOriginalMessage(t *testing.T) {
	const msg = "Course not found"
	if got := Classify(msg).Message; got != msg {
		t.Errorf("Message = %q, want the original casing and wording", got)
	}
}

func TestRegisterClassifier_AddsProjectWording(t *testing.T) {
	t.Cleanup(ResetClassifiers)

	// Wording no framework could guess.
	RegisterClassifier("only the facilitator", func(m string) *AppError {
		return NewForbidden(m, nil)
	})

	if got := Classify("only the facilitator may publish this unit").Type; got != Forbidden {
		t.Errorf("Type = %v, want Forbidden from the registered rule", got)
	}
}

func TestRegisterClassifier_OverridesADefault(t *testing.T) {
	t.Cleanup(ResetClassifiers)

	// Registered rules run first, so a project that disagrees with a default
	// can replace it rather than fork the mechanism.
	if got := Classify("invalid state transition").Type; got != Validation {
		t.Fatalf("precondition: default made it %v, want Validation", got)
	}

	RegisterClassifier("invalid state transition", func(m string) *AppError {
		return NewPreconditionFailed(m, nil)
	})

	if got := Classify("invalid state transition").Type; got != PreconditionFailed {
		t.Errorf("Type = %v, want the registered override", got)
	}
}

func TestRegisterClassifier_IgnoresIncompleteRules(t *testing.T) {
	t.Cleanup(ResetClassifiers)

	// A rule with no phrase would match everything; one with no builder would
	// panic on the first match. Both are dropped rather than armed.
	RegisterClassifier("", func(m string) *AppError { return NewForbidden(m, nil) })
	RegisterClassifier("something", nil)

	if got := Classify("course not found").Type; got != NotFound {
		t.Errorf("Type = %v; an incomplete rule was armed", got)
	}
}

func TestClassifyError_PassesAnAppErrorThrough(t *testing.T) {
	t.Cleanup(ResetClassifiers)

	// Re-classifying would discard the type the producer chose and the details
	// it attached, in favor of a guess made from the message text.
	original := NewPreconditionRequired("payment method required", map[string]string{"field": "card"})

	got := ClassifyError(original)
	if got != original {
		t.Fatalf("ClassifyError returned a different value")
	}
	if got.Details == nil {
		t.Error("details were dropped")
	}
}

func TestClassifyError_UnwrapsToFindAnAppError(t *testing.T) {
	wrapped := fmt.Errorf("loading course: %w", NewNotFound("course not found", nil))

	if got := ClassifyError(wrapped).Type; got != NotFound {
		t.Errorf("Type = %v, want the wrapped AppError's type", got)
	}
}

func TestClassifyError_ClassifiesAPlainError(t *testing.T) {
	if got := ClassifyError(stderrors.New("course not found")).Type; got != NotFound {
		t.Errorf("Type = %v, want NotFound", got)
	}
}

func TestClassifyError_NilIsNil(t *testing.T) {
	if got := ClassifyError(nil); got != nil {
		t.Errorf("ClassifyError(nil) = %v, want nil", got)
	}
}
