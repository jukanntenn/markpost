package observability

import (
	"context"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/DeRuina/timberjack"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
)

// metricExportInterval is how often the metric PeriodicReader flushes (to the
// metrics JSONL file, or to the OTLP collector in OTLP mode).
const metricExportInterval = 60 * time.Second

// Providers bundles the OTel providers built at startup so they can be shut
// down cleanly on SIGTERM.
type Providers struct {
	tracerProvider *trace.TracerProvider
	meterProvider  *metric.MeterProvider
	logProvider    *sdklog.LoggerProvider // nil in file mode
	closers        []io.Closer
	otlpMode       bool
}

// otlpMode reports whether the process runs in OTLP mode: exporters and the
// SDK read OTEL_EXPORTER_OTLP_* / OTEL_SERVICE_NAME from the environment
// (OTLP/HTTP + gzip to the collector, bearer-token header included). File
// mode (stdout exporters to timberjack JSONL) stays the default so the
// loadtest/capacity stack is unchanged.
func otlpMode() bool { return os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") != "" }

// otelResource is the shared resource identifying the service in OTLP mode.
func otelResource() *resource.Resource {
	res, err := resource.Merge(
		resource.Default(), // picks up OTEL_SERVICE_NAME / OTEL_RESOURCE_ATTRIBUTES
		resource.NewWithAttributes(semconv.SchemaURL, semconv.ServiceName("markpost")),
	)
	if err != nil {
		return resource.Default()
	}
	return res
}

// Init builds the three-pillar pipeline. In OTLP mode (OTEL_EXPORTER_OTLP_ENDPOINT
// set) all three pillars ship via OTLP/HTTP exporters configured from the
// environment; in file mode the stdout exporters write the timberjack JSONL
// files as before.
func Init(appLogger, tracesLogger, metricsLogger *timberjack.Logger) (*Providers, error) {
	if otlpMode() {
		return initOTLP(appLogger)
	}
	return initFile(appLogger, tracesLogger, metricsLogger)
}

// initOTLP wires otlptracehttp / otlpmetrichttp / otlploghttp exporters; the
// endpoint, headers (bearer token) and gzip compression all come from the
// standard OTEL_EXPORTER_OTLP_* environment variables.
func initOTLP(appLogger *timberjack.Logger) (*Providers, error) {
	ctx := context.Background()
	res := otelResource()

	traceExporter, err := otlptracehttp.New(ctx)
	if err != nil {
		return nil, err
	}
	tracerProvider := trace.NewTracerProvider(
		trace.WithBatcher(traceExporter),
		trace.WithResource(res),
		trace.WithSampler(trace.ParentBased(trace.AlwaysSample())),
	)

	metricExporter, err := otlpmetrichttp.New(ctx)
	if err != nil {
		return nil, err
	}
	meterProvider := metric.NewMeterProvider(
		metric.WithReader(metric.NewPeriodicReader(metricExporter, metric.WithInterval(metricExportInterval))),
		metric.WithResource(res),
	)

	logExporter, err := otlploghttp.New(ctx)
	if err != nil {
		return nil, err
	}
	logProvider := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(logExporter)),
		sdklog.WithResource(res),
	)

	otel.SetTracerProvider(tracerProvider)
	otel.SetMeterProvider(meterProvider)

	return &Providers{
		tracerProvider: tracerProvider,
		meterProvider:  meterProvider,
		logProvider:    logProvider,
		closers:        []io.Closer{appLogger},
		otlpMode:       true,
	}, nil
}

// initFile is the original files-only pipeline (stdout exporters → timberjack).
func initFile(appLogger, tracesLogger, metricsLogger *timberjack.Logger) (*Providers, error) {
	traceExporter, err := stdouttrace.New(stdouttrace.WithWriter(tracesLogger))
	if err != nil {
		return nil, err
	}
	tracerProvider := trace.NewTracerProvider(
		trace.WithBatcher(traceExporter),
		// ParentBased(AlwaysOn): default to sampling every trace. Single
		// service, no cross-service propagation — volume is manageable.
		// Switch to TraceIDRatioBased if QPS grows (observability.md §采样策略).
		trace.WithSampler(trace.ParentBased(trace.AlwaysSample())),
	)

	metricExporter, err := stdoutmetric.New(stdoutmetric.WithWriter(metricsLogger))
	if err != nil {
		return nil, err
	}
	meterProvider := metric.NewMeterProvider(
		metric.WithReader(metric.NewPeriodicReader(metricExporter, metric.WithInterval(metricExportInterval))),
	)

	otel.SetTracerProvider(tracerProvider)
	otel.SetMeterProvider(meterProvider)

	return &Providers{
		tracerProvider: tracerProvider,
		meterProvider:  meterProvider,
		closers:        []io.Closer{appLogger, tracesLogger, metricsLogger},
	}, nil
}

// InstallSlogDefault sets the default slog logger: always the timberjack file
// handler with trace correlation (the crash channel), fanning out to the OTLP
// logs pipeline as well when running in OTLP mode (dual-write).
func (p *Providers) InstallSlogDefault(appLogger *timberjack.Logger) {
	file := NewTraceHandler(appLogger)
	if !p.otlpMode || p.logProvider == nil {
		slog.SetDefault(slog.New(file))
		return
	}
	otlpLogs := otelslog.NewHandler("markpost", otelslog.WithLoggerProvider(p.logProvider))
	slog.SetDefault(slog.New(fanoutHandler{[]slog.Handler{file, otlpLogs}}))
}

// fanoutHandler writes each record to every handler that has it enabled.
type fanoutHandler struct {
	handlers []slog.Handler
}

func (h fanoutHandler) Enabled(ctx context.Context, l slog.Level) bool {
	for _, hd := range h.handlers {
		if hd.Enabled(ctx, l) {
			return true
		}
	}
	return false
}

func (h fanoutHandler) Handle(ctx context.Context, r slog.Record) error {
	var first error
	for _, hd := range h.handlers {
		if !hd.Enabled(ctx, r.Level) {
			continue
		}
		if err := hd.Handle(ctx, r.Clone()); err != nil && first == nil {
			first = err
		}
	}
	return first
}

func (h fanoutHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	next := make([]slog.Handler, len(h.handlers))
	for i, hd := range h.handlers {
		next[i] = hd.WithAttrs(attrs)
	}
	return fanoutHandler{next}
}

func (h fanoutHandler) WithGroup(name string) slog.Handler {
	next := make([]slog.Handler, len(h.handlers))
	for i, hd := range h.handlers {
		next[i] = hd.WithGroup(name)
	}
	return fanoutHandler{next}
}

// Shutdown flushes the trace/metric/log exporters and closes the timberjack
// loggers. It must be called on graceful shutdown so buffered spans/metrics
// are not lost. Errors are returned but shutdown is best-effort: a failing
// exporter flush is logged by the caller and does not block process exit.
func (p *Providers) Shutdown(ctx context.Context) error {
	var first error
	if err := p.tracerProvider.Shutdown(ctx); err != nil {
		first = err
	}
	if err := p.meterProvider.Shutdown(ctx); err != nil && first == nil {
		first = err
	}
	if p.logProvider != nil {
		if err := p.logProvider.Shutdown(ctx); err != nil && first == nil {
			first = err
		}
	}
	for _, c := range p.closers {
		_ = c.Close()
	}
	return first
}
