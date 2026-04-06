package middleware

import (
	"net/http"

	"github.com/gofastadev/gofasta/pkg/config"
	"github.com/unrolled/secure"
)

// SecurityHeaders adds security headers (HSTS, X-Frame-Options, CSP, etc.) to all responses.
func SecurityHeaders(cfg config.SecurityConfig) Middleware {
	s := secure.New(secure.Options{
		STSSeconds:            int64(cfg.HSTSMaxAge),
		STSIncludeSubdomains:  cfg.HSTS,
		FrameDeny:             cfg.FrameDeny,
		ContentTypeNosniff:    cfg.ContentTypeNosniff,
		BrowserXssFilter:      cfg.BrowserXSSFilter,
		ContentSecurityPolicy: cfg.ContentSecurityPolicy,
		ReferrerPolicy:        cfg.ReferrerPolicy,
	})
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if err := s.Process(w, r); err != nil {
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
