package observability

import (
	"context"
	"io"

	"log/slog"

	"go.opentelemetry.io/otel/trace"
)

// traceHandler is a slog.Handler that wraps a JSON handler and injects the
// active span's trace_id and span_id into every record. This is the ~20-line
// custom handler described in observability.md ("路线 A"): trace↔log
// correlation without the slog-otel dependency. Records written without a
// span in context are emitted unchanged (no trace attrs).
type traceHandler struct {
	inner slog.Handler
}

// NewTraceHandler wraps the given writer with a JSON slog handler that adds
// trace_id/span_id attrs from the context's active span.
func NewTraceHandler(w io.Writer) slog.Handler {
	return &traceHandler{inner: slog.NewJSONHandler(w, nil)}
}

func (h *traceHandler) Handle(ctx context.Context, r slog.Record) error {
	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.IsValid() {
		r.AddAttrs(
			slog.String("trace_id", spanCtx.TraceID().String()),
			slog.String("span_id", spanCtx.SpanID().String()),
		)
	}
	return h.inner.Handle(ctx, r)
}

func (h *traceHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &traceHandler{inner: h.inner.WithAttrs(attrs)}
}

func (h *traceHandler) WithGroup(name string) slog.Handler {
	return &traceHandler{inner: h.inner.WithGroup(name)}
}

func (h *traceHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}
