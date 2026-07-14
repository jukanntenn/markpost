package observability

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/trace"
)

// TestTraceHandler_InjectsTraceID verifies the custom slog handler adds
// trace_id/span_id attrs when a span is active in the context, and omits them
// when no span is present.
func TestTraceHandler_InjectsTraceID(t *testing.T) {
	tp := trace.NewTracerProvider()
	otel.SetTracerProvider(tp)
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })

	var buf bytes.Buffer
	logger := slog.New(NewTraceHandler(&buf))

	ctx, span := tp.Tracer("test").Start(context.Background(), "op")
	defer span.End()

	logger.InfoContext(ctx, "hello", "user_id", 42)

	var record map[string]any
	if err := json.Unmarshal(buf.Bytes(), &record); err != nil {
		t.Fatalf("unmarshal log line: %v\nline: %s", err, buf.String())
	}
	if record["trace_id"] == nil || record["trace_id"] == "" {
		t.Errorf("expected non-empty trace_id, got %v", record["trace_id"])
	}
	if record["span_id"] == nil || record["span_id"] == "" {
		t.Errorf("expected non-empty span_id, got %v", record["span_id"])
	}
	if record["msg"] != "hello" {
		t.Errorf("msg = %v, want hello", record["msg"])
	}
}

// TestTraceHandler_NoSpan verifies that without an active span the record is
// emitted without trace attrs.
func TestTraceHandler_NoSpan(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(NewTraceHandler(&buf))

	logger.Info("plain message")

	var record map[string]any
	if err := json.Unmarshal(buf.Bytes(), &record); err != nil {
		t.Fatalf("unmarshal log line: %v", err)
	}
	if _, ok := record["trace_id"]; ok {
		t.Errorf("did not expect trace_id when no span is active")
	}
	if record["msg"] != "plain message" {
		t.Errorf("msg = %v, want 'plain message'", record["msg"])
	}
}
