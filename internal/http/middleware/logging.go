package middleware

import (
	"context"
	"net/http"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
)

// Logging writes one record per request once the response is ready. It reads
// the route from the otelhttp Labeler, which Route (registered deeper, per
// matched chi route) fills in before this middleware's next.ServeHTTP returns.
func Logging(log logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}

			next.ServeHTTP(sw, r)

			args := []any{
				"method", r.Method,
				"route", routeFromContext(r.Context()),
				"status", sw.status,
				"duration_ms", time.Since(start).Milliseconds(),
			}

			if sw.status >= http.StatusInternalServerError {
				log.ErrorContext(r.Context(), "request handled", args...)
				return
			}

			log.InfoContext(r.Context(), "request handled", args...)
		})
	}
}

func routeFromContext(ctx context.Context) string {
	labeler, ok := otelhttp.LabelerFromContext(ctx)
	if !ok {
		return ""
	}

	for _, attr := range labeler.Get() {
		if attr.Key == semconv.HTTPRouteKey {
			return attr.Value.AsString()
		}
	}

	return ""
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}
