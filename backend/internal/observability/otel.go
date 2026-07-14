package observability

import (
	"context"
	"io"
	"time"

	"github.com/DeRuina/timberjack"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/trace"
)

// metricExportInterval is how often the metric PeriodicReader flushes to the
// metrics JSONL file.
const metricExportInterval = 60 * time.Second

// Providers bundles the OTel tracer and meter providers built at startup so
// they can be shut down cleanly on SIGTERM.
type Providers struct {
	tracerProvider *trace.TracerProvider
	meterProvider  *metric.MeterProvider
	traceExporter  *stdouttrace.Exporter
	metricExporter metric.Exporter
	closers        []io.Closer
}

// Init builds the three-pillar pipeline:
//  1. stdouttrace exporter writing to the traces timberjack → sdktrace
//     TracerProvider (ParentBased(AlwaysOn) sampling, batched).
//  2. stdoutmetric exporter writing to the metrics timberjack → sdkmetric
//     MeterProvider (60s PeriodicReader).
//  3. The globals are set via otel.SetTracerProvider / SetMeterProvider so
//     otelgin and business instrumentation pick them up.
//
// appLogger is accepted for symmetry but the slog default logger is wired
// separately (see SetDefaultLogger); it is closed on Shutdown.
func Init(appLogger, tracesLogger, metricsLogger *timberjack.Logger) (*Providers, error) {
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
		traceExporter:  traceExporter,
		metricExporter: metricExporter,
		closers:        []io.Closer{appLogger, tracesLogger, metricsLogger},
	}, nil
}

// Shutdown flushes the trace/metric exporters and closes the timberjack
// loggers. It must be called on graceful shutdown so buffered spans/metrics are
// not lost. Errors are returned but shutdown is best-effort: a failing exporter
// flush is logged by the caller and does not block process exit.
func (p *Providers) Shutdown(ctx context.Context) error {
	var first error
	if err := p.tracerProvider.Shutdown(ctx); err != nil {
		first = err
	}
	if err := p.meterProvider.Shutdown(ctx); err != nil && first == nil {
		first = err
	}
	for _, c := range p.closers {
		_ = c.Close()
	}
	return first
}
