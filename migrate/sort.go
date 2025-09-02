package migrate

import (
	"sort"
	"sync"
)

// MigrationSortFunc defines the signature for functions that sort migrations.
type MigrationSortFunc func(ms MigrationSlice)

// sortMutex protects access to the global sort functions.
var sortMutex sync.RWMutex

// Default sort implementations
var defaultAscSort MigrationSortFunc = func(ms MigrationSlice) {
	sort.Slice(ms, func(i, j int) bool {
		return ms[i].Name < ms[j].Name
	})
}

var defaultDescSort MigrationSortFunc = func(ms MigrationSlice) {
	sort.Slice(ms, func(i, j int) bool {
		return ms[i].Name > ms[j].Name
	})
}

// AscSort is the global ascending sort function.
// Default is to sort by migration name in ascending order.
// Can be overridden to use custom sorting logic.
var AscSort MigrationSortFunc = defaultAscSort

// DescSort is the global descending sort function.
// Default is to sort by migration name in descending order.
// Can be overridden to use custom sorting logic.
var DescSort MigrationSortFunc = defaultDescSort

// SafeAscSort applies the current ascending sort function in a thread-safe manner.
func SafeAscSort(ms MigrationSlice) {
	sortMutex.RLock()
	defer sortMutex.RUnlock()
	AscSort(ms)
}

// SafeDescSort applies the current descending sort function in a thread-safe manner.
func SafeDescSort(ms MigrationSlice) {
	sortMutex.RLock()
	defer sortMutex.RUnlock()
	DescSort(ms)
}
