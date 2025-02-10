package bunotel

import (
	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
	"go.opentelemetry.io/otel/trace"
)

type Option func(h *QueryHook)

// WithAttributes configures attributes that are used to create a span.
func WithAttributes(attrs ...attribute.KeyValue) Option {
	return func(h *QueryHook) {
		h.attrs = append(h.attrs, attrs...)
	}
}

// WithDBName configures a db.name attribute.
func WithDBName(name string) Option {
	return func(h *QueryHook) {
		h.attrs = append(h.attrs, semconv.DBNameKey.String(name))
	}
}

// WithFormattedQueries enables formatting of the query that is added
// as the statement attribute to the trace.
// This means that all placeholders and arguments will be filled first
// and the query will contain all information as sent to the database.
func WithFormattedQueries(format bool) Option {
	return func(h *QueryHook) {
		h.formatQueries = format
	}
}

// WithSpanNameFormatter takes a function that determines the span name
// for a given query event.
func WithSpanNameFormatter(f func(*bun.QueryEvent) string) Option {
	return func(h *QueryHook) {
		h.spanNameFormatter = f
	}
}

// WithTracerProvider returns an Option to use the TracerProvider when
// creating a Tracer.
func WithTracerProvider(tp trace.TracerProvider) Option {
	return func(h *QueryHook) {
		if tp != nil {
			h.tracer = tp.Tracer("github.com/uptrace/bun")
		}
	}
}

// WithMeterProvider returns an Option to use the MeterProvider when
// creating a Meter.
func WithMeterProvider(mp metric.MeterProvider) Option {
	return func(h *QueryHook) {
		if mp != nil {
			h.meter = mp.Meter("github.com/uptrace/bun")
		}
	}
}
