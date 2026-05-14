package middleware

import (
	"net/http"
	"strconv"
	"strings"
)

type CORSOptions struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	AllowCredentials bool
	MaxAgeSeconds    int
}

func CORS(opts CORSOptions) Middleware {
	allowedOriginSet := make(map[string]struct{}, len(opts.AllowedOrigins))
	allowAnyOrigin := false
	for _, o := range opts.AllowedOrigins {
		o = strings.TrimSpace(o)
		if o == "" {
			continue
		}
		if o == "*" {
			allowAnyOrigin = true
		}
		allowedOriginSet[o] = struct{}{}
	}

	allowedMethods := strings.Join(nonEmptyTrim(opts.AllowedMethods), ", ")
	allowedHeaders := strings.Join(nonEmptyTrim(opts.AllowedHeaders), ", ")
	maxAge := opts.MaxAgeSeconds
	if maxAge < 0 {
		maxAge = 0
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" {
				// NOTE: If credentials are enabled, we must echo a concrete origin (no "*").
				if opts.AllowCredentials {
					if allowAnyOrigin || isAllowedOrigin(origin, allowedOriginSet) {
						w.Header().Set("Access-Control-Allow-Origin", origin)
						w.Header().Add("Vary", "Origin")
						w.Header().Set("Access-Control-Allow-Credentials", "true")
					}
				} else {
					if allowAnyOrigin {
						w.Header().Set("Access-Control-Allow-Origin", "*")
					} else if isAllowedOrigin(origin, allowedOriginSet) {
						w.Header().Set("Access-Control-Allow-Origin", origin)
						w.Header().Add("Vary", "Origin")
					}
				}

				if allowedMethods != "" {
					w.Header().Set("Access-Control-Allow-Methods", allowedMethods)
				}
				if allowedHeaders != "" {
					w.Header().Set("Access-Control-Allow-Headers", allowedHeaders)
				}
				if maxAge > 0 {
					w.Header().Set("Access-Control-Max-Age", strconv.Itoa(maxAge))
				}
			}

			// Preflight request.
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func isAllowedOrigin(origin string, allowed map[string]struct{}) bool {
	_, ok := allowed[origin]
	return ok
}

func nonEmptyTrim(in []string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		out = append(out, s)
	}
	return out
}

