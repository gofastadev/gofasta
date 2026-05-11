package whatsapp

import "errors"

// ErrUnsupported is returned by providers that don't implement an
// optional method (e.g. Twilio's lack of message deletion). Callers
// should test with errors.Is and log + continue rather than treating
// it as a hard failure.
var ErrUnsupported = errors.New("whatsapp: operation not supported by this provider")

// errMissingContentType — returned by uploadMetaMedia when raw bytes
// are passed without a MIME type. Sentinel rather than a fmt.Errorf
// at the call site so the meta_upload helper stays free of fmt
// (avoids errcheck noise) and callers can errors.Is if they care.
var errMissingContentType = errors.New("meta media: ContentType is required when uploading raw bytes")
