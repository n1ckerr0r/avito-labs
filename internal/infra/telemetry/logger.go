package telemetry

import (
	"context"
	"log/slog"
	"os"

	"go.opentelemetry.io/otel/trace"
)

// NewLogger builds a JSON slog.Logger that adds trace_id/span_id to any
// record logged within a recording span. levelName is one of debug, info,
// warn, error, validated by config.Load.
func NewLogger(levelName string) *slog.Logger {
	return slog.New(&spanContextHandler{Handler: slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: parseLevel(levelName)})})
}

func parseLevel(name string) slog.Level {
	switch name {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

type spanContextHandler struct {
	slog.Handler
}

func (h *spanContextHandler) Handle(ctx context.Context, record slog.Record) error {
	if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
		record.AddAttrs(
			slog.String("trace_id", sc.TraceID().String()),
			slog.String("span_id", sc.SpanID().String()),
		)
	}

	return h.Handler.Handle(ctx, record)
}

func (h *spanContextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &spanContextHandler{Handler: h.Handler.WithAttrs(attrs)}
}

func (h *spanContextHandler) WithGroup(name string) slog.Handler {
	return &spanContextHandler{Handler: h.Handler.WithGroup(name)}
}
