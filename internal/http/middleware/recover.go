// Package middleware holds the HTTP middleware chain: Recover, Tracing, Logging, Metrics.
package middleware

import (
	"context"
	"net/http"
	"runtime/debug"

	"github.com/n1ckerr0r/avito-labs/internal/http/httputil"
)

type logger interface {
	InfoContext(ctx context.Context, msg string, args ...any)
	ErrorContext(ctx context.Context, msg string, args ...any)
}

// Recover must be the outermost middleware: it stops a panic anywhere in the
// chain from crashing the process and answers with 500 internal_error instead.
func Recover(log logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if recovered := recover(); recovered != nil {
					log.ErrorContext(r.Context(), "panic recovered", "error", recovered, "stack", string(debug.Stack()))
					httputil.WriteProblem(w, r, http.StatusInternalServerError,
						httputil.Problem("internal_error", "Internal server error", "An unexpected error occurred", http.StatusInternalServerError))
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
