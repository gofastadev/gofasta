package mailer

import (
	"net/http"
	"time"
)

// defaultHTTPTimeout bounds a call to a provider's HTTP API.
//
// The API senders used to share a zero-value http.Client, which has no
// timeout: a provider that accepted the connection and then stopped answering
// held the calling goroutine open for as long as the process lived. Thirty
// seconds is long enough for a large attachment on a slow link and short
// enough that a stuck send is a delayed email rather than a leak.
const defaultHTTPTimeout = 30 * time.Second

// SenderOption customizes an API-backed sender.
type SenderOption func(*senderOptions)

type senderOptions struct {
	httpClient *http.Client
}

// WithHTTPClient supplies the http.Client an API-backed sender uses.
//
// The reason to pass one is transport policy the mailer has no business
// deciding: retry and circuit-breaking against a specific upstream, a proxy, a
// custom TLS configuration, or a recording client in tests. A client supplied
// here is used as given — including its timeout, so set one.
//
// Ignored by the SMTP sender, which speaks to a socket rather than over HTTP.
func WithHTTPClient(client *http.Client) SenderOption {
	return func(o *senderOptions) {
		if client != nil {
			o.httpClient = client
		}
	}
}

// resolveSenderOptions applies opts over the defaults.
func resolveSenderOptions(opts []SenderOption) senderOptions {
	resolved := senderOptions{
		httpClient: &http.Client{Timeout: defaultHTTPTimeout},
	}
	for _, opt := range opts {
		opt(&resolved)
	}
	return resolved
}
