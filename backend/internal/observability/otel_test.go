package observability

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"strings"
	"testing"
)

func TestOtlpMode_EnvDetection(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://collector:4318")
	if !otlpMode() {
		t.Fatal("otlpMode() = false with OTEL_EXPORTER_OTLP_ENDPOINT set")
	}
	os.Unsetenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if otlpMode() {
		t.Fatal("otlpMode() = true without OTEL_EXPORTER_OTLP_ENDPOINT")
	}
}

// fanoutHandler must write each record to every handler that has the level
// enabled, and only to those — the file crash channel and the OTLP pipeline
// must not see each other's filtering decisions.
func TestFanoutHandler_FansOutToEnabledHandlersOnly(t *testing.T) {
	var enabled, disabled bytes.Buffer
	h := fanoutHandler{[]slog.Handler{
		slog.NewJSONHandler(&enabled, &slog.HandlerOptions{Level: slog.LevelInfo}),
		slog.NewJSONHandler(&disabled, &slog.HandlerOptions{Level: slog.LevelWarn}),
	}}
	logger := slog.New(h)

	logger.InfoContext(context.Background(), "post created", "post_id", 7)

	if !strings.Contains(enabled.String(), "post created") {
		t.Errorf("info-level handler missed the record: %q", enabled.String())
	}
	if disabled.String() != "" {
		t.Errorf("warn-only handler received an info record: %q", disabled.String())
	}
}

// Enabled must be true when any wrapped handler is enabled, so the default
// logger never drops a record one of the sinks still wants.
func TestFanoutHandler_EnabledIfAnyHandlerEnabled(t *testing.T) {
	var buf bytes.Buffer
	h := fanoutHandler{[]slog.Handler{
		slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn}),
		slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}),
	}}
	if !h.Enabled(context.Background(), slog.LevelInfo) {
		t.Error("Enabled = false while one handler accepts info level")
	}
	if h.Enabled(context.Background(), slog.LevelDebug-1) {
		t.Error("Enabled = true while no handler accepts below debug")
	}
}
