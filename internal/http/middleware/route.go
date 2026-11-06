package middleware

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	"go.opentelemetry.io/otel/trace"
)

// Route labels the span and otelhttp's own metrics with the matched chi
// route pattern, and tracks http.server.active_requests. It must run after
// chi has resolved routing, so it is registered via
// api.ChiServerOptions.Middlewares rather than wrapping the whole mux: that
// is the only place the route pattern is known, and symmetric labels on an
// up-down counter require knowing it at both increment and decrement.
func Route(activeRequests metric.Int64UpDownCounter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			pattern := chi.RouteContext(r.Context()).RoutePattern()
			routeAttr := semconv.HTTPRoute(pattern)

			if labeler, ok := otelhttp.LabelerFromContext(r.Context()); ok {
				labeler.Add(routeAttr)
			}

			if span := trace.SpanFromContext(r.Context()); span.IsRecording() {
				span.SetName(r.Method + " " + pattern)
				span.SetAttributes(routeAttr)
			}

			attrs := metric.WithAttributes(semconv.HTTPRequestMethodKey.String(r.Method), routeAttr)
			activeRequests.Add(r.Context(), 1, attrs)
			defer activeRequests.Add(r.Context(), -1, attrs)

			next.ServeHTTP(w, r)
		})
	}
}
