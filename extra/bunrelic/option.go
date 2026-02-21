package bunrelic

import newrelic "github.com/newrelic/go-agent/v3/newrelic"

// Option configures a QueryHook.
type Option func(h *QueryHook)

// WithDBName sets the database name to report in recorded segments
func WithDBName(name string) Option {
	return func(h *QueryHook) {
		h.baseSegment.DatabaseName = name
	}
}

// WithProduct sets the product to report in recorded segments
func WithProduct(product newrelic.DatastoreProduct) Option {
	return func(h *QueryHook) {
		h.baseSegment.Product = product
	}
}

// WithHost sets the host to report in recorded segments
func WithHost(host string) Option {
	return func(h *QueryHook) {
		h.baseSegment.Host = host
	}
}

// WithPortPathOrId sets the Port/Path/ID to report in recorded segments
func WithPortPathOrId(portPathOrId string) Option {
	return func(h *QueryHook) {
		h.baseSegment.PortPathOrID = portPathOrId
	}
}
