// Package prom provides observability hooks for the scope library.
//
// [Metrics] is an in-memory counter implementation without external dependencies.
// [Exporter] registers Prometheus counters, gauges, and histograms with a
// [github.com/prometheus/client_golang/prometheus.Registerer].
package prom
