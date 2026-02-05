package middleware

import "net/http"

// Middleware is a function that wraps an http.Handler.
type Middleware func(http.Handler) http.Handler

// Chain creates a new middleware chain from the given middlewares.
// Middlewares are applied in the order they are passed, meaning the first
// middleware is the outermost wrapper (first to receive the request,
// last to see the response).
//
// Example:
//
//	chain := middleware.Chain(
//	    middleware.Recovery(logger),
//	    middleware.Logging(logger),
//	)
//	server := &http.Server{Handler: chain(mux)}
func Chain(middlewares ...Middleware) Middleware {
	return func(next http.Handler) http.Handler {
		for i := len(middlewares) - 1; i >= 0; i-- {
			next = middlewares[i](next)
		}
		return next
	}
}
