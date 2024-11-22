package ordered_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/uptrace/bun/internal/ordered"
)

func TestBasicFeatures(t *testing.T) {
	m := ordered.NewMap[int, int]()

	// Test initialization
	assert.Equal(t, 0, m.Len(), "expected length 0")

	// Test storing and loading
	m.Store(5, 50)
	m.Store(3, 30)
	m.Store(1, 10)
	m.Store(4, 40)
	m.Store(2, 20)

	assert.Equal(t, 5, m.Len(), "expected length 5 after storing 5 elements")

	val, ok := m.Load(1)
	assert.True(t, ok, "expected key 1 to be present")
	assert.Equal(t, 10, val, "expected value 10 for key 1")

	val, ok = m.Load(2)
	assert.True(t, ok, "expected key 2 to be present")
	assert.Equal(t, 20, val, "expected value 20 for key 2")

	val, ok = m.Load(3)
	assert.True(t, ok, "expected key 3 to be present")
	assert.Equal(t, 30, val, "expected value 30 for key 3")

	val, ok = m.Load(4)
	assert.True(t, ok, "expected key 4 to be present")
	assert.Equal(t, 40, val, "expected value 40 for key 4")

	val, ok = m.Load(5)
	assert.True(t, ok, "expected key 5 to be present")
	assert.Equal(t, 50, val, "expected value 50 for key 5")

	// Test ordering
	expectedKeys := []int{5, 3, 1, 4, 2}
	assert.Equal(t, expectedKeys, m.Keys(), "expected keys to be [5, 3, 1, 4, 2]")

	expectedValues := []int{50, 30, 10, 40, 20}
	assert.Equal(t, expectedValues, m.Values(), "expected values to be [50, 30, 10, 40, 20]")

	// Test deletion
	m.Delete(3)
	assert.Equal(t, 4, m.Len(), "expected length 4 after deleting key 3")
	expectedKeys = []int{5, 1, 4, 2}
	assert.Equal(t, expectedKeys, m.Keys(), "expected keys to be [5, 1, 4, 2]")

	expectedValues = []int{50, 10, 40, 20}
	assert.Equal(t, expectedValues, m.Values(), "expected values to be [50, 10, 40, 20]")

	m.Delete(1)
	assert.Equal(t, 3, m.Len(), "expected length 3 after deleting key 1")
	expectedKeys = []int{5, 4, 2}
	assert.Equal(t, expectedKeys, m.Keys(), "expected keys to be [5, 4, 2]")

	expectedValues = []int{50, 40, 20}
	assert.Equal(t, expectedValues, m.Values(), "expected values to be [50, 40, 20]")

	// Test clearing the map
	m.Clear()
	assert.Equal(t, 0, m.Len(), "expected length 0 after clearing the map")
	assert.Empty(t, m.Keys(), "expected no keys after clearing the map")
	assert.Empty(t, m.Values(), "expected no values after clearing the map")
}
