package pgdriver

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWithChannelOverflowHandler(t *testing.T) {
	// Create a test handler
	testHandler := func(n Notification) {}

	// Create a channel instance
	c := &channel{}

	// Apply the option
	opt := WithChannelOverflowHandler(testHandler)
	opt(c)

	// Verify the handler was set correctly
	assert.NotNil(t, c.overflowHandler, "overflow handler should be set")
	assert.Equal(t,
		reflect.ValueOf(testHandler).Pointer(),
		reflect.ValueOf(c.overflowHandler).Pointer(),
		"overflow handler should match the provided handler",
	)
}
