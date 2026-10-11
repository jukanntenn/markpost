package observability

// gen.go carries the regeneration directive for the metrics inventory
// documentation (docs/metrics.md and its pair): the tools/metricsdoc command
// parses the instrument declarations in this package, the middleware, and the
// cmd/server adapters, and rewrites the generated docs at the repository root.
//
//go:generate go run ../../tools/metricsdoc
