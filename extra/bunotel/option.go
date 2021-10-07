package bunotel

import (
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
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
