// Package observability wires the three observability pillars (logs, traces,
// metrics) to the local filesystem as JSONL files, per specs/backend/observability.md.
// No external services (Jaeger/Prometheus/Loki/OTLP collector) are used.
package observability

import (
	"path/filepath"

	"github.com/DeRuina/timberjack"
)

const (
	// rollingMaxSizeMB is the per-file size cap before a mid-day rotation kicks
	// in (the daily 00:00 rotation is the primary schedule).
	rollingMaxSizeMB = 100
	// rollingMaxBackups keeps roughly two weeks of rotated files.
	rollingMaxBackups = 14
	// rollingMaxAge deletes files older than 30 days (whichever of MaxBackups /
	// MaxAge is stricter binds).
	rollingMaxAge = 30
	// rollingBackupTimeFormat uses millisecond precision so a size-triggered
	// second rotation within one day cannot collide with the 00:00 one.
	rollingBackupTimeFormat = "2006-01-02T15-04-05.000"
)

// rotateAtMidnight is the daily rotation schedule (00:00 local).
var rotateAtMidnight = []string{"00:00"}

// newTimberjack builds a rolling JSONL logger at dir/<name>-*.jsonl with the
// shared rolling policy (daily 00:00 rotation, 100 MB size cap, 14 backups /
// 30 days retention, zstd compression of rotated files). The returned *Logger
// implements io.Writer and is used both as the slog handler sink and as the
// OTel stdout exporter writer.
func newTimberjack(dir, name string) *timberjack.Logger {
	return &timberjack.Logger{
		Filename:         filepath.Join(dir, name+".jsonl"),
		MaxSize:          rollingMaxSizeMB,
		MaxBackups:       rollingMaxBackups,
		MaxAge:           rollingMaxAge,
		Compression:      "zstd",
		RotateAt:         rotateAtMidnight,
		BackupTimeFormat: rollingBackupTimeFormat,
	}
}

// NewAppLogger builds the business/access/error log writer (app-*.jsonl).
func NewAppLogger(dir string) *timberjack.Logger { return newTimberjack(dir, "app") }

// NewTracesLogger builds the OTel span writer (traces-*.jsonl).
func NewTracesLogger(dir string) *timberjack.Logger { return newTimberjack(dir, "traces") }

// NewMetricsLogger builds the OTel metric writer (metrics-*.jsonl).
func NewMetricsLogger(dir string) *timberjack.Logger { return newTimberjack(dir, "metrics") }
