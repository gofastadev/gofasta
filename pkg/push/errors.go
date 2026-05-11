package push

import "errors"

// ErrUnsupported is returned by providers that don't implement an
// optional method (e.g. the noop sender, or providers without
// server-side topic subscription). Callers should test with errors.Is
// and log + continue rather than treating it as a hard failure.
var ErrUnsupported = errors.New("push: operation not supported by this provider")

// ErrNotConfigured is returned by the noop sender's Send* methods —
// it surfaces a missing PUSH_PROVIDER as a *runtime* error rather than
// a silent no-op, so a misconfig doesn't pretend everything's fine.
var ErrNotConfigured = errors.New("push: no provider configured (set PUSH_PROVIDER)")
